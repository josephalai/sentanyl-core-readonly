// Package billing owns the Stripe REST calls for PLATFORM billing — charging
// tenants for Sentanyl itself on the platform Stripe account. This is distinct
// from the per-tenant Connect/own-sales keys tenants configure for selling to
// their own customers (marketing-service/internal/checkout).
package billing

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

// stripeClient is the shared HTTP client for platform Stripe calls (BILL-006):
// a bounded timeout so a hung Stripe connection can't wedge a request
// goroutine forever.
var stripeClient = &http.Client{Timeout: 25 * time.Second}

const (
	stripeMaxRetries  = 3
	stripeMaxBodyRead = 4 << 20 // 4 MiB cap on response bodies
)

// StripeError is a structured Stripe API error (BILL-006): callers can branch
// on Type/Code instead of string-matching a message.
type StripeError struct {
	StatusCode int
	Type       string
	Code       string
	Message    string
	RequestID  string
}

func (e *StripeError) Error() string {
	return fmt.Sprintf("stripe error [%d %s/%s]: %s (request %s)", e.StatusCode, e.Type, e.Code, e.Message, e.RequestID)
}

// newIdempotencyKey mints a random key so a retried mutation can never
// double-charge or double-create at Stripe.
func newIdempotencyKey() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return "sntl_" + hex.EncodeToString(b)
}

// stripeDo performs one Stripe request with retry classification (BILL-006):
// network errors and 429/5xx are retried with backoff; 4xx are returned as a
// structured StripeError immediately. A stable idempotency key is reused
// across retries of the same POST so retries are safe.
func stripeDo(method, secretKey, path string, form url.Values, idempotencyKey string, out interface{}) error {
	base := os.Getenv("STRIPE_API_BASE")
	if base == "" {
		base = "https://api.stripe.com"
	}
	return stripeDoBase(strings.TrimRight(base, "/"), method, secretKey, path, form, idempotencyKey, out)
}

// stripeDoBase is stripeDo with an overridable base URL (test seam).
func stripeDoBase(base, method, secretKey, path string, form url.Values, idempotencyKey string, out interface{}) error {
	var lastErr error
	for attempt := 0; attempt < stripeMaxRetries; attempt++ {
		if attempt > 0 {
			time.Sleep(time.Duration(attempt*attempt) * 250 * time.Millisecond)
		}
		var body io.Reader
		if form != nil {
			body = strings.NewReader(form.Encode())
		}
		req, err := http.NewRequest(method, base+path, body)
		if err != nil {
			return err
		}
		req.SetBasicAuth(secretKey, "")
		if form != nil {
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		}
		if idempotencyKey != "" {
			req.Header.Set("Idempotency-Key", idempotencyKey)
		}
		resp, err := stripeClient.Do(req)
		if err != nil {
			lastErr = fmt.Errorf("stripe API request failed: %w", err)
			continue // network error — retry
		}
		raw, rerr := io.ReadAll(io.LimitReader(resp.Body, stripeMaxBodyRead))
		reqID := resp.Header.Get("Request-Id")
		resp.Body.Close()
		if rerr != nil {
			lastErr = fmt.Errorf("failed to read stripe response: %w", rerr)
			continue
		}
		if resp.StatusCode == 429 || resp.StatusCode >= 500 {
			lastErr = &StripeError{StatusCode: resp.StatusCode, Message: "stripe transient error", RequestID: reqID}
			continue // retryable
		}
		var envelope struct {
			Error *struct {
				Type    string `json:"type"`
				Code    string `json:"code"`
				Message string `json:"message"`
			} `json:"error"`
		}
		if err := json.Unmarshal(raw, &envelope); err != nil {
			return fmt.Errorf("failed to decode stripe response: %w", err)
		}
		if envelope.Error != nil {
			return &StripeError{
				StatusCode: resp.StatusCode, Type: envelope.Error.Type,
				Code: envelope.Error.Code, Message: envelope.Error.Message, RequestID: reqID,
			}
		}
		if out != nil {
			if err := json.Unmarshal(raw, out); err != nil {
				return fmt.Errorf("failed to decode stripe response: %w", err)
			}
		}
		return nil
	}
	return lastErr
}

// stripePost sends a form-encoded POST. A random idempotency key makes a
// retried create safe. Callers needing a caller-stable key use stripePostIdem.
func stripePost(secretKey, path string, form url.Values, out interface{}) error {
	return stripeDo("POST", secretKey, path, form, newIdempotencyKey(), out)
}

// stripePostIdem sends a POST with a caller-supplied idempotency key — used
// when the CALLER controls uniqueness (BILL-008 checkout: one session per
// tenant+plan intent even across duplicate submits).
func stripePostIdem(secretKey, path string, form url.Values, idempotencyKey string, out interface{}) error {
	return stripeDo("POST", secretKey, path, form, idempotencyKey, out)
}

// AsStripeError extracts a *StripeError from err, if present.
func AsStripeError(err error) (*StripeError, bool) {
	var se *StripeError
	if errors.As(err, &se) {
		return se, true
	}
	return nil, false
}

// CreateCustomer creates a platform Stripe Customer for a tenant.
func CreateCustomer(secretKey, email, businessName, tenantIDHex string) (string, error) {
	form := url.Values{}
	form.Set("email", email)
	if businessName != "" {
		form.Set("name", businessName)
	}
	form.Set("metadata[tenant_id]", tenantIDHex)

	var out struct {
		ID string `json:"id"`
	}
	if err := stripePost(secretKey, "/v1/customers", form, &out); err != nil {
		return "", err
	}
	if out.ID == "" {
		return "", fmt.Errorf("stripe returned no customer id")
	}
	return out.ID, nil
}

