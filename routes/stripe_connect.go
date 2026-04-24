package routes

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"gopkg.in/mgo.v2/bson"

	"github.com/josephalai/sentanyl/pkg/auth"
	"github.com/josephalai/sentanyl/pkg/db"
	"github.com/josephalai/sentanyl/pkg/models"
)

// Stripe Connect OAuth flow.
//
// Flow:
//   1. Tenant dashboard calls GET /api/tenant/stripe/connect (authed) to get an
//      authorize URL containing a CSRF state token that server-side maps to the
//      tenant. Frontend sets window.location to that URL.
//   2. Stripe redirects the user to GET /api/tenant/stripe/oauth/callback with
//      ?code=...&state=... (public route — auth is via the state token).
//   3. We POST the code to Stripe's token endpoint with the platform secret
//      key, receive stripe_user_id + access/refresh tokens, and persist them
//      onto the Tenant doc.
//   4. Browser is redirected back to /settings?stripe_connected=1.
//
// Dev/test mode: set STRIPE_CONNECT_DEV_MODE=1. The initiate endpoint returns
// a local callback URL with a fake code; the callback short-circuits the
// token exchange and writes acct_dev_<tenantSuffix> onto the tenant. This
// lets you click through the whole flow without registering a real Stripe
// Connect application.

type stripeConnectState struct {
	tenantID bson.ObjectId
	expires  time.Time
}

var (
	stripeConnectStates = map[string]stripeConnectState{}
	stripeConnectMu     sync.Mutex
)

const stripeConnectStateTTL = 10 * time.Minute

func stripeConnectDevMode() bool {
	return os.Getenv("STRIPE_CONNECT_DEV_MODE") == "1"
}

func storeStripeConnectState(tenantID bson.ObjectId) (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	state := hex.EncodeToString(buf)

	stripeConnectMu.Lock()
	defer stripeConnectMu.Unlock()
	// Opportunistic sweep of expired entries so the map doesn't grow forever.
	now := time.Now()
	for k, v := range stripeConnectStates {
		if now.After(v.expires) {
			delete(stripeConnectStates, k)
		}
	}
	stripeConnectStates[state] = stripeConnectState{
		tenantID: tenantID,
		expires:  now.Add(stripeConnectStateTTL),
	}
	return state, nil
}

func consumeStripeConnectState(state string) (bson.ObjectId, bool) {
	stripeConnectMu.Lock()
	defer stripeConnectMu.Unlock()
	entry, ok := stripeConnectStates[state]
	if !ok {
		return "", false
	}
	delete(stripeConnectStates, state)
	if time.Now().After(entry.expires) {
		return "", false
	}
	return entry.tenantID, true
}

// HandleStripeConnectInitiate returns the Stripe OAuth authorize URL for the
// authenticated tenant. Callers navigate the browser to the returned URL.
func HandleStripeConnectInitiate(c *gin.Context) {
	tenantID := auth.GetTenantObjectID(c)
	if tenantID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	state, err := storeStripeConnectState(tenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create state"})
		return
	}

	if stripeConnectDevMode() {
		// Short-circuit: point the browser straight at our own callback with
		// a synthetic code. The callback recognizes DEV_MODE and skips the
		// real token exchange.
		devURL := fmt.Sprintf("%s/api/tenant/stripe/oauth/callback?code=dev&state=%s",
			publicAPIBase(), url.QueryEscape(state))
		c.JSON(http.StatusOK, gin.H{
			"authorize_url": devURL,
			"dev_mode":      true,
		})
		return
	}

	clientID := os.Getenv("STRIPE_CONNECT_CLIENT_ID")
	if clientID == "" {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "stripe connect is not configured on this platform (set STRIPE_CONNECT_CLIENT_ID or STRIPE_CONNECT_DEV_MODE=1 to test)",
		})
		return
	}

	authorizeURL := fmt.Sprintf(
		"https://connect.stripe.com/oauth/authorize?response_type=code&client_id=%s&scope=read_write&state=%s",
		url.QueryEscape(clientID),
		url.QueryEscape(state),
	)
	c.JSON(http.StatusOK, gin.H{
		"authorize_url": authorizeURL,
		"dev_mode":      false,
	})
}

