package billing

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sync/atomic"
	"testing"
)

// BILL-006: transient 5xx is retried; a 4xx is a structured StripeError,
// returned immediately (no retry).
func TestStripeRetryClassification(t *testing.T) {
	var calls int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := atomic.AddInt32(&calls, 1)
		if n < 3 {
			w.WriteHeader(503)
			w.Write([]byte(`{}`))
			return
		}
		w.WriteHeader(200)
		w.Write([]byte(`{"id":"cus_ok"}`))
	}))
	defer srv.Close()
	// Point the client at the fixture by overriding the base via a custom do.
	var out struct{ ID string `json:"id"` }
	err := stripeDoBase(srv.URL, "POST", "sk", "/v1/customers", url.Values{}, "k", &out)
	if err != nil || out.ID != "cus_ok" {
		t.Fatalf("retry did not recover: err=%v out=%+v (calls=%d)", err, out, calls)
	}
	if calls != 3 {
		t.Fatalf("expected 3 attempts, got %d", calls)
	}
}

func TestStripeStructuredError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Request-Id", "req_123")
		w.WriteHeader(402)
		w.Write([]byte(`{"error":{"type":"card_error","code":"card_declined","message":"declined"}}`))
	}))
	defer srv.Close()
	err := stripeDoBase(srv.URL, "POST", "sk", "/v1/x", url.Values{}, "k", nil)
	var se *StripeError
	if !errors.As(err, &se) || se.Code != "card_declined" || se.StatusCode != 402 || se.RequestID != "req_123" {
		t.Fatalf("expected structured StripeError, got %v", err)
	}
}

// TestIdempotencyKeyForwarded proves the caller-stable key reaches Stripe.
func TestIdempotencyKeyForwarded(t *testing.T) {
	var gotKey string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotKey = r.Header.Get("Idempotency-Key")
		w.Write([]byte(`{}`))
	}))
	defer srv.Close()
	_ = stripeDoBase(srv.URL, "POST", "sk", "/v1/x", url.Values{}, "stable-key-1", nil)
	if gotKey != "stable-key-1" {
		t.Fatalf("idempotency key not forwarded: %q", gotKey)
	}
}
