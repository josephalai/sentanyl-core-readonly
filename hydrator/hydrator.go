// Package hydrator provides the AI content hydration background worker.
// It is a direct port of story/sentanyl/ai_hydrator.go adapted to use
// the pkg/db package and the ServiceBridge for cross-service HTTP calls.
package hydrator

import (
"bytes"
"context"
"encoding/json"
"fmt"
"io"
"log"
"net/http"
"os"
"strings"
"time"

"gopkg.in/mgo.v2/bson"

"github.com/josephalai/sentanyl/core-service/routes"
"github.com/josephalai/sentanyl/pkg/db"
pkgmodels "github.com/josephalai/sentanyl/pkg/models"
"github.com/josephalai/sentanyl/pkg/storage"
)

// HydratorInterval controls how often the AI hydrator checks for pending content.
var HydratorInterval = 30 * time.Second

// pdfServiceURL is the PDF microservice endpoint.
var pdfServiceURL = "http://localhost:9774"

// Hydrator coordinates background AI content generation jobs.
type Hydrator struct {
Bridge   *routes.ServiceBridge
Storage  storage.StorageProvider
GCSBucket string
}

// New creates a Hydrator that delegates cross-service writes via bridge and
// uploads generated PDFs (cert + funnel assets) to GCS via storeProvider.
// If storeProvider is nil, PDF jobs will fail with a clear "GCS not
// configured" error rather than silently writing to ephemeral disk.
func New(bridge *routes.ServiceBridge, storeProvider storage.StorageProvider, gcsBucket string) *Hydrator {
if u := os.Getenv("PDF_SERVICE_URL"); u != "" {
pdfServiceURL = u
}
return &Hydrator{Bridge: bridge, Storage: storeProvider, GCSBucket: gcsBucket}
}

// Start launches the AI content hydration background worker.
func (h *Hydrator) Start() {
log.Println("[Hydrator] Starting AI content hydration worker...")
go h.runHydrationWorker()
}

func (h *Hydrator) runHydrationWorker() {
ticker := time.NewTicker(HydratorInterval)
defer ticker.Stop()
for range ticker.C {
h.processPendingBlocks()
h.processPendingPDFs()
h.processPendingAssets()
h.processPendingMediaTranscripts()
h.processPendingChannelDescriptions()
h.processPendingLessonContent()
h.processPendingCourseDescriptions()
h.processPendingCertificates()
}
}

// ---------- Minimal local structs for reading documents from DB ----------

type contentGen struct {
Status       string   `bson:"status"`
Length       string   `bson:"length"`
PromptAppend string   `bson:"prompt_append"`
ContextURLs  []string `bson:"context_urls"`
}

type aiContext struct {
ContextURLs []string `bson:"context_urls"`
}

type pdfConfig struct {
Status    string     `bson:"status"`
AIContext *aiContext `bson:"ai_context"`
}

type genConfig struct {
Status      string   `bson:"status"`
AssetType   string   `bson:"asset_type"`
Instruction string   `bson:"instruction"`
References  []string `bson:"references"`
Theme       string   `bson:"theme"`
ErrorMsg    string   `bson:"error_msg"`
}

type pageBlock struct {
Id           bson.ObjectId `bson:"_id"`
PublicId     string        `bson:"public_id"`
SectionID    string        `bson:"section_id"`
ContentGen   *contentGen   `bson:"content_gen"`
AIContext    *aiContext     `bson:"ai_context"`
}

type funnelStage struct {
Id        bson.ObjectId `bson:"_id"`
PublicId  string        `bson:"public_id"`
Name      string        `bson:"name"`
PDFConfig *pdfConfig    `bson:"pdf_config"`
}

type asset struct {
Id        bson.ObjectId `bson:"_id"`
PublicId  string        `bson:"public_id"`
TenantID  bson.ObjectId `bson:"tenant_id"`
FileName  string        `bson:"file_name"`
GenConfig *genConfig    `bson:"gen_config"`
}

type mediaTranscript struct {
Status string `bson:"status"`
Text   string `bson:"text"`
}

type mediaItem struct {
Id           bson.ObjectId    `bson:"_id"`
PublicId     string           `bson:"public_id"`
Title        string           `bson:"title"`
Description  string           `bson:"description"`
Kind         string           `bson:"kind"`
DurationSec  int              `bson:"duration_sec"`
Transcript   *mediaTranscript `bson:"transcript"`
}

type mediaChannel struct {
Id     bson.ObjectId `bson:"_id"`
PublicId string      `bson:"public_id"`
Title  string        `bson:"title"`
Layout string        `bson:"layout"`
Items  []interface{} `bson:"items"`
}

type contentGenConfig struct {
Status      string   `bson:"status"`
Instruction string   `bson:"instruction"`
References  []string `bson:"references"`
ErrorMsg    string   `bson:"error_msg"`
}

type courseLesson struct {
Title            string            `bson:"title"`
Slug             string            `bson:"slug"`
Order            int               `bson:"order"`
ContentGenStatus string            `bson:"content_gen_status"`
ContentHTML      string            `bson:"content_html"`
ContentGenConfig *contentGenConfig `bson:"content_gen_config"`
}

