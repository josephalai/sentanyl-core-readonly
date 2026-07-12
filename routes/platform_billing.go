package routes

import (
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"gopkg.in/mgo.v2/bson"

	"github.com/josephalai/sentanyl/core-service/internal/billing"
	"github.com/josephalai/sentanyl/pkg/auth"
	"github.com/josephalai/sentanyl/pkg/db"
	"github.com/josephalai/sentanyl/pkg/models"
)

// Platform billing endpoints — charging the tenant for Sentanyl itself via
// Stripe Billing on the platform account. These must be registered on the
// UNGATED tenant group so an unpaid tenant can still reach them to pay.

func platformStripeKey() string  { return os.Getenv("STRIPE_PLATFORM_SECRET_KEY") }
func platformPriceID() string    { return os.Getenv("STRIPE_PLATFORM_PRICE_ID") }

// HandleGetBillingStatus returns the tenant's platform subscription state,
// computed with the same SubscriptionAllowed logic the enforcement middleware
// uses so the dashboard and the gate can never disagree.
func HandleGetBillingStatus(c *gin.Context) {
	tenantID := auth.GetTenantObjectID(c)
	if tenantID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	var tenant models.Tenant
	if err := db.GetCollection(models.TenantCollection).FindId(tenantID).One(&tenant); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "tenant not found"})
		return
	}

	now := time.Now()
	daysLeft := 0
	if tenant.SubscriptionStatus == "trial" && tenant.TrialEndsAt != nil {
		if d := tenant.TrialEndsAt.Sub(now); d > 0 {
			daysLeft = int(d.Hours()/24) + 1
		}
	}
	c.JSON(http.StatusOK, gin.H{
		"subscription_status": tenant.SubscriptionStatus,
		"trial_ends_at":       tenant.TrialEndsAt,
		"past_due_at":         tenant.PastDueAt,
		"has_subscription":    tenant.PlatformSubscriptionID != "",
		"gated":               !auth.SubscriptionAllowed(&tenant, now),
		"days_left":           daysLeft,
	})
}

// HandleCreateBillingCheckoutSession lazily creates the platform Stripe
// customer, then returns a hosted Checkout URL (subscription mode) that
// preserves whatever trial time the tenant has left.
func HandleCreateBillingCheckoutSession(c *gin.Context) {
	tenantID := auth.GetTenantObjectID(c)
	if tenantID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	key, priceID := platformStripeKey(), platformPriceID()
	if key == "" || priceID == "" {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "platform billing is not configured"})
		return
	}

	var tenant models.Tenant
	if err := db.GetCollection(models.TenantCollection).FindId(tenantID).One(&tenant); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "tenant not found"})
		return
	}

	if tenant.PlatformStripeCustomerID == "" {
		email, _ := c.Get(auth.ContextEmail)
		emailStr, _ := email.(string)
		customerID, err := billing.CreateCustomer(key, emailStr, tenant.BusinessName, tenantID.Hex())
		if err != nil {
			log.Printf("[platform billing] create customer: %v", err)
			c.JSON(http.StatusBadGateway, gin.H{"error": "failed to create billing customer"})
			return
		}
		if err := db.GetCollection(models.TenantCollection).UpdateId(tenantID,
			bson.M{"$set": bson.M{"platform_stripe_customer_id": customerID}}); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save billing customer"})
			return
		}
		tenant.PlatformStripeCustomerID = customerID
	}

	base := publicAPIBase()
	url, err := billing.CreateSubscriptionCheckoutSession(
		key, priceID, tenant.PlatformStripeCustomerID, tenantID.Hex(),
		base+"/billing?checkout=success", base+"/billing?checkout=cancel",
		tenant.TrialEndsAt,
	)
	if err != nil {
		log.Printf("[platform billing] create checkout session: %v", err)
		c.JSON(http.StatusBadGateway, gin.H{"error": "failed to create checkout session"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"url": url})
}

// HandleCreateBillingPortalSession returns a Stripe Billing Portal URL for
// card updates, invoices, and cancellation.
func HandleCreateBillingPortalSession(c *gin.Context) {
	tenantID := auth.GetTenantObjectID(c)
	if tenantID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	key := platformStripeKey()
	if key == "" {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "platform billing is not configured"})
		return
	}

	var tenant models.Tenant
	if err := db.GetCollection(models.TenantCollection).FindId(tenantID).One(&tenant); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "tenant not found"})
		return
	}
	if tenant.PlatformStripeCustomerID == "" {
		c.JSON(http.StatusConflict, gin.H{"error": "no billing customer yet — add a payment method first"})
		return
	}

	url, err := billing.CreatePortalSession(key, tenant.PlatformStripeCustomerID, publicAPIBase()+"/billing")
	if err != nil {
		log.Printf("[platform billing] create portal session: %v", err)
		c.JSON(http.StatusBadGateway, gin.H{"error": "failed to create portal session"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"url": url})
}
