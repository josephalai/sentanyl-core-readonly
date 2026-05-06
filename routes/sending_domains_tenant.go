package routes

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gopkg.in/mgo.v2/bson"

	"github.com/josephalai/sentanyl/pkg/auth"
	"github.com/josephalai/sentanyl/pkg/db"
	pkgmodels "github.com/josephalai/sentanyl/pkg/models"
)

// JWT-aware sending-domain handlers. The frontend at
// frontend/src/services/crud.ts:387-428 calls /api/tenant/sending-domain*
// without ever passing subscriber_id; the legacy handlers in domains.go
// still require it. These handlers read tenant identity from the JWT and
// scope every query by the tenant's hex string (stored in the existing
// creator_id column to avoid a schema migration).

// requireTenantHex resolves the JWT tenant for a sending-domain request and
// writes a 401 if absent. The hex string doubles as the creator_id value.
func requireTenantHex(c *gin.Context) (string, bool) {
	tenantOID := auth.GetTenantObjectID(c)
	if tenantOID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"status": "error", "error": "tenant authentication required"})
		return "", false
	}
	return tenantOID.Hex(), true
}

// findDomainForTenant looks up a SendingDomain owned by the JWT tenant.
func findDomainForTenant(domainId, tenantHex string) (*pkgmodels.SendingDomain, error) {
	sd := pkgmodels.SendingDomain{}
	query := bson.M{
		"public_id":             domainId,
		"creator_id":            tenantHex,
		"timestamps.deleted_at": nil,
	}
	if err := db.GetCollection(pkgmodels.SendingDomainCollection).Find(query).One(&sd); err != nil {
		return nil, err
	}
	return &sd, nil
}

