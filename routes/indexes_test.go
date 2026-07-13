package routes

import (
	"testing"

	pkgmodels "github.com/josephalai/sentanyl/pkg/models"
)

// ID-004: every tenant-owned story-graph collection that hydrateStory /
// hydrateStoryGraph resolve references from must carry the per-tenant compound
// unique index. This guards against a new entity being added to the hydration
// path without its (subscriber_id, public_id) uniqueness guarantee.
func TestStoryGraphIndexCoverage(t *testing.T) {
	required := []string{
		pkgmodels.StoryCollection,
		pkgmodels.StorylineCollection,
		pkgmodels.EnactmentCollection,
		pkgmodels.SceneCollection,
		pkgmodels.MessageCollection,
		pkgmodels.TriggerCollection,
		pkgmodels.ActionCollection,
		pkgmodels.BadgeCollection,
		pkgmodels.TagCollection,
		pkgmodels.UserCollection,
	}
	have := make(map[string]bool, len(storyGraphCollections))
	for _, c := range storyGraphCollections {
		have[c] = true
	}
	for _, c := range required {
		if !have[c] {
			t.Fatalf("collection %q resolved during hydration but missing from storyGraphCollections index set", c)
		}
	}
}