type courseModule struct {
Title   string         `bson:"title"`
Order   int            `bson:"order"`
Lessons []courseLesson `bson:"lessons"`
}

type descGenConfig struct {
Status      string   `bson:"status"`
Instruction string   `bson:"instruction"`
References  []string `bson:"references"`
ErrorMsg    string   `bson:"error_msg"`
}

type lmsProduct struct {
Id                   bson.ObjectId  `bson:"_id"`
PublicId             string         `bson:"public_id"`
Name                 string         `bson:"name"`
Description          string         `bson:"description"`
InstructorName       string         `bson:"instructor_name"`
CourseModules        []courseModule `bson:"course_modules"`
DescriptionGenStatus string         `bson:"description_gen_status"`
DescriptionGenConfig *descGenConfig `bson:"description_gen_config"`
}

type certificate struct {
Id          bson.ObjectId `bson:"_id"`
PublicId    string        `bson:"public_id"`
TenantID    bson.ObjectId `bson:"tenant_id"`
ProductID   bson.ObjectId `bson:"product_id"`
ContactName string        `bson:"contact_name"`
CourseTitle string        `bson:"course_title"`
CompletedAt time.Time     `bson:"completed_at"`
Template    string        `bson:"template"`
GenStatus   string        `bson:"gen_status"`
Locale      string        `bson:"locale"`
}

// ---------- Processor functions ----------

func (h *Hydrator) processPendingBlocks() {
if db.Session == nil {
return
}
var blocks []pageBlock
err := db.GetCollection(pkgmodels.PageBlockCollection).Find(bson.M{
"content_gen.status":    "pending",
"timestamps.deleted_at": nil,
}).Limit(10).All(&blocks)
if err != nil || len(blocks) == 0 {
return
}
log.Printf("[Hydrator] Found %d pending blocks to hydrate", len(blocks))
for _, block := range blocks {
db.GetCollection(pkgmodels.PageBlockCollection).UpdateId(block.Id, bson.M{
"$set": bson.M{"content_gen.status": "processing"},
})
content, err := generateBlockContent(&block)
if err != nil {
log.Printf("[Hydrator] Failed to generate content for block %s: %v", block.PublicId, err)
db.GetCollection(pkgmodels.PageBlockCollection).UpdateId(block.Id, bson.M{
"$set": bson.M{
"content_gen.status": "failed",
"rendered_content":   fmt.Sprintf("<!-- generation failed: %v -->", err),
},
})
continue
}
now := time.Now()
db.GetCollection(pkgmodels.PageBlockCollection).UpdateId(block.Id, bson.M{
"$set": bson.M{
"content_gen.status":    "completed",
"rendered_content":      content,
"timestamps.updated_at": now,
},
})
log.Printf("[Hydrator] Successfully hydrated block %s", block.PublicId)
}
}

func (h *Hydrator) processPendingPDFs() {
if db.Session == nil {
return
}
var stages []funnelStage
err := db.GetCollection(pkgmodels.FunnelStageCollection).Find(bson.M{
"pdf_config.status":     "pending",
"timestamps.deleted_at": nil,
}).Limit(5).All(&stages)
if err != nil || len(stages) == 0 {
return
}
for _, stage := range stages {
db.GetCollection(pkgmodels.FunnelStageCollection).UpdateId(stage.Id, bson.M{
"$set": bson.M{"pdf_config.status": "processing"},
})
content, err := generatePDFContent(&stage)
if err != nil {
log.Printf("[Hydrator] Failed to generate PDF for stage %s: %v", stage.PublicId, err)
db.GetCollection(pkgmodels.FunnelStageCollection).UpdateId(stage.Id, bson.M{
"$set": bson.M{"pdf_config.status": "failed"},
})
continue
}
now := time.Now()
db.GetCollection(pkgmodels.FunnelStageCollection).UpdateId(stage.Id, bson.M{
"$set": bson.M{
"pdf_config.status":     "completed",
"pdf_config.file_url":   fmt.Sprintf("/generated/pdf_%s.pdf", stage.PublicId),
"timestamps.updated_at": now,
},
})
log.Printf("[Hydrator] Generated PDF content for stage %s (%d chars)", stage.PublicId, len(content))
}
}

