package routes

import (
	"log"

	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"

	"github.com/josephalai/sentanyl/pkg/db"
	pkgmodels "github.com/josephalai/sentanyl/pkg/models"
)

// storyGraphCollections are the tenant-owned story-builder collections whose
// (subscriber_id, public_id) pair must be unique per tenant. Public IDs are
// external identifiers, not an authorization boundary; the compound unique
// index makes cross-tenant reference collisions impossible at the storage
// layer (ID-004).
var storyGraphCollections = []string{
	pkgmodels.StoryCollection,
	pkgmodels.StorylineCollection,
	pkgmodels.EnactmentCollection,
	pkgmodels.SceneCollection,
	pkgmodels.MessageCollection,
	pkgmodels.MessageContentCollection,
	pkgmodels.TriggerCollection,
	pkgmodels.ActionCollection,
	pkgmodels.BadgeCollection,
	pkgmodels.TagCollection,
	pkgmodels.TemplateVariablesCollection,
	pkgmodels.UserCollection,
}

// EnsureStoryGraphIndexes creates the per-tenant compound unique indexes for
// the story-builder collections. It is safe to call at startup: mgo's
// EnsureIndex is idempotent.
//
// If a collection already holds duplicate (subscriber_id, public_id) rows the
// unique index creation fails. Rather than crashing the service, it reports the
// offending collection (with a small sample) and falls back to a non-unique
// compound index so lookups stay fast while an operator resolves the data. A
// later re-run promotes it to unique once duplicates are cleared.
func EnsureStoryGraphIndexes() {
	for _, coll := range storyGraphCollections {
		col := db.GetCollection(coll)
		unique := mgo.Index{
			Key:        []string{"subscriber_id", "public_id"},
			Unique:     true,
			Background: true,
			Sparse:     true,
		}
		if err := col.EnsureIndex(unique); err != nil {
			log.Printf("indexes: %s: unique (subscriber_id, public_id) failed (%v); reporting duplicates and falling back to non-unique", coll, err)
			reportDuplicatePublicIDs(coll)
			nonUnique := mgo.Index{
				Key:        []string{"subscriber_id", "public_id"},
				Background: true,
			}
			if err2 := col.EnsureIndex(nonUnique); err2 != nil {
				log.Printf("indexes: %s: non-unique fallback also failed: %v", coll, err2)
			}
		}
	}
	log.Println("indexes: story-graph per-tenant indexes ensured")
}

// reportDuplicatePublicIDs logs a small sample of (subscriber_id, public_id)
// groups that have more than one live document, so an operator can reconcile
// before the unique index is promoted.
func reportDuplicatePublicIDs(coll string) {
	var dupes []struct {
		ID struct {
			Subscriber string `bson:"subscriber_id"`
			Public     string `bson:"public_id"`
		} `bson:"_id"`
		Count int `bson:"count"`
	}
	pipeline := []bson.M{
		{"$group": bson.M{
			"_id":   bson.M{"subscriber_id": "$subscriber_id", "public_id": "$public_id"},
			"count": bson.M{"$sum": 1},
		}},
		{"$match": bson.M{"count": bson.M{"$gt": 1}}},
		{"$limit": 10},
	}
	if err := db.GetCollection(coll).Pipe(pipeline).All(&dupes); err != nil {
		log.Printf("indexes: %s: could not enumerate duplicates: %v", coll, err)
		return
	}
	for _, d := range dupes {
		log.Printf("indexes: %s: duplicate public_id %q in tenant %q (%d rows)", coll, d.ID.Public, d.ID.Subscriber, d.Count)
	}
}

// EnsureIdentityIndexes creates the identity invariants:
//   - account_users.email unique (ID-010): concurrent duplicate signups can't
//     both land; the register saga rolls the tenant back on the dup.
// Falls back to non-unique + a loud log if legacy duplicates exist, so an
// operator can dedupe and re-run rather than the service failing to start.
func EnsureIdentityIndexes() {
	col := db.GetCollection(pkgmodels.AccountUserCollection)
	if err := col.EnsureIndex(mgo.Index{
		Key:        []string{"email"},
		Unique:     true,
		Background: true,
	}); err != nil {
		log.Printf("identity: account_users.email unique index failed (likely legacy duplicates): %v — creating non-unique fallback; dedupe and re-run", err)
		_ = col.EnsureIndex(mgo.Index{Key: []string{"email"}, Background: true})
	}

	// ID-008: contacts are unique per (tenant, email) — the authoritative
	// reconciliation key. Partial (email present) + unique; falls back to
	// non-unique when legacy duplicates exist (run cmd/dedupe-contacts first).
	contacts := db.GetCollection(pkgmodels.UserCollection)
	if err := contacts.EnsureIndex(mgo.Index{
		Key:        []string{"tenant_id", "email"},
		Unique:     true,
		Sparse:     true,
		Background: true,
	}); err != nil {
		log.Printf("identity: contacts (tenant_id,email) unique index failed (legacy duplicates): %v — non-unique fallback; run cmd/dedupe-contacts and re-run", err)
		_ = contacts.EnsureIndex(mgo.Index{Key: []string{"tenant_id", "email"}, Background: true})
	}
}
