package routes

import (
	"net"
	"strings"

	coremodels "github.com/josephalai/sentanyl/core-service/models"
)

type DNSVerificationResult struct {
	DKIMValid      bool   `json:"dkim_valid"`
	DKIMFoundValue string `json:"dkim_found_value"`
	SPFValid       bool   `json:"spf_valid"`
	SPFFoundValue  string `json:"spf_found_value"`
	MXValid        bool   `json:"mx_valid"`
	MXFoundValue   string `json:"mx_found_value"`
}

// VerifyDomainDNS checks whether the required DNS records for a sending domain
// have been properly configured by the user.
func VerifyDomainDNS(domain *coremodels.SendingDomain, serverIP string) *DNSVerificationResult {
	result := &DNSVerificationResult{}

	// DKIM check: look up TXT record at selector._domainkey.parentDomain
	dkimRecords, _ := net.LookupTXT(domain.DNSRecords.DKIMName)
	for _, r := range dkimRecords {
		result.DKIMFoundValue = r
		if strings.Contains(r, domain.PublicKey) {
			result.DKIMValid = true
			break
		}
	}

	// SPF check: look up TXT records on the sending domain
	spfRecords, _ := net.LookupTXT(domain.Domain)
	for _, r := range spfRecords {
		if strings.HasPrefix(r, "v=spf1") {
			result.SPFFoundValue = r
			if strings.Contains(r, serverIP) || strings.Contains(r, "include:spf.sendhero.co") {
				result.SPFValid = true
			}
			break
		}
	}

	// MX check (optional — only needed if receiving replies)
	mxRecords, _ := net.LookupMX(domain.Domain)
	for _, mx := range mxRecords {
		result.MXFoundValue = strings.TrimSuffix(mx.Host, ".")
		if strings.Contains(mx.Host, "sendhero") {
			result.MXValid = true
			break
		}
	}

	return result
}