func (h *Hydrator) processPendingAssets() {
if db.Session == nil {
return
}
var assets []asset
err := db.GetCollection(pkgmodels.AssetCollection).Find(bson.M{
"gen_config.status":     "pending",
"timestamps.deleted_at": nil,
}).Limit(5).All(&assets)
if err != nil || len(assets) == 0 {
return
}
log.Printf("[Hydrator] Found %d pending asset generation jobs", len(assets))
for _, a := range assets {
if a.GenConfig == nil {
continue
}
db.GetCollection(pkgmodels.AssetCollection).UpdateId(a.Id, bson.M{
"$set": bson.M{"gen_config.status": "processing"},
})
pdf, err := generateAsset(&a)
if err != nil {
log.Printf("[Hydrator] Failed to generate asset %s: %v", a.PublicId, err)
db.GetCollection(pkgmodels.AssetCollection).UpdateId(a.Id, bson.M{
"$set": bson.M{
"gen_config.status":   "failed",
"gen_config.error_msg": err.Error(),
},
})
continue
}
fileName := a.FileName
if fileName == "" {
fileName = a.GenConfig.AssetType + "-" + a.PublicId + ".pdf"
}
if h.Storage == nil {
db.GetCollection(pkgmodels.AssetCollection).UpdateId(a.Id, bson.M{
"$set": bson.M{
"gen_config.status":    "failed",
"gen_config.error_msg": "GCS not configured (set GCP_PROJECT_ID + GOOGLE_APPLICATION_CREDENTIALS)",
},
})
continue
}
objectPath := fmt.Sprintf("%s/assets/%s", a.TenantID.Hex(), fileName)
fileURL, upErr := h.Storage.UploadObject(h.GCSBucket, objectPath, "application/pdf", bytes.NewReader(pdf))
if upErr != nil {
db.GetCollection(pkgmodels.AssetCollection).UpdateId(a.Id, bson.M{
"$set": bson.M{
"gen_config.status":    "failed",
"gen_config.error_msg": "GCS upload failed: " + upErr.Error(),
},
})
continue
}
now := time.Now()
db.GetCollection(pkgmodels.AssetCollection).UpdateId(a.Id, bson.M{
"$set": bson.M{
"gen_config.status":     "completed",
"file_url":              fileURL,
"file_name":             fileName,
"file_type":             "application/pdf",
"file_size":             int64(len(pdf)),
"timestamps.updated_at": now,
},
})
}
}

func (h *Hydrator) processPendingMediaTranscripts() {
if db.Session == nil {
return
}
var items []mediaItem
err := db.GetCollection(pkgmodels.MediaCollection).Find(bson.M{
"transcript.status":     "pending",
"timestamps.deleted_at": nil,
}).Limit(5).All(&items)
if err != nil || len(items) == 0 {
return
}
log.Printf("[Hydrator] Found %d pending media transcripts", len(items))
for _, m := range items {
db.GetCollection(pkgmodels.MediaCollection).UpdateId(m.Id, bson.M{
"$set": bson.M{"transcript.status": "processing"},
})
summary, err := generateMediaSummary(&m)
if err != nil {
log.Printf("[Hydrator] Failed to generate transcript for %s: %v", m.PublicId, err)
db.GetCollection(pkgmodels.MediaCollection).UpdateId(m.Id, bson.M{
"$set": bson.M{"transcript.status": "failed"},
})
continue
}
now := time.Now()
db.GetCollection(pkgmodels.MediaCollection).UpdateId(m.Id, bson.M{
"$set": bson.M{
"transcript.status":     "ready",
"transcript.text":       summary,
"transcript.generated":  true,
"timestamps.updated_at": now,
},
})
}
}

func (h *Hydrator) processPendingChannelDescriptions() {
if db.Session == nil {
return
}
var channels []mediaChannel
err := db.GetCollection(pkgmodels.MediaChannelCollection).Find(bson.M{
"description_status":    "pending",
"timestamps.deleted_at": nil,
}).Limit(5).All(&channels)
if err != nil || len(channels) == 0 {
return
}
apiKey := getGeminiKey()
if apiKey == "" {
return
}
for _, ch := range channels {
db.GetCollection(pkgmodels.MediaChannelCollection).UpdateId(ch.Id, bson.M{
"$set": bson.M{"description_status": "processing"},
})
systemPrompt := "You are a content strategist for video platforms. Write engaging channel descriptions that help viewers understand what content the channel offers. Keep it to 2-3 paragraphs, professional but approachable. Output plain text, no HTML."
prompt := fmt.Sprintf("Write a description for a video channel titled %q", ch.Title)
if ch.Layout != "" {
prompt += fmt.Sprintf(" (layout: %s)", ch.Layout)
}
if len(ch.Items) > 0 {
prompt += fmt.Sprintf(" with %d videos", len(ch.Items))
}
description, err := callGeminiAPIWithSystem(apiKey, prompt, systemPrompt)
if err != nil {
log.Printf("[Hydrator] Failed channel description for %s: %v", ch.PublicId, err)
db.GetCollection(pkgmodels.MediaChannelCollection).UpdateId(ch.Id, bson.M{
"$set": bson.M{"description_status": "failed"},
})
continue
}
now := time.Now()
db.GetCollection(pkgmodels.MediaChannelCollection).UpdateId(ch.Id, bson.M{
"$set": bson.M{
"description":           description,
"description_status":    "completed",
"timestamps.updated_at": now,
},
})
}
}

