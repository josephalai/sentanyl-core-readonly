package routes

import (
	"errors"
	"net/http"

	"github.com/josephalai/sentanyl/core-service/internal/sidecar"

	"github.com/gin-gonic/gin"
)

// sidecarClient is the singleton PowerMTA client used by every Sidecar*
// helper below. SetSidecarClient is called once from cmd/main.go so the
// route registration signature stays unchanged. When the env var
// POWERMTA_SIDECAR_URL is unset, the client returns ErrSidecarUnconfigured
// for every method and the helpers below translate that into a clear
// "sidecar not configured" error rather than fake success.
var sidecarClient *sidecar.Client

// SetSidecarClient injects the deliverability sidecar HTTP client.
func SetSidecarClient(c *sidecar.Client) { sidecarClient = c }

// requireSidecar returns the configured client or a wrapped error suitable
// for handlers to map onto HTTP 503.
func requireSidecar() (*sidecar.Client, error) {
	if sidecarClient == nil || !sidecarClient.Configured() {
		return nil, sidecar.ErrSidecarUnconfigured
	}
	return sidecarClient, nil
}

// ServerIP is the PowerMTA server's public IP, used in SPF record instructions.
const ServerIP = "5.78.200.152"

// ── helpers ─────────────────────────────────────────────────────────────────

// ── Sidecar wrappers ────────────────────────────────────────────────────────
// Thin shims around sidecar.Client preserving the historical function
// signatures used by the tenant sending-domain handlers. ErrSidecarUnconfigured propagates from the
// client when POWERMTA_SIDECAR_URL is unset, and handleSidecarErr translates
// that into a 503 instead of pretending the call succeeded.

type SidecarResponse = sidecar.AddDomainResponse
type SidecarHealth = sidecar.HealthResponse

func SidecarAddDomain(domain, selector, privPEM string) (*SidecarResponse, error) {
	c, err := requireSidecar()
	if err != nil {
		return nil, err
	}
	return c.AddDomain(domain, selector, privPEM)
}

func SidecarDeleteDomain(domain string) error {
	c, err := requireSidecar()
	if err != nil {
		return err
	}
	return c.DeleteDomain(domain)
}

func SidecarTestSend(domain, to, from, subject string) ([]byte, error) {
	c, err := requireSidecar()
	if err != nil {
		return nil, err
	}
	return c.TestSend(domain, to, from, subject)
}

func SidecarGetHealth() (*SidecarHealth, error) {
	c, err := requireSidecar()
	if err != nil {
		return nil, err
	}
	return c.Health()
}

func SidecarGetStats(domain, since string) ([]byte, error) {
	c, err := requireSidecar()
	if err != nil {
		return nil, err
	}
	return c.Stats(domain, since)
}

func SidecarGetQueueDepth() ([]byte, error) {
	c, err := requireSidecar()
	if err != nil {
		return nil, err
	}
	return c.QueueDepth()
}

func SidecarGetReputation(domain string) ([]byte, error) {
	c, err := requireSidecar()
	if err != nil {
		return nil, err
	}
	return c.Reputation(domain)
}

func SidecarGetWarming(domain string) ([]byte, error) {
	c, err := requireSidecar()
	if err != nil {
		return nil, err
	}
	return c.Warming(domain)
}

func SidecarGetBounces(domain, since string) ([]byte, error) {
	c, err := requireSidecar()
	if err != nil {
		return nil, err
	}
	return c.Bounces(domain, since)
}

func SidecarPauseDomain(domain string) error {
	c, err := requireSidecar()
	if err != nil {
		return err
	}
	return c.PauseDomain(domain)
}

func SidecarResumeDomain(domain string) error {
	c, err := requireSidecar()
	if err != nil {
		return err
	}
	return c.ResumeDomain(domain)
}

// handleSidecarErr writes 503 when the sidecar is unconfigured and 502
// otherwise. The legacy "warning saved anyway" path in HandleAddDomain is
// preserved for back-compat.
func handleSidecarErr(c *gin.Context, err error, fallbackMsg string) {
	if errors.Is(err, sidecar.ErrSidecarUnconfigured) {
		c.JSON(http.StatusServiceUnavailable, gin.H{"status": "error", "error": "deliverability sidecar not configured"})
		return
	}
	handleReturnError(c, errors.New(fallbackMsg+": "+err.Error()), http.StatusBadGateway)
}

// ── HTTP helpers ────────────────────────────────────────────────────────────

func handleInvalidBind(c *gin.Context) {
	c.JSON(http.StatusBadRequest, gin.H{"status": "error", "error": "invalid request body"})
}

func handleReturnError(c *gin.Context, err error, code int) {
	c.JSON(code, gin.H{"status": "error", "error": err.Error()})
}

func handleReturnNotFoundError(c *gin.Context, err error) {
	c.JSON(http.StatusNotFound, gin.H{"status": "error", "error": err.Error()})
}

func handleReturnUpdateError(c *gin.Context, err error) {
	c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "error": err.Error()})
}
