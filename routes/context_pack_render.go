package routes

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gopkg.in/mgo.v2/bson"

	"github.com/josephalai/sentanyl/pkg/auth"
	"github.com/josephalai/sentanyl/pkg/db"
	pkgmodels "github.com/josephalai/sentanyl/pkg/models"
	"github.com/josephalai/sentanyl/pkg/storage"
)

// contextPDFStorage and contextPDFBucket are wired from core-service main
// after GCS init. nil means render endpoints return 503 — same convention
// as the downloads routes in marketing-service.
var (
	contextPDFStorage storage.StorageProvider
	contextPDFBucket  string
)

// SetContextRenderStorage wires the storage provider used to upload rendered
// context-pack PDFs. Called once at startup from core-service/cmd/main.go.
func SetContextRenderStorage(p storage.StorageProvider, bucket string) {
	contextPDFStorage = p
	contextPDFBucket = bucket
}

// RegisterContextPackRenderRoutes registers the two PDF-render endpoints on
// the same authenticated tenant group used by RegisterContextPackRoutes.
//   - render-pdf: produce a PDF asset from a saved context, return the asset.
//   - render-product: do the above, then materialize a draft Digital Download
//     Product wrapping the asset so the tenant can publish/sell it.
func RegisterContextPackRenderRoutes(tenantAPI *gin.RouterGroup) {
	tenantAPI.POST("/context-packs/:packId/render-pdf", handleRenderContextPDF)
	tenantAPI.POST("/context-packs/:packId/render-product", handleRenderContextProduct)
}

// safePackFilename strips characters that would be unsafe in an HTTP
// Content-Disposition value or a filesystem path. Empty input falls back to
// "context".
func safePackFilename(name string) string {
	if name == "" {
		return "context"
	}
	var b strings.Builder
	prevUnderscore := false
	for _, r := range name {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '-', r == '_', r == '.':
			b.WriteRune(r)
			prevUnderscore = false
		default:
			if !prevUnderscore {
				b.WriteByte('_')
				prevUnderscore = true
			}
		}
	}
	out := strings.Trim(b.String(), "_.")
	if out == "" {
		return "context"
	}
	return out
}

func pdfServiceBase() string {
	if u := os.Getenv("PDF_SERVICE_URL"); u != "" {
		return u
	}
	return "http://localhost:9774"
}

// loadOwnedContextPack returns the pack (resolving by Mongo _id or PublicId)
// scoped to the requesting tenant. Returns (nil, "msg") on miss so the caller
// can write a 404.
func loadOwnedContextPack(tenantID bson.ObjectId, packIdParam string) (*pkgmodels.ContextPack, string) {
	q := bson.M{
		"tenant_id":             tenantID,
		"timestamps.deleted_at": nil,
	}
	if bson.IsObjectIdHex(packIdParam) {
		q["_id"] = bson.ObjectIdHex(packIdParam)
	} else {
		q["public_id"] = packIdParam
	}
	var pack pkgmodels.ContextPack
	if err := db.GetCollection(pkgmodels.ContextPackCollection).Find(q).One(&pack); err != nil {
		return nil, "context pack not found"
	}
	return &pack, ""
}

// renderAndUploadContextPDF concatenates the pack's chunks into markdown,
// posts to pdf-service /generate, and uploads the resulting PDF to GCS. The
// caller then attaches the fileURL/objectPath to an Asset row. The function
// is intentionally not exported — the only consumers are the two endpoints.
func renderAndUploadContextPDF(pack *pkgmodels.ContextPack, theme string) (fileURL, objectPath string, size int64, err error) {
	if contextPDFStorage == nil {
		return "", "", 0, fmt.Errorf("storage not configured")
	}
	if theme == "" {
		theme = "minimal"
	}

	var sb strings.Builder
	if pack.Notes != "" {
		sb.WriteString("> ")
		sb.WriteString(strings.ReplaceAll(pack.Notes, "\n", "\n> "))
		sb.WriteString("\n\n")
	}
	for _, ch := range pack.Chunks {
		sb.WriteString(ch.Text)
	}
	markdown := sb.String()
	if strings.TrimSpace(markdown) == "" {
		return "", "", 0, fmt.Errorf("context pack has no content")
	}

	reqBody, _ := json.Marshal(map[string]string{
		"markdown": markdown,
		"title":    pack.Name,
		"theme":    theme,
	})
	client := &http.Client{Timeout: 120 * time.Second}
	resp, err := client.Post(pdfServiceBase()+"/generate", "application/json", bytes.NewReader(reqBody))
	if err != nil {
		return "", "", 0, fmt.Errorf("pdf service unavailable: %w", err)
	}
	defer resp.Body.Close()
	pdfBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", "", 0, err
	}
	if resp.StatusCode != http.StatusOK {
		return "", "", 0, fmt.Errorf("pdf service error: %s", string(pdfBytes))
	}

	objectPath = fmt.Sprintf("%s/downloads/context_%s_%d.pdf",
		pack.TenantID.Hex(),
		pack.PublicId,
		time.Now().Unix(),
	)
	fileURL, err = contextPDFStorage.UploadObject(contextPDFBucket, objectPath, "application/pdf", bytes.NewReader(pdfBytes))
	if err != nil {
		return "", "", 0, fmt.Errorf("gcs upload failed: %w", err)
	}
	return fileURL, objectPath, int64(len(pdfBytes)), nil
}