func (h *Hydrator) processPendingLessonContent() {
if db.Session == nil {
return
}
var products []lmsProduct
err := db.GetCollection(pkgmodels.ProductCollection).Find(bson.M{
"course_modules.lessons.content_gen_status": "pending",
"timestamps.deleted_at":                     nil,
}).Limit(5).All(&products)
if err != nil || len(products) == 0 {
return
}
apiKey := getGeminiKey()
if apiKey == "" {
return
}
log.Printf("[Hydrator] Found %d products with pending lesson content", len(products))
for _, product := range products {
for mi, mod := range product.CourseModules {
for li, lesson := range mod.Lessons {
if lesson.ContentGenStatus != "pending" {
continue
}
updateLessonField(product.Id, mi, li, bson.M{"content_gen_status": "processing"})
userPrompt, systemPrompt := buildLessonPrompt(&product, &mod, &lesson)
if lesson.ContentGenConfig != nil && len(lesson.ContentGenConfig.References) > 0 {
contextText := fetchContextFromURLs(lesson.ContentGenConfig.References)
if contextText != "" {
userPrompt += fmt.Sprintf("\n\nReference material:\n%s", contextText)
}
}
content, err := callGeminiAPIWithSystem(apiKey, userPrompt, systemPrompt)
if err != nil {
log.Printf("[Hydrator] Failed lesson content for %s/%s: %v", product.PublicId, lesson.Slug, err)
updateLessonField(product.Id, mi, li, bson.M{
"content_gen_status":           "failed",
"content_gen_config.error_msg": err.Error(),
})
continue
}
content = stripUnsafeHTML(content)
updateLessonField(product.Id, mi, li, bson.M{
"content_html":       content,
"content_gen_status": "completed",
})
}
}
}
}

func (h *Hydrator) processPendingCourseDescriptions() {
if db.Session == nil {
return
}
var products []lmsProduct
err := db.GetCollection(pkgmodels.ProductCollection).Find(bson.M{
"description_gen_status": "pending",
"timestamps.deleted_at":  nil,
}).Limit(5).All(&products)
if err != nil || len(products) == 0 {
return
}
apiKey := getGeminiKey()
if apiKey == "" {
return
}
for _, product := range products {
db.GetCollection(pkgmodels.ProductCollection).UpdateId(product.Id, bson.M{
"$set": bson.M{"description_gen_status": "processing"},
})
userPrompt, systemPrompt := buildCourseDescriptionPrompt(&product)
if product.DescriptionGenConfig != nil && len(product.DescriptionGenConfig.References) > 0 {
contextText := fetchContextFromURLs(product.DescriptionGenConfig.References)
if contextText != "" {
userPrompt += fmt.Sprintf("\n\nReference material:\n%s", contextText)
}
}
description, err := callGeminiAPIWithSystem(apiKey, userPrompt, systemPrompt)
if err != nil {
log.Printf("[Hydrator] Failed course description for %s: %v", product.PublicId, err)
db.GetCollection(pkgmodels.ProductCollection).UpdateId(product.Id, bson.M{
"$set": bson.M{
"description_gen_status":           "failed",
"description_gen_config.error_msg": err.Error(),
},
})
continue
}
now := time.Now()
db.GetCollection(pkgmodels.ProductCollection).UpdateId(product.Id, bson.M{
"$set": bson.M{
"description":            description,
"description_gen_status": "completed",
"timestamps.updated_at":  now,
},
})
}
}