func HandleAddTenantSendingDomain(c *gin.Context) {
	tenantHex, ok := requireTenantHex(c)
	if !ok {
		return
	}
	var req struct {
		Domain   string `json:"domain" binding:"required"`
		Selector string `json:"selector"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		handleInvalidBind(c)
		return
	}
	req.Domain = strings.ToLower(strings.TrimSpace(req.Domain))
	if req.Selector == "" {
		req.Selector = "s1"
	}

	existing := pkgmodels.SendingDomain{}
	dupQuery := bson.M{
		"domain":                req.Domain,
		"creator_id":            tenantHex,
		"timestamps.deleted_at": nil,
	}
	if db.GetCollection(pkgmodels.SendingDomainCollection).Find(dupQuery).One(&existing) == nil {
		handleReturnError(c, errors.New("domain already configured for this account"), http.StatusConflict)
		return
	}

	privPEM, pubBase64, err := GenerateDKIMKeyPair()
	if err != nil {
		handleReturnError(c, errors.New("failed to generate DKIM key pair"), http.StatusInternalServerError)
		return
	}

	vmta := "vm-" + req.Domain
	sidecarOK := false
	sidecarResp, err := SidecarAddDomain(req.Domain, req.Selector, privPEM)
	if err != nil {
		log.Printf("sidecar add domain warning (will save anyway): %v", err)
	} else {
		sidecarOK = true
		if sidecarResp.VMTA != "" {
			vmta = sidecarResp.VMTA
		}
	}

	parentDom := ParentDomain(req.Domain)
	dnsRecords := FormatDNSRecords(req.Selector, req.Domain, parentDom, pubBase64, ServerIP)

	sd := pkgmodels.NewSendingDomain()
	sd.CreatorId = tenantHex
	sd.Domain = req.Domain
	sd.Selector = req.Selector
	sd.VMTA = vmta
	sd.PublicKey = pubBase64
	sd.PrivateKey = privPEM
	sd.DNSRecords = dnsRecords
	sd.SetCreated()

	if err := db.GetCollection(pkgmodels.SendingDomainCollection).Insert(sd); err != nil {
		if sidecarOK {
			_ = SidecarDeleteDomain(req.Domain)
		}
		handleReturnError(c, errors.New("could not save domain"), http.StatusInternalServerError)
		return
	}

	resp := gin.H{"status": pkgmodels.HttpResponseStatusOK, "domain": sd}
	if !sidecarOK {
		resp["sidecar_warning"] = "Domain saved but mail server provisioning failed. It will need to be synced when the sidecar is reachable."
	}
	c.JSON(http.StatusCreated, resp)
}

func HandleListTenantSendingDomains(c *gin.Context) {
	tenantHex, ok := requireTenantHex(c)
	if !ok {
		return
	}
	var domains []pkgmodels.SendingDomain
	query := bson.M{
		"creator_id":            tenantHex,
		"timestamps.deleted_at": nil,
	}
	if err := db.GetCollection(pkgmodels.SendingDomainCollection).Find(query).All(&domains); err != nil {
		handleReturnError(c, err, http.StatusInternalServerError)
		return
	}
	if domains == nil {
		domains = []pkgmodels.SendingDomain{}
	}
	c.JSON(http.StatusOK, gin.H{"status": pkgmodels.HttpResponseStatusOK, "domains": domains})
}

func HandleGetTenantSendingDomain(c *gin.Context) {
	tenantHex, ok := requireTenantHex(c)
	if !ok {
		return
	}
	sd, err := findDomainForTenant(c.Param("domainId"), tenantHex)
	if err != nil {
		handleReturnNotFoundError(c, errors.New("domain not found"))
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": pkgmodels.HttpResponseStatusOK, "domain": sd})
}

func HandleDeleteTenantSendingDomain(c *gin.Context) {
	tenantHex, ok := requireTenantHex(c)
	if !ok {
		return
	}
	sd, err := findDomainForTenant(c.Param("domainId"), tenantHex)
	if err != nil {
		handleReturnNotFoundError(c, errors.New("domain not found"))
		return
	}
	if sidecarErr := SidecarDeleteDomain(sd.Domain); sidecarErr != nil {
		log.Printf("sidecar delete warning for %s: %v", sd.Domain, sidecarErr)
	}
	sd.SetDeleted()
	if err := db.GetCollection(pkgmodels.SendingDomainCollection).UpdateId(sd.Id, sd); err != nil {
		handleReturnUpdateError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": pkgmodels.HttpResponseStatusOK, "domain": sd})
}

func HandleVerifyTenantSendingDomainDNS(c *gin.Context) {
	tenantHex, ok := requireTenantHex(c)
	if !ok {
		return
	}
	sd, err := findDomainForTenant(c.Param("domainId"), tenantHex)
	if err != nil {
		handleReturnNotFoundError(c, errors.New("domain not found"))
		return
	}
	result := VerifyDomainDNS(sd, ServerIP)
	if result.DKIMValid && result.SPFValid && sd.Status == pkgmodels.DomainStatusPendingDNS {
		sd.Status = pkgmodels.DomainStatusActive
		sd.SetUpdated()
		_ = db.GetCollection(pkgmodels.SendingDomainCollection).UpdateId(sd.Id, sd)
	}
	c.JSON(http.StatusOK, gin.H{
		"status":       pkgmodels.HttpResponseStatusOK,
		"verification": result,
		"domain":       sd,
	})
}

func HandleTenantSendingDomainTestSend(c *gin.Context) {
	tenantHex, ok := requireTenantHex(c)
	if !ok {
		return
	}
	domainId := c.Param("domainId")
	var req struct {
		To      string `json:"to" binding:"required"`
		Subject string `json:"subject"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		handleInvalidBind(c)
		return
	}
	if req.Subject == "" {
		req.Subject = "Sidecar domain verification test"
	}
	sd, err := findDomainForTenant(domainId, tenantHex)
	if err != nil {
		handleReturnNotFoundError(c, errors.New("domain not found"))
		return
	}
	from := "test@" + sd.Domain
	result, err := SidecarTestSend(sd.Domain, req.To, from, req.Subject)
	if err != nil {
		handleReturnError(c, errors.New("test send failed: "+err.Error()), http.StatusBadGateway)
		return
	}
	var sidecarResp map[string]interface{}
	_ = json.Unmarshal(result, &sidecarResp)
	messageId, _ := sidecarResp["message_id"].(string)
	record := bson.M{
		"_id":            bson.NewObjectId(),
		"domain_pub_id":  domainId,
		"domain":         sd.Domain,
		"message_id":     messageId,
		"to":             req.To,
		"from":           from,
		"subject":        req.Subject,
		"vmta":           sd.VMTA,
		"submitted_at":   time.Now().UTC(),
		"sidecar_status": sidecarResp["status"],
	}
	if insertErr := db.GetCollection(pkgmodels.DomainTestSendCollection).Insert(record); insertErr != nil {
		log.Printf("warn: could not store test send record: %v", insertErr)
	}
	health, _ := SidecarGetHealth()
	c.JSON(http.StatusOK, gin.H{
		"status": pkgmodels.HttpResponseStatusOK,
		"result": json.RawMessage(result),
		"health": health,
	})
}

func HandleGetTenantSendingDomainTestSendStatus(c *gin.Context) {
	tenantHex, ok := requireTenantHex(c)
	if !ok {
		return
	}
	domainId := c.Param("domainId")
	sd, err := findDomainForTenant(domainId, tenantHex)
	if err != nil {
		handleReturnNotFoundError(c, errors.New("domain not found"))
		return
	}
	var testSends []bson.M
	_ = db.GetCollection(pkgmodels.DomainTestSendCollection).Find(bson.M{"domain_pub_id": domainId}).
		Sort("-submitted_at").Limit(5).All(&testSends)
	if testSends == nil {
		testSends = []bson.M{}
	}
	for i, ts := range testSends {
		if id, idOk := ts["_id"].(bson.ObjectId); idOk {
			testSends[i]["_id"] = id.Hex()
		}
	}
	health, healthErr := SidecarGetHealth()
	var healthData interface{}
	if healthErr == nil {
		healthData = health
	}
	stats, _ := SidecarGetStats(sd.Domain, "24h")
	queue, _ := SidecarGetQueueDepth()
	c.JSON(http.StatusOK, gin.H{
		"status":                 pkgmodels.HttpResponseStatusOK,
		"domain":                 sd.Domain,
		"accounting_log_enabled": health != nil && health.AccountingLogExists,
		"health":                 healthData,
		"stats_24h":              json.RawMessage(stats),
		"queue":                  json.RawMessage(queue),
		"test_sends":             testSends,
	})
}

