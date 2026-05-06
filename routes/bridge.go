package routes

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"time"
)

// ServiceBridge is an HTTP client bridge that the compiler/hydrator will use
// to talk to other macro-services (lms-service, marketing-service).
type ServiceBridge struct {
	LMSBaseURL       string
	MarketingBaseURL string
	client           *http.Client
}

// NewServiceBridge creates a bridge with the given base URLs.
func NewServiceBridge(lmsBaseURL, marketingBaseURL string) *ServiceBridge {
	return &ServiceBridge{
		LMSBaseURL:       lmsBaseURL,
		MarketingBaseURL: marketingBaseURL,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// HydrateLMS sends data to the LMS service for hydration.
func (b *ServiceBridge) HydrateLMS(data []byte) error {
	url := b.LMSBaseURL + "/internal/hydrate-lms"
	resp, err := b.client.Post(url, "application/json", bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("lms hydrate request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("lms hydrate failed (status %d): %s", resp.StatusCode, string(body))
	}
	return nil
}

// HydrateFunnel sends data to the marketing/funnel service for hydration.
func (b *ServiceBridge) HydrateFunnel(data []byte) error {
	url := b.MarketingBaseURL + "/internal/hydrate-funnel"
	resp, err := b.client.Post(url, "application/json", bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("funnel hydrate request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("funnel hydrate failed (status %d): %s", resp.StatusCode, string(body))
	}
	return nil
}

// HydrateGraph sends a full compiled-graph payload to marketing-service's
// /internal/hydrate-graph endpoint. Body shape must match HydrateGraphRequest
// (snake_case top-level slices). Marketing-service upserts each entity by _id
// or (tenant_id, public_id) so re-running the same compile is idempotent.
// Returns the marketing-service response body so callers can surface counts.
func (b *ServiceBridge) HydrateGraph(data []byte) ([]byte, error) {
	url := b.MarketingBaseURL + "/internal/hydrate-graph"
	resp, err := b.client.Post(url, "application/json", bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("graph hydrate request failed: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		return body, fmt.Errorf("graph hydrate failed (status %d): %s", resp.StatusCode, string(body))
	}
	return body, nil
}
