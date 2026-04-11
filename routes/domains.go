package routes

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strings"
	"time"

	coremodels "github.com/josephalai/sentanyl/core-service/models"
	"github.com/josephalai/sentanyl/pkg/db"
	"github.com/josephalai/sentanyl/pkg/models"

	"github.com/gin-gonic/gin"
	"gopkg.in/mgo.v2/bson"
)

// ServerIP is the PowerMTA server's public IP, used in SPF record instructions.
const ServerIP = "5.78.200.152"

// ── helpers ─────────────────────────────────────────────────────────────────

// findDomainByPublicId looks up a SendingDomain by public_id + creator_id
// (verifying the creator owns this domain).
func findDomainByPublicId(domainId, creatorPublicId string) (*coremodels.SendingDomain, error) {
	sd := coremodels.SendingDomain{}
	query := bson.M{
		"public_id":             domainId,
		"creator_id":            creatorPublicId,
		"timestamps.deleted_at": nil,
	}
	if err := db.GetCollection(models.SendingDomainCollection).Find(query).One(&sd); err != nil {
		return nil, err
	}
	return &sd, nil
}

// GetCreatorByPublicId looks up a Creator by public_id.
func GetCreatorByPublicId(publicId string) (*coremodels.Creator, error) {
	c := coremodels.Creator{}
	err := db.GetCollection(models.CreatorCollection).Find(bson.M{"public_id": publicId}).One(&c)
	if err != nil {
		return nil, err
	}
	return &c, nil
}

// ── Sidecar stubs ───────────────────────────────────────────────────────────
// These are stubs for the PowerMTA sidecar integration. In a full deployment
// they will call the sidecar HTTP API.

type SidecarResponse struct {
	VMTA string `json:"vmta"`
}

func SidecarAddDomain(domain, selector, privPEM string) (*SidecarResponse, error) {
	log.Printf("sidecar: add domain %s (stub)", domain)
	return &SidecarResponse{VMTA: "vm-" + domain}, nil
}

func SidecarDeleteDomain(domain string) error {
	log.Printf("sidecar: delete domain %s (stub)", domain)
	return nil
}

func SidecarTestSend(domain, to, from, subject string) ([]byte, error) {
	log.Printf("sidecar: test send from %s to %s (stub)", from, to)
	resp := map[string]interface{}{
		"status":     "accepted",
		"message_id": "stub-" + domain,
	}
	return json.Marshal(resp)
}

type SidecarHealth struct {
	AccountingLogExists bool `json:"accounting_log_exists"`
}

func SidecarGetHealth() (*SidecarHealth, error) {
	return &SidecarHealth{AccountingLogExists: true}, nil
}

func SidecarGetStats(domain, since string) ([]byte, error) {
	return json.Marshal(map[string]interface{}{"domain": domain, "since": since})
}

func SidecarGetQueueDepth() ([]byte, error) {
	return json.Marshal(map[string]interface{}{"depth": 0})
}

func SidecarGetReputation(domain string) ([]byte, error) {
	return json.Marshal(map[string]interface{}{"domain": domain})
}

func SidecarGetWarming(domain string) ([]byte, error) {
	return json.Marshal(map[string]interface{}{"domain": domain})
}

func SidecarGetBounces(domain, since string) ([]byte, error) {
	return json.Marshal(map[string]interface{}{"domain": domain, "since": since})
}

func SidecarPauseDomain(domain string) error {
	log.Printf("sidecar: pause domain %s (stub)", domain)
	return nil
}

