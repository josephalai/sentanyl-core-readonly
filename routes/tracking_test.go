package routes

import (
	"encoding/base64"
	"strings"
	"testing"
	"time"
)

func TestSignedTrackingTokenRejectsTamper(t *testing.T) {
	t.Setenv("TRACKING_TOKEN_SECRET", "0123456789abcdef0123456789abcdef")
	now := time.Date(2026, 7, 15, 0, 0, 0, 0, time.UTC)
	token := encodeTrackingTokenAt("https://example.com/path", "user_1", "send_1", now)
	if !strings.HasPrefix(token, "trk1.") {
		t.Fatalf("token = %q", token)
	}
	url, user, send, ok := decodeTrackingToken(token)
	if !ok || url != "https://example.com/path" || user != "user_1" || send != "send_1" {
		t.Fatalf("decode = %q %q %q %v", url, user, send, ok)
	}
	tampered := token[:len(token)-1] + "A"
	if _, _, _, ok := decodeTrackingToken(tampered); ok {
		t.Fatal("tampered token verified")
	}
}

func TestTrackingTokensRejectUnsafeRedirects(t *testing.T) {
	t.Setenv("TRACKING_TOKEN_SECRET", "0123456789abcdef0123456789abcdef")
	if token := encodeTrackingToken("javascript:alert(1)", "user_1", ""); token != "" {
		t.Fatalf("unsafe signed token = %q", token)
	}
	legacy := base64.URLEncoding.EncodeToString([]byte("javascript:alert(1)|user_1"))
	if _, _, _, ok := decodeLegacyTrackingToken(legacy); ok {
		t.Fatal("unsafe legacy redirect accepted")
	}
}

func TestRewriteLinksUsesVersionedSignedTrackingRoute(t *testing.T) {
	t.Setenv("TRACKING_TOKEN_SECRET", "0123456789abcdef0123456789abcdef")
	got := RewriteLinksForTracking(`<a href="https://example.com">go</a>`, "user_1", "https://sentanyl.test", "send_1")
	if !strings.Contains(got, "/api/v1/tracking/click/trk1.") {
		t.Fatalf("rewritten link = %s", got)
	}
}
