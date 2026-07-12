package routes

import (
	"fmt"
	"log"
	"strings"

	"gopkg.in/mgo.v2/bson"

	"github.com/josephalai/sentanyl/pkg/db"
	pkgmodels "github.com/josephalai/sentanyl/pkg/models"
)

// ProvisionSendingDomain is the single path that hooks a domain into the
// email infrastructure: DKIM keypair, pmta-sidecar registration (PowerMTA
// vmta + signing key), and the tenant-scoped sending_domains record with the
// DNS records the tenant must publish. Idempotent — an existing non-deleted
// row for (tenant, domain) is returned as-is.
//
// Called from the explicit HandleAddTenantSendingDomain API and from
// AutoProvisionEmailForDomain when a tenant connects a website domain, so a
// tenant who brings a domain anywhere in the product is automatically wired
// for sending without a second manual setup step.
func ProvisionSendingDomain(tenantHex, domain, selector string) (sd *pkgmodels.SendingDomain, created, sidecarOK bool, err error) {
	if selector == "" {
		selector = "s1"
	}

	existing := pkgmodels.SendingDomain{}
	err = db.GetCollection(pkgmodels.SendingDomainCollection).Find(bson.M{
		"domain":                domain,
		"creator_id":            tenantHex,
		"timestamps.deleted_at": nil,
	}).One(&existing)
	if err == nil {
		return &existing, false, true, nil
	}

	privPEM, pubBase64, err := GenerateDKIMKeyPair()
	if err != nil {
		return nil, false, false, fmt.Errorf("generate DKIM key pair: %w", err)
	}

	vmta := "vm-" + domain
	sidecarResp, err := SidecarAddDomain(domain, selector, privPEM)
	if err != nil {
		// Saved anyway — the row carries the key material, so the sidecar can
		// be synced later without re-issuing DNS instructions to the tenant.
		log.Printf("[sending-domain] sidecar add %s warning (saving anyway): %v", domain, err)
	} else {
		sidecarOK = true
		if sidecarResp.VMTA != "" {
			vmta = sidecarResp.VMTA
		}
	}

	parentDom := ParentDomain(domain)
	dnsRecords := FormatDNSRecords(selector, domain, parentDom, pubBase64, ServerIP)

	sd = pkgmodels.NewSendingDomain()
	sd.CreatorId = tenantHex
	sd.Domain = domain
	sd.Selector = selector
	sd.VMTA = vmta
	sd.PublicKey = pubBase64
	sd.PrivateKey = privPEM
	sd.DNSRecords = dnsRecords
	sd.SetCreated()

	if err := db.GetCollection(pkgmodels.SendingDomainCollection).Insert(sd); err != nil {
		if sidecarOK {
			_ = SidecarDeleteDomain(domain)
		}
		return nil, false, sidecarOK, fmt.Errorf("save sending domain: %w", err)
	}
	return sd, true, sidecarOK, nil
}

// AutoProvisionEmailForDomain wires email sending for the registrable parent
// of a website domain the tenant just connected (staging.example.com →
// example.com). Best-effort: failures are logged, never surfaced to the
// website-domain flow that triggered it.
func AutoProvisionEmailForDomain(tenantID bson.ObjectId, hostname string) {
	apex := ParentDomain(hostname)
	if apex == "" || !strings.Contains(apex, ".") ||
		strings.HasSuffix(apex, ".localhost") || strings.HasSuffix(apex, ".lvh.me") ||
		apex == "localhost" || isSharedInfraHost(apex) {
		return
	}
	sd, created, _, err := ProvisionSendingDomain(tenantID.Hex(), apex, "s1")
	if err != nil {
		log.Printf("[sending-domain] auto-provision %s for tenant %s failed: %v", apex, tenantID.Hex(), err)
		return
	}
	if created {
		log.Printf("[sending-domain] auto-provisioned %s (vmta %s) for tenant %s from website domain %s",
			apex, sd.VMTA, tenantID.Hex(), hostname)
	}
}

// MaybeSetTenantFromDomain sets the tenant's default from-domain (the
// mailgun_domain field, read by password-setup emails, coaching mail, and
// campaign defaults) the first time one of their sending domains becomes
// active. Never overwrites an explicit choice.
func MaybeSetTenantFromDomain(tenantHex, domain string) {
	if !bson.IsObjectIdHex(tenantHex) {
		return
	}
	err := db.GetCollection(pkgmodels.TenantCollection).Update(
		bson.M{
			"_id": bson.ObjectIdHex(tenantHex),
			"$or": []bson.M{
				{"mailgun_domain": ""},
				{"mailgun_domain": bson.M{"$exists": false}},
			},
		},
		bson.M{"$set": bson.M{"mailgun_domain": domain}},
	)
	if err == nil {
		log.Printf("[sending-domain] tenant %s default from-domain set to %s", tenantHex, domain)
	}
}

// isSharedInfraHost blocks auto-provisioning for hosts that belong to the
// platform rather than the tenant (dev hosts and Sentanyl's own domains).
func isSharedInfraHost(apex string) bool {
	switch apex {
	case "lvh.me", "sentanyl.com", "sendhero.co", "localhost.localdomain":
		return true
	}
	return false
}
