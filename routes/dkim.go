package routes

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"fmt"
	"strings"

	pkgmodels "github.com/josephalai/sentanyl/pkg/models"
)

// GenerateDKIMKeyPair creates an RSA-2048 key pair for DKIM signing.
// Returns PEM-encoded private key and base64-encoded DER public key.
func GenerateDKIMKeyPair() (privatePEM string, publicBase64 string, err error) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return "", "", fmt.Errorf("rsa key generation failed: %w", err)
	}

	privDER := x509.MarshalPKCS1PrivateKey(key)
	privBlock := &pem.Block{Type: "RSA PRIVATE KEY", Bytes: privDER}
	privatePEM = string(pem.EncodeToMemory(privBlock))

	pubDER, err := x509.MarshalPKIXPublicKey(&key.PublicKey)
	if err != nil {
		return "", "", fmt.Errorf("public key marshal failed: %w", err)
	}
	publicBase64 = base64.StdEncoding.EncodeToString(pubDER)

	return privatePEM, publicBase64, nil
}

// FormatDNSRecords computes the DNS records a user must add for their sending domain.
func FormatDNSRecords(selector, sendingDomain, parentDomain, publicBase64, serverIP string) pkgmodels.DNSRecords {
	return pkgmodels.DNSRecords{
		DKIMName: fmt.Sprintf("%s._domainkey.%s", selector, parentDomain),
		DKIM:     fmt.Sprintf("v=DKIM1; k=rsa; p=%s", publicBase64),
		SPF:      fmt.Sprintf("v=spf1 ip4:%s ~all", serverIP),
		MX:       "smtp.sendhero.co",
	}
}

// ParentDomain extracts the registrable domain from a sending subdomain.
// "mail.acme.com" -> "acme.com", "acme.com" -> "acme.com"
func ParentDomain(domain string) string {
	parts := strings.Split(domain, ".")
	if len(parts) <= 2 {
		return domain
	}
	return strings.Join(parts[len(parts)-2:], ".")
}