// makeContextDownloadAsset persists an Asset row pointing at the rendered PDF.
// GenConfig stamps the source pack so future tenant flows (e.g. "regenerate
// when the context updates") can find downloads derived from a given pack.
func makeContextDownloadAsset(tenantID bson.ObjectId, pack *pkgmodels.ContextPack, fileURL, objectPath string, size int64) (*pkgmodels.Asset, error) {
	asset := pkgmodels.NewAsset()
	asset.TenantID = tenantID
	asset.Title = pack.Name
	asset.Kind = "download_pdf_from_context"
	asset.Status = "ready"
	asset.FileURL = fileURL
	asset.FileName = fmt.Sprintf("%s.pdf", safePackFilename(pack.Name))
	asset.FileType = "application/pdf"
	asset.FileSize = size
	asset.S3Key = objectPath

	if err := db.GetCollection(pkgmodels.AssetCollection).Insert(asset); err != nil {
		return nil, err
	}
	return asset, nil
}

// handleRenderContextPDF generates the PDF and returns the new Asset. The
// tenant manually attaches it to a Digital Download Product (or a course
// lesson) via the existing admin UI.
func handleRenderContextPDF(c *gin.Context) {
	tenantID := auth.GetTenantObjectID(c)
	if tenantID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	pack, missing := loadOwnedContextPack(tenantID, c.Param("packId"))
	if missing != "" {
		c.JSON(http.StatusNotFound, gin.H{"error": missing})
		return
	}
	theme := c.Query("theme")

	fileURL, objectPath, size, err := renderAndUploadContextPDF(pack, theme)
	if err != nil {
		log.Printf("[context-render] render failed: %v", err)
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}
	asset, err := makeContextDownloadAsset(tenantID, pack, fileURL, objectPath, size)
	if err != nil {
		log.Printf("[context-render] asset insert failed: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save asset"})
		return
	}
	c.JSON(http.StatusCreated, asset)
}

// handleRenderContextProduct chains render-pdf + asset insert + a fresh
// digital_download Product creation. Returns the new product so the tenant
// frontend can redirect into the Files page (/products/:id/downloads).
func handleRenderContextProduct(c *gin.Context) {
	tenantID := auth.GetTenantObjectID(c)
	if tenantID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	pack, missing := loadOwnedContextPack(tenantID, c.Param("packId"))
	if missing != "" {
		c.JSON(http.StatusNotFound, gin.H{"error": missing})
		return
	}
	var req struct {
		Theme       string `json:"theme"`
		Name        string `json:"name"`
		Description string `json:"description"`
	}
	_ = c.ShouldBindJSON(&req)

	fileURL, objectPath, size, err := renderAndUploadContextPDF(pack, req.Theme)
	if err != nil {
		log.Printf("[context-render] render failed: %v", err)
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}
	asset, err := makeContextDownloadAsset(tenantID, pack, fileURL, objectPath, size)
	if err != nil {
		log.Printf("[context-render] asset insert failed: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save asset"})
		return
	}

	product := pkgmodels.NewProduct()
	product.TenantID = tenantID
	product.Name = strings.TrimSpace(req.Name)
	if product.Name == "" {
		product.Name = pack.Name
	}
	product.Description = req.Description
	product.ProductType = pkgmodels.ProductTypeDigitalDownload
	product.Status = "draft"
	product.Downloads = &pkgmodels.DigitalDownloadConfig{
		AssetIDs: []bson.ObjectId{asset.Id},
	}
	if err := db.GetCollection(pkgmodels.ProductCollection).Insert(product); err != nil {
		log.Printf("[context-render] product insert failed: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create product"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"asset":   asset,
		"product": product,
	})
}
