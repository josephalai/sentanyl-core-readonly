// Package hydrator provides stub functions for the AI hydration worker.
// In the monolith this was ai_hydrator.go with direct DB writes to
// LMS/Funnel collections. In the macro-service architecture these writes
// are delegated to lms-service and marketing-service via the ServiceBridge.
package hydrator

import (
	"log"

	"github.com/josephalai/sentanyl/core-service/routes"
)

// Hydrator coordinates background AI content generation jobs.
type Hydrator struct {
	Bridge *routes.ServiceBridge
}

// New creates a Hydrator that delegates cross-service writes via bridge.
func New(bridge *routes.ServiceBridge) *Hydrator {
	return &Hydrator{Bridge: bridge}
}

// Start begins the hydration worker loop. This is a stub that will be
// filled in when the full AI pipeline is migrated.
func (h *Hydrator) Start() {
	log.Println("hydrator: worker started (stub — no-op until full migration)")
}

// ProcessPendingBlocks is a stub for AI block content generation.
func (h *Hydrator) ProcessPendingBlocks() error {
	log.Println("hydrator: ProcessPendingBlocks (stub)")
	return nil
}

// ProcessPendingPDFs is a stub for AI PDF generation.
func (h *Hydrator) ProcessPendingPDFs() error {
	log.Println("hydrator: ProcessPendingPDFs (stub)")
	return nil
}

// ProcessPendingAssets is a stub for AI asset generation.
func (h *Hydrator) ProcessPendingAssets() error {
	log.Println("hydrator: ProcessPendingAssets (stub)")
	return nil
}

// ProcessPendingLessonContent is a stub for AI lesson content generation.
// In the full implementation, this will use h.Bridge.HydrateLMS() instead
// of direct DB writes.
func (h *Hydrator) ProcessPendingLessonContent() error {
	log.Println("hydrator: ProcessPendingLessonContent (stub)")
	return nil
}

// ProcessPendingCourseDescriptions is a stub for AI course description generation.
func (h *Hydrator) ProcessPendingCourseDescriptions() error {
	log.Println("hydrator: ProcessPendingCourseDescriptions (stub)")
	return nil
}

// ProcessPendingCertificates is a stub for AI certificate generation.
func (h *Hydrator) ProcessPendingCertificates() error {
	log.Println("hydrator: ProcessPendingCertificates (stub)")
	return nil
}