func (h *Hydrator) processPendingCertificates() {
if db.Session == nil {
return
}
var certs []certificate
err := db.GetCollection(pkgmodels.CertificateCollection).Find(bson.M{
"gen_status":            "pending",
"timestamps.deleted_at": nil,
}).Limit(5).All(&certs)
if err != nil || len(certs) == 0 {
return
}
apiKey := getGeminiKey()
log.Printf("[Hydrator] Found %d pending certificates (ai=%v)", len(certs), apiKey != "")
for _, cert := range certs {
now := time.Now()
db.GetCollection(pkgmodels.CertificateCollection).UpdateId(cert.Id, bson.M{
"$set": bson.M{"gen_status": "processing", "timestamps.updated_at": now},
})
var htmlContent string
if apiKey != "" {
userPrompt, systemPrompt := buildCertificatePrompt(&cert)
generated, genErr := callGeminiAPIWithSystem(apiKey, userPrompt, systemPrompt)
if genErr != nil {
log.Printf("[Hydrator] cert %s: AI failed (%v) — using deterministic template", cert.PublicId, genErr)
htmlContent = buildCertificateHTMLFallback(&cert)
} else {
htmlContent = generated
}
} else {
htmlContent = buildCertificateHTMLFallback(&cert)
}
htmlContent = stripUnsafeHTML(htmlContent)
pdfReqBody, _ := json.Marshal(map[string]string{
"html":  htmlContent,
"title": cert.CourseTitle,
})
pdfResp, err := http.Post(pdfServiceURL+"/from-html", "application/json", bytes.NewReader(pdfReqBody))
if err != nil {
db.GetCollection(pkgmodels.CertificateCollection).UpdateId(cert.Id, bson.M{
"$set": bson.M{"gen_status": "failed", "gen_error_msg": "PDF service error: " + err.Error()},
})
continue
}
if pdfResp.StatusCode != http.StatusOK {
errBody, _ := io.ReadAll(pdfResp.Body)
pdfResp.Body.Close()
db.GetCollection(pkgmodels.CertificateCollection).UpdateId(cert.Id, bson.M{
"$set": bson.M{"gen_status": "failed", "gen_error_msg": fmt.Sprintf("PDF service returned %d: %s", pdfResp.StatusCode, string(errBody))},
})
continue
}
pdfBytes, readErr := io.ReadAll(pdfResp.Body)
pdfResp.Body.Close()
if readErr != nil || len(pdfBytes) == 0 {
db.GetCollection(pkgmodels.CertificateCollection).UpdateId(cert.Id, bson.M{
"$set": bson.M{"gen_status": "failed", "gen_error_msg": "empty PDF body"},
})
continue
}
if h.Storage == nil {
db.GetCollection(pkgmodels.CertificateCollection).UpdateId(cert.Id, bson.M{
"$set": bson.M{"gen_status": "failed", "gen_error_msg": "GCS not configured (set GCP_PROJECT_ID + GOOGLE_APPLICATION_CREDENTIALS)"},
})
continue
}
objectPath := fmt.Sprintf("%s/certs/cert_%s.pdf", cert.TenantID.Hex(), cert.PublicId)
publicURL, upErr := h.Storage.UploadObject(h.GCSBucket, objectPath, "application/pdf", bytes.NewReader(pdfBytes))
if upErr != nil {
db.GetCollection(pkgmodels.CertificateCollection).UpdateId(cert.Id, bson.M{
"$set": bson.M{"gen_status": "failed", "gen_error_msg": "GCS upload failed: " + upErr.Error()},
})
continue
}
completedAt := time.Now()
db.GetCollection(pkgmodels.CertificateCollection).UpdateId(cert.Id, bson.M{
"$set": bson.M{
"asset_url":             publicURL,
"gen_status":            "completed",
"gen_error_msg":         "",
"timestamps.updated_at": completedAt,
},
})
}
}

// ---------- Helper functions ----------

func updateLessonField(productId bson.ObjectId, moduleIdx, lessonIdx int, fields bson.M) {
set := bson.M{}
for k, v := range fields {
set[fmt.Sprintf("course_modules.%d.lessons.%d.%s", moduleIdx, lessonIdx, k)] = v
}
db.GetCollection(pkgmodels.ProductCollection).UpdateId(productId, bson.M{"$set": set})
}

func getGeminiKey() string {
if k := os.Getenv("GEMINI_API_KEY"); k != "" {
return k
}
return ""
}

func stripUnsafeHTML(html string) string {
html = strings.ReplaceAll(html, "<script", "<!-- script")
html = strings.ReplaceAll(html, "</script>", "-->")
html = strings.ReplaceAll(html, "<form", "<!-- form")
html = strings.ReplaceAll(html, "</form>", "-->")
html = strings.ReplaceAll(html, "<iframe", "<!-- iframe")
html = strings.ReplaceAll(html, "</iframe>", "-->")
return html
}

func fetchContextFromURLs(urls []string) string {
if len(urls) == 0 {
return ""
}
var sb strings.Builder
client := &http.Client{Timeout: 15 * time.Second}
for _, url := range urls {
resp, err := client.Get(url)
if err != nil {
continue
}
body, err := io.ReadAll(io.LimitReader(resp.Body, 50000))
resp.Body.Close()
if err != nil {
continue
}
sb.WriteString(fmt.Sprintf("--- Source: %s ---\n%s\n\n", url, string(body)))
}
return sb.String()
}

func generateBlockContent(block *pageBlock) (string, error) {
apiKey := getGeminiKey()
if apiKey == "" {
return "", fmt.Errorf("GEMINI_API_KEY not configured")
}
var sb strings.Builder
sb.WriteString("Generate HTML content for a web page section.\n\n")
if block.SectionID != "" {
sb.WriteString(fmt.Sprintf("Section: %s\n", block.SectionID))
}
if block.ContentGen != nil {
if block.ContentGen.Length != "" {
sb.WriteString(fmt.Sprintf("Length: %s\n", block.ContentGen.Length))
}
if block.ContentGen.PromptAppend != "" {
sb.WriteString(fmt.Sprintf("Instructions: %s\n", block.ContentGen.PromptAppend))
}
}
var urls []string
if block.ContentGen != nil {
urls = append(urls, block.ContentGen.ContextURLs...)
}
if block.AIContext != nil {
urls = append(urls, block.AIContext.ContextURLs...)
}
if ctx := fetchContextFromURLs(urls); ctx != "" {
sb.WriteString(fmt.Sprintf("\nContext:\n%s\n", ctx))
}
sb.WriteString("\nRules:\n- Return ONLY the HTML content. No markdown, no code fences.\n- Use INLINE STYLES on every element. No CSS class names.\n- Do NOT generate <form>, <input>, or <button type=\"submit\"> elements.\n- Do NOT wrap in <html>, <head>, or <body> tags.")
return callGeminiAPI(apiKey, sb.String())
}