func SidecarResumeDomain(domain string) error {
	log.Printf("sidecar: resume domain %s (stub)", domain)
	return nil
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

// ── POST /api/domain ────────────────────────────────────────────────────────

func HandleAddDomain(c *gin.Context) {
	var req struct {
		SubscriberId string `json:"subscriber_id" binding:"required"`
		Domain       string `json:"domain" binding:"required"`
		Selector     string `json:"selector"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		handleInvalidBind(c)
		return
	}

	req.Domain = strings.ToLower(strings.TrimSpace(req.Domain))
	if req.Selector == "" {
		req.Selector = "s1"
	}

	creator, err := GetCreatorByPublicId(req.SubscriberId)
	if err != nil {
		handleReturnNotFoundError(c, errors.New("creator not found"))
		return
	}

	// Check for duplicate domain under this creator.
	existing := coremodels.SendingDomain{}
	dupQuery := bson.M{
		"domain":                req.Domain,
		"creator_id":            creator.PublicId,
		"timestamps.deleted_at": nil,
	}
	if db.GetCollection(models.SendingDomainCollection).Find(dupQuery).One(&existing) == nil {
		handleReturnError(c, errors.New("domain already configured for this account"), http.StatusConflict)
		return
	}

	// 1. Generate DKIM key pair.
	privPEM, pubBase64, err := GenerateDKIMKeyPair()
	if err != nil {
		handleReturnError(c, errors.New("failed to generate DKIM key pair"), http.StatusInternalServerError)
		return
	}

	// 2. Call sidecar to configure domain on PowerMTA (best-effort).
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

	// 3. Build DNS records.
	parentDom := ParentDomain(req.Domain)
	dnsRecords := FormatDNSRecords(req.Selector, req.Domain, parentDom, pubBase64, ServerIP)

	// 4. Save SendingDomain entity.
	sd := coremodels.NewSendingDomain()
	sd.CreatorId = creator.PublicId
	sd.Domain = req.Domain
	sd.Selector = req.Selector
	sd.VMTA = vmta
	sd.PublicKey = pubBase64
	sd.PrivateKey = privPEM
	sd.DNSRecords = dnsRecords
	sd.SetCreated()

	if err := db.GetCollection(models.SendingDomainCollection).Insert(sd); err != nil {
		if sidecarOK {
			_ = SidecarDeleteDomain(req.Domain)
		}
		handleReturnError(c, errors.New("could not save domain"), http.StatusInternalServerError)
		return
	}

	resp := gin.H{
		"status": models.HttpResponseStatusOK,
		"domain": sd,
	}
	if !sidecarOK {
		resp["sidecar_warning"] = "Domain saved but mail server provisioning failed. It will need to be synced when the sidecar is reachable."
	}
	c.JSON(http.StatusCreated, resp)
}

// ── GET /api/domains ────────────────────────────────────────────────────────

func HandleGetDomains(c *gin.Context) {
	subscriberId := c.Query("subscriber_id")
	if subscriberId == "" {
		handleInvalidBind(c)
		return
	}

	var domains []coremodels.SendingDomain
	query := bson.M{
		"creator_id":            subscriberId,
		"timestamps.deleted_at": nil,
	}
	if err := db.GetCollection(models.SendingDomainCollection).Find(query).All(&domains); err != nil {
		handleReturnError(c, err, http.StatusInternalServerError)
		return
	}
	if domains == nil {
		domains = []coremodels.SendingDomain{}
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  models.HttpResponseStatusOK,
		"domains": domains,
	})
}

// ── GET /api/domain/:domainId ───────────────────────────────────────────────

func HandleGetDomain(c *gin.Context) {
	domainId := c.Param("domainId")
	subscriberId := c.Query("subscriber_id")
	if subscriberId == "" {
		handleInvalidBind(c)
		return
	}

	sd, err := findDomainByPublicId(domainId, subscriberId)
	if err != nil {
		handleReturnNotFoundError(c, errors.New("domain not found"))
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status": models.HttpResponseStatusOK,
		"domain": sd,
	})
}

// ── DELETE /api/domain/:domainId ────────────────────────────────────────────

func HandleDeleteDomain(c *gin.Context) {
	domainId := c.Param("domainId")
	var req struct {
		SubscriberId string `json:"subscriber_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		handleInvalidBind(c)
		return
	}

	sd, err := findDomainByPublicId(domainId, req.SubscriberId)
	if err != nil {
		handleReturnNotFoundError(c, errors.New("domain not found"))
		return
	}

	if sidecarErr := SidecarDeleteDomain(sd.Domain); sidecarErr != nil {
		log.Printf("sidecar delete warning for %s: %v", sd.Domain, sidecarErr)
	}

	sd.SetDeleted()
	if err := db.GetCollection(models.SendingDomainCollection).UpdateId(sd.Id, sd); err != nil {
		handleReturnUpdateError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status": models.HttpResponseStatusOK,
		"domain": sd,
	})
}

// ── POST /api/domain/:domainId/verify-dns ───────────────────────────────────

func HandleVerifyDNS(c *gin.Context) {
	domainId := c.Param("domainId")
	var req struct {
		SubscriberId string `json:"subscriber_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		handleInvalidBind(c)
		return
	}

	sd, err := findDomainByPublicId(domainId, req.SubscriberId)
	if err != nil {
		handleReturnNotFoundError(c, errors.New("domain not found"))
		return
	}

	result := VerifyDomainDNS(sd, ServerIP)

	if result.DKIMValid && result.SPFValid && sd.Status == coremodels.DomainStatusPendingDNS {
		sd.Status = coremodels.DomainStatusActive
		sd.SetUpdated()
		_ = db.GetCollection(models.SendingDomainCollection).UpdateId(sd.Id, sd)
	}

	c.JSON(http.StatusOK, gin.H{
		"status":       models.HttpResponseStatusOK,
		"verification": result,
		"domain":       sd,
	})
}

// ── POST /api/domain/:domainId/test-send ────────────────────────────────────

func HandleTestSend(c *gin.Context) {
	domainId := c.Param("domainId")
	var req struct {
		SubscriberId string `json:"subscriber_id" binding:"required"`
		To           string `json:"to" binding:"required"`
		Subject      string `json:"subject"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		handleInvalidBind(c)
		return
	}
	if req.Subject == "" {
		req.Subject = "Sidecar domain verification test"
	}

	sd, err := findDomainByPublicId(domainId, req.SubscriberId)
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
	if insertErr := db.GetCollection(models.DomainTestSendCollection).Insert(record); insertErr != nil {
		log.Printf("warn: could not store test send record: %v", insertErr)
	}

	health, _ := SidecarGetHealth()

	c.JSON(http.StatusOK, gin.H{
		"status": models.HttpResponseStatusOK,
		"result": json.RawMessage(result),
		"health": health,
	})
}

