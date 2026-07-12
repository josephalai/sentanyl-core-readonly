// Package billing owns the Stripe REST calls for PLATFORM billing — charging
// tenants for Sentanyl itself on the platform Stripe account. This is distinct
// from the per-tenant Connect/own-sales keys tenants configure for selling to
// their own customers (marketing-service/internal/checkout).
package billing

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// stripePost sends a form-encoded POST to the Stripe API and decodes the
// response, surfacing Stripe error messages. Same hand-rolled style as
// marketing-service/internal/checkout — no SDK.
func stripePost(secretKey, path string, form url.Values, out interface{}) error {
	req, err := http.NewRequest("POST", "https://api.stripe.com"+path, strings.NewReader(form.Encode()))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.SetBasicAuth(secretKey, "")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("stripe API request failed: %w", err)
	}
	defer resp.Body.Close()

	var envelope struct {
		Error *struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	dec := json.NewDecoder(resp.Body)
	raw := json.RawMessage{}
	if err := dec.Decode(&raw); err != nil {
		return fmt.Errorf("failed to decode stripe response: %w", err)
	}
	if err := json.Unmarshal(raw, &envelope); err != nil {
		return fmt.Errorf("failed to decode stripe response: %w", err)
	}
	if envelope.Error != nil {
		return fmt.Errorf("stripe error: %s", envelope.Error.Message)
	}
	if out != nil {
		if err := json.Unmarshal(raw, out); err != nil {
			return fmt.Errorf("failed to decode stripe response: %w", err)
		}
	}
	return nil
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
func CreateSubscriptionCheckoutSession(secretKey, priceID, customerID, tenantIDHex, planTier, successURL, cancelURL string, trialEnd *time.Time) (string, error) {
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
	if err := stripePost(secretKey, "/v1/checkout/sessions", form, &out); err != nil {
		return "", err
	}
	if out.URL == "" {
		return "", fmt.Errorf("stripe returned no checkout URL")
	}
	return out.URL, nil
}

// stripeGet sends a GET to the Stripe API, surfacing Stripe error messages.
func stripeGet(secretKey, path string, out interface{}) error {
	req, err := http.NewRequest("GET", "https://api.stripe.com"+path, nil)
	if err != nil {
		return err
	}
	req.SetBasicAuth(secretKey, "")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("stripe API request failed: %w", err)
	}
	defer resp.Body.Close()

	var envelope struct {
		Error *struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	raw := json.RawMessage{}
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return fmt.Errorf("failed to decode stripe response: %w", err)
	}
	if err := json.Unmarshal(raw, &envelope); err != nil {
		return fmt.Errorf("failed to decode stripe response: %w", err)
	}
	if envelope.Error != nil {
		return fmt.Errorf("stripe error: %s", envelope.Error.Message)
	}
	if out != nil {
		if err := json.Unmarshal(raw, out); err != nil {
			return fmt.Errorf("failed to decode stripe response: %w", err)
		}
	}
	return nil
}

// SubscriptionItem is the single line item on a platform subscription.
type SubscriptionItem struct {
	ItemID  string
	PriceID string
	Status  string
}

// GetSubscriptionItem fetches the subscription's first item — platform
// subscriptions always have exactly one (the plan Price).
func GetSubscriptionItem(secretKey, subscriptionID string) (SubscriptionItem, error) {
	var out struct {
		Status string `json:"status"`
		Items  struct {
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
		ItemID:  out.Items.Data[0].ID,
		PriceID: out.Items.Data[0].Price.ID,
		Status:  out.Status,
	}, nil
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