func generatePDFContent(stage *funnelStage) (string, error) {
apiKey := getGeminiKey()
if apiKey == "" {
return "", fmt.Errorf("GEMINI_API_KEY not configured")
}
var sb strings.Builder
sb.WriteString("Generate content for a PDF lead magnet/guide.\n\n")
sb.WriteString(fmt.Sprintf("Stage: %s\n", stage.Name))
if stage.PDFConfig != nil && stage.PDFConfig.AIContext != nil {
if ctx := fetchContextFromURLs(stage.PDFConfig.AIContext.ContextURLs); ctx != "" {
sb.WriteString(fmt.Sprintf("\nContext:\n%s\n", ctx))
}
}
sb.WriteString("\nReturn well-structured content suitable for a professional PDF guide. Use headers, bullet points, and clear sections.")
return callGeminiAPI(apiKey, sb.String())
}

func generateAsset(a *asset) ([]byte, error) {
apiKey := getGeminiKey()
if apiKey == "" {
return nil, fmt.Errorf("GEMINI_API_KEY not configured")
}
cfg := a.GenConfig
refText := fetchContextFromURLs(cfg.References)
var promptSB strings.Builder
promptSB.WriteString(fmt.Sprintf("Create a professional %s PDF document.\n\n", cfg.AssetType))
if cfg.Instruction != "" {
promptSB.WriteString(fmt.Sprintf("Instructions: %s\n\n", cfg.Instruction))
}
if refText != "" {
promptSB.WriteString(fmt.Sprintf("Use the following source material as context:\n\n%s\n\n", refText))
}
promptSB.WriteString("Format the output as clean markdown. Return ONLY the markdown content, no explanations or code fences.")
markdown, err := callGeminiAPIWithSystem(apiKey, promptSB.String(),
"You are an expert content creator specializing in high-value lead magnets, worksheets, and guides.")
if err != nil {
return nil, fmt.Errorf("gemini generation failed: %w", err)
}
title := cfg.AssetType
if cfg.Instruction != "" && len(cfg.Instruction) < 60 {
title = cfg.Instruction
}
type pdfReq struct {
Markdown string `json:"markdown"`
Title    string `json:"title"`
Theme    string `json:"theme"`
}
reqBody, _ := json.Marshal(pdfReq{Markdown: markdown, Title: title, Theme: cfg.Theme})
client := &http.Client{Timeout: 120 * time.Second}
resp, err := client.Post(pdfServiceURL+"/generate", "application/json", bytes.NewReader(reqBody))
if err != nil {
return nil, fmt.Errorf("pdf service unavailable: %w", err)
}
defer resp.Body.Close()
pdfBytes, err := io.ReadAll(resp.Body)
if err != nil {
return nil, err
}
if resp.StatusCode != 200 {
return nil, fmt.Errorf("pdf service error: %s", string(pdfBytes))
}
return pdfBytes, nil
}

func generateMediaSummary(m *mediaItem) (string, error) {
apiKey := getGeminiKey()
if apiKey == "" {
return "", fmt.Errorf("GEMINI_API_KEY not configured")
}
var sb strings.Builder
sb.WriteString("Generate a structured summary of the following media content.\n\n")
if m.Title != "" {
sb.WriteString(fmt.Sprintf("Title: %s\n", m.Title))
}
if m.Description != "" {
sb.WriteString(fmt.Sprintf("Description: %s\n", m.Description))
}
if m.Kind != "" {
sb.WriteString(fmt.Sprintf("Media Type: %s\n", m.Kind))
}
if m.DurationSec > 0 {
sb.WriteString(fmt.Sprintf("Duration: %d seconds\n", m.DurationSec))
}
sb.WriteString("\nOutput Rules:\n- Return ONLY clean markdown.\n- Structure: ## Overview, ## Key Topics, ## Chapters (if applicable).\n- Keep the tone professional.")
return callGeminiAPIWithSystem(apiKey, sb.String(),
"You are an expert content analyst. Produce clear, structured summaries of video and audio content. Output must be clean markdown.")
}

func buildLessonPrompt(product *lmsProduct, mod *courseModule, lesson *courseLesson) (string, string) {
systemPrompt := `You are an expert course content writer for online learning platforms.
You produce well-structured, engaging lesson content in clean HTML format.

Rules:
- Output ONLY the HTML body content (no <html>, <head>, or <body> tags)
- Use semantic HTML: <h2>, <h3>, <p>, <ul>, <ol>, <li>, <code>, <pre>, <blockquote>, <strong>, <em>
- Do NOT include <form>, <script>, <iframe>, <style>, <link> tags
- Target 800-2000 words depending on topic complexity`

instruction := ""
if lesson.ContentGenConfig != nil {
instruction = lesson.ContentGenConfig.Instruction
}
userPrompt := fmt.Sprintf(`Course: %s
Course Description: %s
Module: %s (Module %d of %d)
Lesson: %s (Lesson %d of %d in this module)

Lesson Generation Instructions: %s

Write the complete lesson content for this lesson.`,
product.Name, product.Description,
mod.Title, mod.Order, len(product.CourseModules),
lesson.Title, lesson.Order, len(mod.Lessons),
instruction,
)
return userPrompt, systemPrompt
}

