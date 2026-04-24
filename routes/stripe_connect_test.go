package routes

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"gopkg.in/mgo.v2/bson"
)

func TestStripeConnectState_RoundTrip(t *testing.T) {
	tenantID := bson.NewObjectId()
	state, err := storeStripeConnectState(tenantID)
	if err != nil {
		t.Fatalf("store: %v", err)
	}
	if state == "" {
		t.Fatal("empty state")
	}
	got, ok := consumeStripeConnectState(state)
	if !ok {
		t.Fatal("consume: not found")
	}
	if got != tenantID {
		t.Fatalf("tenantID mismatch: got %s want %s", got.Hex(), tenantID.Hex())
	}
	// Second consume must fail — one-shot.
	if _, ok := consumeStripeConnectState(state); ok {
		t.Fatal("state was not consumed (should be one-shot)")
	}
}

func TestStripeConnectState_Expired(t *testing.T) {
	tenantID := bson.NewObjectId()
	state, err := storeStripeConnectState(tenantID)
	if err != nil {
		t.Fatalf("store: %v", err)
	}
	// Force expiry.
	stripeConnectMu.Lock()
	entry := stripeConnectStates[state]
	entry.expires = time.Now().Add(-time.Minute)
	stripeConnectStates[state] = entry
	stripeConnectMu.Unlock()

	if _, ok := consumeStripeConnectState(state); ok {
		t.Fatal("expired state should not consume")
	}
}

func TestStripeConnectState_UnknownState(t *testing.T) {
	if _, ok := consumeStripeConnectState("nope"); ok {
		t.Fatal("unknown state should not consume")
	}
}

func TestExchangeStripeConnectCode_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			t.Errorf("parse form: %v", err)
		}
		if r.FormValue("grant_type") != "authorization_code" {
			t.Errorf("grant_type = %q", r.FormValue("grant_type"))
		}
		if r.FormValue("code") != "abc123" {
			t.Errorf("code = %q", r.FormValue("code"))
		}
		if r.FormValue("client_secret") != "sk_test_platform" {
			t.Errorf("client_secret = %q", r.FormValue("client_secret"))
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"access_token":"sk_acct_1","refresh_token":"rt_1","stripe_user_id":"acct_123","scope":"read_write"}`))
	}))
	defer srv.Close()

	tr, err := exchangeStripeConnectCodeAt(srv.URL, "sk_test_platform", "abc123")
	if err != nil {
		t.Fatalf("exchange: %v", err)
	}
	if tr.StripeUserID != "acct_123" || tr.AccessToken != "sk_acct_1" || tr.RefreshToken != "rt_1" {
		t.Fatalf("unexpected token response: %+v", tr)
	}
}

func TestExchangeStripeConnectCode_StripeError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"error":"invalid_grant","error_description":"code expired"}`))
	}))
	defer srv.Close()

	_, err := exchangeStripeConnectCodeAt(srv.URL, "sk_test_platform", "bad")
	if err == nil {
		t.Fatal("expected error on stripe error response")
	}
}