// minTrialLead is Stripe's minimum for subscription_data[trial_end]: at least
// 48h in the future. Below that we omit the trial and billing starts at
// checkout — acceptable, the trial was about to expire anyway.
const minTrialLead = 48 * time.Hour

// CreateSubscriptionCheckoutSession creates a subscription-mode Checkout
// Session for the platform plan, preserving the tenant's remaining trial.
func CreateSubscriptionCheckoutSession(secretKey, priceID, customerID, tenantIDHex, planTier, successURL, cancelURL, idempotencyKey string, trialEnd *time.Time) (string, error) {
	form := url.Values{}
	form.Set("mode", "subscription")
	form.Set("customer", customerID)
	form.Set("line_items[0][price]", priceID)
	form.Set("line_items[0][quantity]", "1")
	form.Set("client_reference_id", tenantIDHex)
	form.Set("subscription_data[metadata][tenant_id]", tenantIDHex)
	if planTier != "" {
		form.Set("subscription_data[metadata][plan_tier]", planTier)
	}
	form.Set("success_url", successURL)
	form.Set("cancel_url", cancelURL)
	if trialEnd != nil && time.Until(*trialEnd) > minTrialLead {
		form.Set("subscription_data[trial_end]", strconv.FormatInt(trialEnd.Unix(), 10))
	}

	var out struct {
		URL string `json:"url"`
	}
	if idempotencyKey == "" {
		idempotencyKey = newIdempotencyKey()
	}
	if err := stripePostIdem(secretKey, "/v1/checkout/sessions", form, idempotencyKey, &out); err != nil {
		return "", err
	}
	if out.URL == "" {
		return "", fmt.Errorf("stripe returned no checkout URL")
	}
	return out.URL, nil
}

// stripeGet sends a GET to the Stripe API (retryable, bounded, structured
// errors via stripeDo).
func stripeGet(secretKey, path string, out interface{}) error {
	return stripeDo("GET", secretKey, path, nil, "", out)
}

// SubscriptionItem is the single line item on a platform subscription.
type SubscriptionItem struct {
	ItemID            string
	PriceID           string
	Status            string
	CurrentPeriodEnd  time.Time
	CancelAtPeriodEnd bool
}

// GetSubscriptionItem fetches the subscription's first item — platform
// subscriptions always have exactly one (the plan Price).
func GetSubscriptionItem(secretKey, subscriptionID string) (SubscriptionItem, error) {
	var out struct {
		Status            string `json:"status"`
		CurrentPeriodEnd  int64  `json:"current_period_end"`
		CancelAtPeriodEnd bool   `json:"cancel_at_period_end"`
		Items             struct {
			Data []struct {
				ID    string `json:"id"`
				Price struct {
					ID string `json:"id"`
				} `json:"price"`
			} `json:"data"`
		} `json:"items"`
	}
	if err := stripeGet(secretKey, "/v1/subscriptions/"+subscriptionID, &out); err != nil {
		return SubscriptionItem{}, err
	}
	if len(out.Items.Data) == 0 {
		return SubscriptionItem{}, fmt.Errorf("subscription %s has no items", subscriptionID)
	}
	return SubscriptionItem{
		ItemID:            out.Items.Data[0].ID,
		PriceID:           out.Items.Data[0].Price.ID,
		Status:            out.Status,
		CurrentPeriodEnd:  time.Unix(out.CurrentPeriodEnd, 0).UTC(),
		CancelAtPeriodEnd: out.CancelAtPeriodEnd,
	}, nil
}

// SetSubscriptionCancelAtPeriodEnd schedules or reverses end-of-period
// cancellation without changing access before the paid-through timestamp.
func SetSubscriptionCancelAtPeriodEnd(secretKey, subscriptionID string, cancel bool) error {
	form := url.Values{}
	form.Set("cancel_at_period_end", strconv.FormatBool(cancel))
	return stripePost(secretKey, "/v1/subscriptions/"+subscriptionID, form, nil)
}

// UpdateSubscriptionPrice swaps the subscription's item to a new Price with
// proration — the Stripe mechanics of a tier upgrade/downgrade.
func UpdateSubscriptionPrice(secretKey, subscriptionID, itemID, newPriceID string) error {
	form := url.Values{}
	form.Set("items[0][id]", itemID)
	form.Set("items[0][price]", newPriceID)
	form.Set("proration_behavior", "create_prorations")
	return stripePost(secretKey, "/v1/subscriptions/"+subscriptionID, form, nil)
}

// CreatePortalSession creates a Stripe Billing Portal session so the tenant
// can manage cards, invoices, and cancellation on Stripe-hosted pages.
func CreatePortalSession(secretKey, customerID, returnURL string) (string, error) {
	form := url.Values{}
	form.Set("customer", customerID)
	form.Set("return_url", returnURL)

	var out struct {
		URL string `json:"url"`
	}
	if err := stripePost(secretKey, "/v1/billing_portal/sessions", form, &out); err != nil {
		return "", err
	}
	if out.URL == "" {
		return "", fmt.Errorf("stripe returned no portal URL")
	}
	return out.URL, nil
}