// HandleStripeConnectCallback receives the OAuth redirect from Stripe.
// Public route — authorization is carried in the state token.
func HandleStripeConnectCallback(c *gin.Context) {
	state := c.Query("state")
	code := c.Query("code")
	if state == "" || code == "" {
		c.String(http.StatusBadRequest, "missing code or state")
		return
	}

	tenantID, ok := consumeStripeConnectState(state)
	if !ok {
		c.String(http.StatusBadRequest, "invalid or expired state")
		return
	}

	var accountID, accessToken, refreshToken string

	if stripeConnectDevMode() {
		accountID = "acct_dev_" + tenantID.Hex()[:16]
		accessToken = "sk_dev_" + tenantID.Hex()
		refreshToken = "rt_dev_" + tenantID.Hex()
	} else {
		platformSecret := os.Getenv("STRIPE_PLATFORM_SECRET_KEY")
		if platformSecret == "" {
			c.String(http.StatusServiceUnavailable, "STRIPE_PLATFORM_SECRET_KEY not configured")
			return
		}
		tr, err := exchangeStripeConnectCode(platformSecret, code)
		if err != nil {
			log.Printf("[stripe connect] token exchange failed: %v", err)
			c.String(http.StatusBadGateway, "stripe token exchange failed: "+err.Error())
			return
		}
		accountID = tr.StripeUserID
		accessToken = tr.AccessToken
		refreshToken = tr.RefreshToken
	}

	update := bson.M{
		"stripe_connect_account_id": accountID,
		"timestamps.updated_at":     time.Now(),
	}
	if accessToken != "" {
		update["stripe_access_token"] = accessToken
	}
	if refreshToken != "" {
		update["stripe_refresh_token"] = refreshToken
	}
	if err := db.GetCollection(models.TenantCollection).UpdateId(tenantID, bson.M{"$set": update}); err != nil {
		log.Printf("[stripe connect] persist failed: %v", err)
		c.String(http.StatusInternalServerError, "failed to persist connect credentials")
		return
	}

	redirectTarget := dashboardReturnURL() + "/settings?stripe_connected=1"
	c.Redirect(http.StatusFound, redirectTarget)
}

// HandleStripeConnectDisconnect clears the tenant's Connect fields. It does
// NOT revoke the access token with Stripe — operators can do that from their
// Stripe dashboard if needed.
func HandleStripeConnectDisconnect(c *gin.Context) {
	tenantID := auth.GetTenantObjectID(c)
	if tenantID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	err := db.GetCollection(models.TenantCollection).UpdateId(tenantID, bson.M{"$unset": bson.M{
		"stripe_connect_account_id": "",
		"stripe_access_token":       "",
		"stripe_refresh_token":      "",
	}})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to disconnect"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "disconnected"})
}

type stripeTokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	StripeUserID string `json:"stripe_user_id"`
	Scope        string `json:"scope"`
	Error        string `json:"error"`
	ErrorDesc    string `json:"error_description"`
}

func exchangeStripeConnectCode(platformSecret, code string) (*stripeTokenResponse, error) {
	return exchangeStripeConnectCodeAt("https://connect.stripe.com/oauth/token", platformSecret, code)
}

// exchangeStripeConnectCodeAt is the injectable variant used in tests.
func exchangeStripeConnectCodeAt(endpoint, platformSecret, code string) (*stripeTokenResponse, error) {
	form := url.Values{}
	form.Set("grant_type", "authorization_code")
	form.Set("code", code)
	form.Set("client_secret", platformSecret)

	req, err := http.NewRequest("POST", endpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var tr stripeTokenResponse
	if err := json.Unmarshal(body, &tr); err != nil {
		return nil, fmt.Errorf("decode: %w (body=%s)", err, string(body))
	}
	if tr.Error != "" {
		return nil, fmt.Errorf("%s: %s", tr.Error, tr.ErrorDesc)
	}
	if tr.StripeUserID == "" {
		return nil, fmt.Errorf("stripe token response missing stripe_user_id (body=%s)", string(body))
	}
	return &tr, nil
}

// dashboardReturnURL is where the OAuth callback sends the browser after a
// successful (or failed) Connect round-trip. Falls back to the public API
// base if DASHBOARD_URL is unset.
func dashboardReturnURL() string {
	if v := os.Getenv("DASHBOARD_URL"); v != "" {
		return strings.TrimRight(v, "/")
	}
	return publicAPIBase()
}