func buildCourseDescriptionPrompt(product *lmsProduct) (string, string) {
systemPrompt := `You are a marketing copywriter for online course platforms.
Write compelling, clear course descriptions that convert browsers into students.

Rules:
- Write 2-4 paragraphs (150-400 words total)
- Use plain text with basic HTML formatting only: <p>, <strong>, <em>, <ul>, <li>
- Do NOT use <h1>-<h6> tags (the title is displayed separately)
- Do NOT include pricing, enrollment CTAs, or instructor bio`

var moduleSummary string
for _, m := range product.CourseModules {
moduleSummary += fmt.Sprintf("- %s (%d lessons)\n", m.Title, len(m.Lessons))
for _, l := range m.Lessons {
moduleSummary += fmt.Sprintf("  - %s\n", l.Title)
}
}
instruction := ""
if product.DescriptionGenConfig != nil {
instruction = product.DescriptionGenConfig.Instruction
}
userPrompt := fmt.Sprintf(`Course Title: %s
Instructor: %s

Course Structure:
%s

Generation Instructions: %s

Write the course description.`,
product.Name, product.InstructorName, moduleSummary, instruction,
)
return userPrompt, systemPrompt
}

// certStrings holds the user-facing UI strings used in cert templates +
// prompts. Built-in translations cover the locales the platform actively
// supports; unknown locales fall back to English. Add a locale by appending
// a row — keep keys in lockstep across all entries.
type certStrings struct {
EyebrowCompletion   string
LineCertifies       string
LineCompleted       string
LineOnDate          string
LabelCertificateID  string
PromptLanguageHint  string
}

var certStringTable = map[string]certStrings{
"en": {
EyebrowCompletion:  "Certificate of Completion",
LineCertifies:      "This certifies that",
LineCompleted:      "has successfully completed",
LineOnDate:         "on",
LabelCertificateID: "Certificate ID",
PromptLanguageHint: "Render all user-facing strings in English.",
},
"es": {
EyebrowCompletion:  "Certificado de Finalización",
LineCertifies:      "Esto certifica que",
LineCompleted:      "ha completado con éxito",
LineOnDate:         "el",
LabelCertificateID: "ID del certificado",
PromptLanguageHint: "Render all user-facing strings in Spanish.",
},
"pt": {
EyebrowCompletion:  "Certificado de Conclusão",
LineCertifies:      "Isto certifica que",
LineCompleted:      "concluiu com sucesso",
LineOnDate:         "em",
LabelCertificateID: "ID do certificado",
PromptLanguageHint: "Render all user-facing strings in Portuguese.",
},
"fr": {
EyebrowCompletion:  "Certificat de Réussite",
LineCertifies:      "Ceci certifie que",
LineCompleted:      "a complété avec succès",
LineOnDate:         "le",
LabelCertificateID: "Identifiant du certificat",
PromptLanguageHint: "Render all user-facing strings in French.",
},
"de": {
EyebrowCompletion:  "Abschlusszertifikat",
LineCertifies:      "Dies bestätigt, dass",
LineCompleted:      "erfolgreich abgeschlossen hat",
LineOnDate:         "am",
LabelCertificateID: "Zertifikat-ID",
PromptLanguageHint: "Render all user-facing strings in German.",
},
}

// pickCertStrings resolves the cert UI string set for a locale, with BCP-47
// fallback (es-MX → es → en).
func pickCertStrings(locale string) certStrings {
if s, ok := certStringTable[locale]; ok {
return s
}
if i := strings.Index(locale, "-"); i > 0 {
if s, ok := certStringTable[locale[:i]]; ok {
return s
}
}
return certStringTable["en"]
}

func buildCertificatePrompt(cert *certificate) (string, string) {
strs := pickCertStrings(cert.Locale)
systemPrompt := `You are a certificate designer for online learning platforms.
You produce elegant, professional certificate HTML suitable for PDF rendering.

Rules:
- Output a single-page certificate layout in clean HTML + inline CSS
- Use a formal, professional design with borders and elegant typography
- Page size: A4 landscape (297mm x 210mm)
- Do NOT include <script>, <form>, or interactive elements
- Do NOT use external fonts or images — use system fonts (Georgia, Times New Roman, Arial)
- ` + strs.PromptLanguageHint

completedDate := cert.CompletedAt.Format("January 2, 2006")
tmpl := cert.Template
if tmpl == "" {
tmpl = "default"
}
userPrompt := fmt.Sprintf(`Generate an HTML certificate with these details:

Recipient Name: %s
Course Title: %s
Completion Date: %s
Certificate ID: %s
Template Style: %s

The certificate should feel premium and worth framing.`,
cert.ContactName, cert.CourseTitle, completedDate, cert.PublicId, tmpl,
)
return userPrompt, systemPrompt
}

