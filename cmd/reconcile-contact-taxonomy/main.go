// Command reconcile-contact-taxonomy migrates legacy Tag definitions and
// User.tags references into Badge(kind=contact_label) plus BadgeAssignment
// provenance. Dry-run is the default; ambiguous public-id collisions are
// reported and never guessed.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/josephalai/sentanyl/pkg/badges"
	"github.com/josephalai/sentanyl/pkg/db"
	"github.com/josephalai/sentanyl/pkg/models"
	"gopkg.in/mgo.v2/bson"
)

type report struct {
	Mode          string `json:"mode"`
	Definitions   int    `json:"definitions"`
	LabelsCreated int    `json:"labels_created"`
	ContactRefs   int    `json:"contact_refs"`
	Retired       int    `json:"retired"`
	Canonical     int    `json:"canonical"`
	Conflicts     int    `json:"conflicts"`
}

func main() {
	var host, port, dbName, tenantHex string
	var apply bool
	flag.StringVar(&host, "mongo-host", envOr("MONGO_HOST", "localhost"), "Mongo host")
	flag.StringVar(&port, "mongo-port", envOr("MONGO_PORT", "27017"), "Mongo port")
	flag.StringVar(&dbName, "mongo-db", envOr("MONGO_DB", "sentanyl_db"), "Mongo db")
	flag.StringVar(&tenantHex, "tenant", "", "optional tenant ObjectId scope")
	flag.BoolVar(&apply, "apply", false, "apply taxonomy migration")
	flag.Parse()
	db.MongoHost, db.MongoPort, db.MongoDB, db.UsingLocalMongo = host, port, dbName, true
	db.InitMongoConnection()
	if apply {
		badges.EnsureIndexes()
	}
	query := bson.M{}
	if tenantHex != "" {
		if !bson.IsObjectIdHex(tenantHex) {
			log.Fatal("-tenant must be a 24-character ObjectId")
		}
		query["$or"] = []bson.M{{"tenant_id": bson.ObjectIdHex(tenantHex)}, {"subscriber_id": tenantHex}}
	}
	var tags []models.Tag
	if err := db.GetCollection(models.TagCollection).Find(query).Sort("_id").All(&tags); err != nil {
		log.Fatalf("scan tags: %v", err)
	}
	r := report{Mode: "dry-run", Definitions: len(tags)}
	if apply {
		r.Mode = "applied"
	}
	for _, tag := range tags {
		tenantID := tag.TenantID
		if tenantID == "" && bson.IsObjectIdHex(tag.SubscriberId) {
			tenantID = bson.ObjectIdHex(tag.SubscriberId)
		}
		if tenantID == "" {
			r.Conflicts++
			continue
		}
		var label models.Badge
		err := db.GetCollection(models.BadgeCollection).Find(bson.M{"legacy_tag_id": tag.Id}).One(&label)
		if err != nil {
			err = db.GetCollection(models.BadgeCollection).Find(bson.M{"tenant_id": tenantID, "public_id": tag.PublicId}).One(&label)
			if err == nil && models.NormalizeBadgeKind(label.Kind) != models.BadgeKindContactLabel {
				r.Conflicts++
				continue
			}
			if err == nil && label.LegacyTagID != "" && label.LegacyTagID != tag.Id {
				r.Conflicts++
				continue
			}
		}
		if err != nil {
			r.LabelsCreated++
			label = models.Badge{
				Id: bson.NewObjectId(), PublicId: tag.PublicId, TenantID: tenantID, SubscriberId: tenantID.Hex(),
				Name: tag.Name, Description: tag.Description, Kind: models.BadgeKindContactLabel, LegacyTagID: tag.Id,
			}
			now := time.Now().UTC()
			label.SoftDeletes.CreatedAt = &now
			if apply {
				if err := db.GetCollection(models.BadgeCollection).Insert(&label); err != nil {
					log.Fatalf("create label for tag %s: %v", tag.Id.Hex(), err)
				}
			}
		} else if apply && label.LegacyTagID == "" {
			if err := db.GetCollection(models.BadgeCollection).UpdateId(label.Id, bson.M{"$set": bson.M{
				"legacy_tag_id": tag.Id, "kind": models.BadgeKindContactLabel, "tenant_id": tenantID,
			}}); err != nil {
				log.Fatalf("link label %s: %v", label.Id.Hex(), err)
			}
		}
		var contacts []struct {
			ID bson.ObjectId `bson:"_id"`
		}
		if err := db.GetCollection(models.UserCollection).Find(bson.M{"tenant_id": tenantID, "tags.tag": tag.Id}).Select(bson.M{"_id": 1}).All(&contacts); err != nil {
			log.Fatalf("scan tag references %s: %v", tag.Id.Hex(), err)
		}
		r.ContactRefs += len(contacts)
		if apply {
			for _, contact := range contacts {
				if _, err := badges.Assign(tenantID, contact.ID, label.Id, "taxonomy_migration", tag.Id.Hex(), "migration"); err != nil {
					log.Fatalf("assign label %s to %s: %v", label.Id.Hex(), contact.ID.Hex(), err)
				}
				_ = db.GetCollection(models.UserCollection).UpdateId(contact.ID, bson.M{"$pull": bson.M{"tags": bson.M{"tag": tag.Id}}})
			}
		}
		if tag.MigratedToBadgeID == label.Id && tag.DeletedAt != nil {
			r.Canonical++
			continue
		}
		r.Retired++
		if apply {
			now := time.Now().UTC()
			if err := db.GetCollection(models.TagCollection).UpdateId(tag.Id, bson.M{"$set": bson.M{
				"tenant_id": tenantID, "migrated_to_badge_id": label.Id, "timestamps.deleted_at": now,
			}}); err != nil {
				log.Fatalf("retire tag %s: %v", tag.Id.Hex(), err)
			}
		}
	}
	out, _ := json.Marshal(r)
	fmt.Println(string(out))
	if r.Conflicts > 0 {
		os.Exit(2)
	}
}

func envOr(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