func HandleGetTenantSendingDomainStats(c *gin.Context) {
	tenantHex, ok := requireTenantHex(c)
	if !ok {
		return
	}
	sd, err := findDomainForTenant(c.Param("domainId"), tenantHex)
	if err != nil {
		handleReturnNotFoundError(c, errors.New("domain not found"))
		return
	}
	since := c.DefaultQuery("since", "24h")
	result, err := SidecarGetStats(sd.Domain, since)
	if err != nil {
		handleReturnError(c, errors.New("stats unavailable: "+err.Error()), http.StatusBadGateway)
		return
	}
	c.Data(http.StatusOK, "application/json", result)
}

func HandleGetTenantSendingDomainReputation(c *gin.Context) {
	tenantHex, ok := requireTenantHex(c)
	if !ok {
		return
	}
	sd, err := findDomainForTenant(c.Param("domainId"), tenantHex)
	if err != nil {
		handleReturnNotFoundError(c, errors.New("domain not found"))
		return
	}
	result, err := SidecarGetReputation(sd.Domain)
	if err != nil {
		handleReturnError(c, errors.New("reputation data unavailable: "+err.Error()), http.StatusBadGateway)
		return
	}
	c.Data(http.StatusOK, "application/json", result)
}

func HandleGetTenantSendingDomainWarming(c *gin.Context) {
	tenantHex, ok := requireTenantHex(c)
	if !ok {
		return
	}
	sd, err := findDomainForTenant(c.Param("domainId"), tenantHex)
	if err != nil {
		handleReturnNotFoundError(c, errors.New("domain not found"))
		return
	}
	result, err := SidecarGetWarming(sd.Domain)
	if err != nil {
		handleReturnError(c, errors.New("warming data unavailable: "+err.Error()), http.StatusBadGateway)
		return
	}
	c.Data(http.StatusOK, "application/json", result)
}

func HandleGetTenantSendingDomainBounces(c *gin.Context) {
	tenantHex, ok := requireTenantHex(c)
	if !ok {
		return
	}
	sd, err := findDomainForTenant(c.Param("domainId"), tenantHex)
	if err != nil {
		handleReturnNotFoundError(c, errors.New("domain not found"))
		return
	}
	since := c.DefaultQuery("since", "24h")
	result, err := SidecarGetBounces(sd.Domain, since)
	if err != nil {
		handleReturnError(c, errors.New("bounce data unavailable: "+err.Error()), http.StatusBadGateway)
		return
	}
	c.Data(http.StatusOK, "application/json", result)
}

func HandlePauseTenantSendingDomain(c *gin.Context) {
	tenantHex, ok := requireTenantHex(c)
	if !ok {
		return
	}
	sd, err := findDomainForTenant(c.Param("domainId"), tenantHex)
	if err != nil {
		handleReturnNotFoundError(c, errors.New("domain not found"))
		return
	}
	if err := SidecarPauseDomain(sd.Domain); err != nil {
		handleReturnError(c, errors.New("pause failed: "+err.Error()), http.StatusBadGateway)
		return
	}
	sd.Status = pkgmodels.DomainStatusPaused
	sd.SetUpdated()
	_ = db.GetCollection(pkgmodels.SendingDomainCollection).UpdateId(sd.Id, sd)
	c.JSON(http.StatusOK, gin.H{"status": pkgmodels.HttpResponseStatusOK, "domain": sd})
}

func HandleResumeTenantSendingDomain(c *gin.Context) {
	tenantHex, ok := requireTenantHex(c)
	if !ok {
		return
	}
	sd, err := findDomainForTenant(c.Param("domainId"), tenantHex)
	if err != nil {
		handleReturnNotFoundError(c, errors.New("domain not found"))
		return
	}
	if err := SidecarResumeDomain(sd.Domain); err != nil {
		handleReturnError(c, errors.New("resume failed: "+err.Error()), http.StatusBadGateway)
		return
	}
	sd.Status = pkgmodels.DomainStatusActive
	sd.SetUpdated()
	_ = db.GetCollection(pkgmodels.SendingDomainCollection).UpdateId(sd.Id, sd)
	c.JSON(http.StatusOK, gin.H{"status": pkgmodels.HttpResponseStatusOK, "domain": sd})
}