// ── GET /api/domain/:domainId/test-send-status ──────────────────────────────

func HandleGetTestSendStatus(c *gin.Context) {
	domainId := c.Param("domainId")
	subscriberId := c.Query("subscriber_id")
	if subscriberId == "" {
		handleInvalidBind(c)
		return
	}

	sd, err := findDomainByPublicId(domainId, subscriberId)
	if err != nil {
		handleReturnNotFoundError(c, errors.New("domain not found"))
		return
	}

	var testSends []bson.M
	_ = db.GetCollection(models.DomainTestSendCollection).Find(bson.M{"domain_pub_id": domainId}).
		Sort("-submitted_at").Limit(5).All(&testSends)
	if testSends == nil {
		testSends = []bson.M{}
	}

	for i, ts := range testSends {
		if id, ok := ts["_id"].(bson.ObjectId); ok {
			testSends[i]["_id"] = id.Hex()
		}
	}

	health, healthErr := SidecarGetHealth()
	var healthData interface{} = nil
	if healthErr == nil {
		healthData = health
	}

	stats, _ := SidecarGetStats(sd.Domain, "24h")
	queue, _ := SidecarGetQueueDepth()

	c.JSON(http.StatusOK, gin.H{
		"status":                 models.HttpResponseStatusOK,
		"domain":                 sd.Domain,
		"accounting_log_enabled": health != nil && health.AccountingLogExists,
		"health":                 healthData,
		"stats_24h":              json.RawMessage(stats),
		"queue":                  json.RawMessage(queue),
		"test_sends":             testSends,
	})
}

// ── Sidecar proxy handlers ──────────────────────────────────────────────────

func HandleGetDomainStats(c *gin.Context) {
	domainId := c.Param("domainId")
	subscriberId := c.Query("subscriber_id")
	if subscriberId == "" {
		handleInvalidBind(c)
		return
	}

	sd, err := findDomainByPublicId(domainId, subscriberId)
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

func HandleGetDomainReputation(c *gin.Context) {
	domainId := c.Param("domainId")
	subscriberId := c.Query("subscriber_id")
	if subscriberId == "" {
		handleInvalidBind(c)
		return
	}

	sd, err := findDomainByPublicId(domainId, subscriberId)
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

func HandleGetDomainWarming(c *gin.Context) {
	domainId := c.Param("domainId")
	subscriberId := c.Query("subscriber_id")
	if subscriberId == "" {
		handleInvalidBind(c)
		return
	}

	sd, err := findDomainByPublicId(domainId, subscriberId)
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

func HandleGetDomainBounces(c *gin.Context) {
	domainId := c.Param("domainId")
	subscriberId := c.Query("subscriber_id")
	if subscriberId == "" {
		handleInvalidBind(c)
		return
	}

	sd, err := findDomainByPublicId(domainId, subscriberId)
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

func HandlePauseDomain(c *gin.Context) {
	domainId := c.Param("domainId")
	var req struct {
		SubscriberId string `json:"subscriber_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		handleInvalidBind(c)
		return
	}

	sd, err := findDomainByPublicId(domainId, req.SubscriberId)
	if err != nil {
		handleReturnNotFoundError(c, errors.New("domain not found"))
		return
	}

	if err := SidecarPauseDomain(sd.Domain); err != nil {
		handleReturnError(c, errors.New("pause failed: "+err.Error()), http.StatusBadGateway)
		return
	}

	sd.Status = coremodels.DomainStatusPaused
	sd.SetUpdated()
	_ = db.GetCollection(models.SendingDomainCollection).UpdateId(sd.Id, sd)

	c.JSON(http.StatusOK, gin.H{
		"status": models.HttpResponseStatusOK,
		"domain": sd,
	})
}

func HandleResumeDomain(c *gin.Context) {
	domainId := c.Param("domainId")
	var req struct {
		SubscriberId string `json:"subscriber_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		handleInvalidBind(c)
		return
	}

	sd, err := findDomainByPublicId(domainId, req.SubscriberId)
	if err != nil {
		handleReturnNotFoundError(c, errors.New("domain not found"))
		return
	}

	if err := SidecarResumeDomain(sd.Domain); err != nil {
		handleReturnError(c, errors.New("resume failed: "+err.Error()), http.StatusBadGateway)
		return
	}

	sd.Status = coremodels.DomainStatusActive
	sd.SetUpdated()
	_ = db.GetCollection(models.SendingDomainCollection).UpdateId(sd.Id, sd)

	c.JSON(http.StatusOK, gin.H{
		"status": models.HttpResponseStatusOK,
		"domain": sd,
	})
}
