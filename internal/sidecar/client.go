// Package sidecar is the HTTP client for the PowerMTA deliverability
// sidecar. The previous stubs in core-service/routes/domains.go returned
// canned data so the API surface lied about what was actually wired up.
//
// This client makes the truth observable: when POWERMTA_SIDECAR_URL is
// unset, every method returns ErrSidecarUnconfigured, which the route
// handlers translate to HTTP 503. When the URL is set, calls are real
// HTTP requests with a 10s timeout and an optional bearer token.
//
// The wire contract documented at docs/handoff/sidecar-contract.md is
// the source of truth for request/response shapes.
package sidecar

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"time"
)

// ErrSidecarUnconfigured indicates POWERMTA_SIDECAR_URL is not set.
// Handlers should map this to 503 Service Unavailable.
var ErrSidecarUnconfigured = errors.New("powermta sidecar not configured")

// Client talks to the PowerMTA sidecar over HTTP.
type Client struct {
	BaseURL string
	Token   string
	http    *http.Client
}

// New constructs a Client. If POWERMTA_SIDECAR_URL is empty, the
// returned Client is non-nil but every method returns
// ErrSidecarUnconfigured. POWERMTA_SIDECAR_TOKEN is optional bearer auth.
func New() *Client {
	return &Client{
		BaseURL: os.Getenv("POWERMTA_SIDECAR_URL"),
		Token:   os.Getenv("POWERMTA_SIDECAR_TOKEN"),
		http:    &http.Client{Timeout: 10 * time.Second},
	}
}

// NewWithBaseURL builds a client against an explicit URL — useful for
// httptest in unit tests.
func NewWithBaseURL(baseURL, token string) *Client {
	return &Client{
		BaseURL: baseURL,
		Token:   token,
		http:    &http.Client{Timeout: 10 * time.Second},
	}
}

// Configured reports whether the client will issue real network calls.
func (c *Client) Configured() bool {
	return c != nil && c.BaseURL != ""
}

// AddDomainResponse mirrors the documented sidecar reply.
type AddDomainResponse struct {
	VMTA string `json:"vmta"`
}

func (c *Client) AddDomain(domain, selector, dkimPrivatePEM string) (*AddDomainResponse, error) {
	if !c.Configured() {
		return nil, ErrSidecarUnconfigured
	}
	body := map[string]string{
		"domain":               domain,
		"selector":             selector,
		"dkim_private_key_pem": dkimPrivatePEM,
	}
	var resp AddDomainResponse
	if err := c.do("POST", "/domains", body, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (c *Client) DeleteDomain(domain string) error {
	if !c.Configured() {
		return ErrSidecarUnconfigured
	}
	return c.do("DELETE", "/domains/"+url.PathEscape(domain), nil, nil)
}

func (c *Client) TestSend(domain, to, from, subject string) ([]byte, error) {
	if !c.Configured() {
		return nil, ErrSidecarUnconfigured
	}
	return c.doRaw("POST", "/domains/"+url.PathEscape(domain)+"/test-send", map[string]string{
		"to":      to,
		"from":    from,
		"subject": subject,
	})
}

// HealthResponse mirrors the documented sidecar health reply.
type HealthResponse struct {
	AccountingLogExists bool `json:"accounting_log_exists"`
}

func (c *Client) Health() (*HealthResponse, error) {
	if !c.Configured() {
		return nil, ErrSidecarUnconfigured
	}
	var resp HealthResponse
	if err := c.do("GET", "/health", nil, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (c *Client) Stats(domain, since string) ([]byte, error) {
	if !c.Configured() {
		return nil, ErrSidecarUnconfigured
	}
	return c.doRaw("GET", fmt.Sprintf("/stats/%s?since=%s", url.PathEscape(domain), url.QueryEscape(since)), nil)
}

func (c *Client) QueueDepth() ([]byte, error) {
	if !c.Configured() {
		return nil, ErrSidecarUnconfigured
	}
	return c.doRaw("GET", "/queue", nil)
}

func (c *Client) Reputation(domain string) ([]byte, error) {
	if !c.Configured() {
		return nil, ErrSidecarUnconfigured
	}
	return c.doRaw("GET", "/reputation/"+url.PathEscape(domain), nil)
}

func (c *Client) Warming(domain string) ([]byte, error) {
	if !c.Configured() {
		return nil, ErrSidecarUnconfigured
	}
	return c.doRaw("GET", "/warming/"+url.PathEscape(domain), nil)
}

func (c *Client) Bounces(domain, since string) ([]byte, error) {
	if !c.Configured() {
		return nil, ErrSidecarUnconfigured
	}
	return c.doRaw("GET", fmt.Sprintf("/bounces/%s?since=%s", url.PathEscape(domain), url.QueryEscape(since)), nil)
}

func (c *Client) PauseDomain(domain string) error {
	if !c.Configured() {
		return ErrSidecarUnconfigured
	}
	return c.do("POST", "/queue/"+url.PathEscape(domain)+"/pause", nil, nil)
}

func (c *Client) ResumeDomain(domain string) error {
	if !c.Configured() {
		return ErrSidecarUnconfigured
	}
	return c.do("POST", "/queue/"+url.PathEscape(domain)+"/resume", nil, nil)
}

// do issues a JSON request and decodes the response into out (if non-nil).
func (c *Client) do(method, path string, body, out interface{}) error {
	raw, err := c.doRaw(method, path, body)
	if err != nil {
		return err
	}
	if out == nil || len(raw) == 0 {
		return nil
	}
	return json.Unmarshal(raw, out)
}

// doRaw issues a request and returns the response body. It accepts any
// JSON-serializable value (or nil) for the request body.
func (c *Client) doRaw(method, path string, body interface{}) ([]byte, error) {
	var reqBody io.Reader
	if body != nil {
		buf, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("sidecar marshal: %w", err)
		}
		reqBody = bytes.NewReader(buf)
	}
	req, err := http.NewRequest(method, c.BaseURL+path, reqBody)
	if err != nil {
		return nil, fmt.Errorf("sidecar new request: %w", err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if c.Token != "" {
		req.Header.Set("Authorization", "Bearer "+c.Token)
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("sidecar request: %w", err)
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		return respBody, fmt.Errorf("sidecar %s %s: status %d: %s", method, path, resp.StatusCode, string(respBody))
	}
	return respBody, nil
}