// buildCertificateHTMLFallback returns a deterministic A4-landscape HTML cert
// used when GEMINI_API_KEY is missing or the AI call fails. Produces the same
// shape every time so e2e tests can verify a real PDF gets rendered without an
// API key. UI strings come from pickCertStrings(cert.Locale).
func buildCertificateHTMLFallback(cert *certificate) string {
strs := pickCertStrings(cert.Locale)
completedDate := cert.CompletedAt.Format("January 2, 2006")
recipient := strings.TrimSpace(cert.ContactName)
if recipient == "" {
recipient = "Student"
}
title := strings.TrimSpace(cert.CourseTitle)
if title == "" {
title = "Course Completion"
}
return fmt.Sprintf(`<!DOCTYPE html>
<html><head><meta charset="utf-8"><title>Certificate</title>
<style>
@page { size: A4 landscape; margin: 0; }
body { font-family: Georgia, "Times New Roman", serif; color: #1a1a2e; margin: 0; padding: 0; }
.frame { width: 277mm; height: 190mm; margin: 10mm; padding: 18mm; border: 4px double #0f3460; text-align: center; box-sizing: border-box; }
.eyebrow { font-size: 14pt; letter-spacing: 6px; text-transform: uppercase; color: #0f3460; margin-bottom: 18mm; }
.recipient { font-size: 36pt; font-weight: 600; margin: 6mm 0; }
.line { font-size: 13pt; color: #444; margin: 4mm 0; }
.course { font-size: 22pt; font-style: italic; margin: 8mm 0; color: #0f3460; }
.meta { margin-top: 18mm; font-size: 10pt; color: #555; display: flex; justify-content: space-between; }
</style></head>
<body><div class="frame">
<div class="eyebrow">%s</div>
<div class="line">%s</div>
<div class="recipient">%s</div>
<div class="line">%s</div>
<div class="course">%s</div>
<div class="line">%s %s</div>
<div class="meta"><span>%s: %s</span><span>%s</span></div>
</div></body></html>`,
escapeHTMLBasic(strs.EyebrowCompletion),
escapeHTMLBasic(strs.LineCertifies),
escapeHTMLBasic(recipient),
escapeHTMLBasic(strs.LineCompleted),
escapeHTMLBasic(title),
escapeHTMLBasic(strs.LineOnDate), completedDate,
escapeHTMLBasic(strs.LabelCertificateID), cert.PublicId, completedDate)
}

func escapeHTMLBasic(s string) string {
s = strings.ReplaceAll(s, "&", "&amp;")
s = strings.ReplaceAll(s, "<", "&lt;")
s = strings.ReplaceAll(s, ">", "&gt;")
s = strings.ReplaceAll(s, "\"", "&quot;")
return s
}

func callGeminiAPI(apiKey, prompt string) (string, error) {
return callGeminiAPIWithSystem(apiKey, prompt,
"You are an expert web content generator. Generate clean, professional HTML content for marketing pages. Focus on high-converting copy that drives action.")
}

func callGeminiAPIWithSystem(apiKey, prompt, systemPrompt string) (string, error) {
modelName := "gemini-2.5-flash"
geminiReq := map[string]interface{}{
"system_instruction": map[string]interface{}{
"parts": []map[string]string{{"text": systemPrompt}},
},
"contents": []map[string]interface{}{
{"role": "user", "parts": []map[string]string{{"text": prompt}}},
},
"generationConfig": map[string]interface{}{
"temperature":     0.7,
"maxOutputTokens": 8192,
},
}
body, err := json.Marshal(geminiReq)
if err != nil {
return "", err
}
apiURL := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/%s:generateContent?key=%s", modelName, apiKey)
ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
defer cancel()
req, err := http.NewRequestWithContext(ctx, http.MethodPost, apiURL, bytes.NewReader(body))
if err != nil {
return "", err
}
req.Header.Set("Content-Type", "application/json")
resp, err := http.DefaultClient.Do(req)
if err != nil {
return "", fmt.Errorf("gemini API call failed: %w", err)
}
defer resp.Body.Close()
respBody, err := io.ReadAll(resp.Body)
if err != nil {
return "", err
}
if resp.StatusCode != 200 {
return "", fmt.Errorf("gemini API returned status %d: %s", resp.StatusCode, string(respBody))
}
var geminiResp struct {
Candidates []struct {
Content struct {
Parts []struct {
Text    string `json:"text"`
Thought bool   `json:"thought,omitempty"`
} `json:"parts"`
} `json:"content"`
} `json:"candidates"`
}
if err := json.Unmarshal(respBody, &geminiResp); err != nil {
return "", err
}
var result strings.Builder
if len(geminiResp.Candidates) > 0 {
for _, part := range geminiResp.Candidates[0].Content.Parts {
if !part.Thought {
result.WriteString(part.Text)
}
}
}
if result.Len() == 0 {
return "", fmt.Errorf("empty response from Gemini")
}
return result.String(), nil
}
