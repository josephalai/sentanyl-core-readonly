// Command reconcile-contact-tenancy makes tenant_id the sole ownership field
// on Contact (users) documents. It is a dry-run by default. With -apply it:
//
//   - backfills a missing tenant_id from a valid legacy subscriber_id;
//   - removes subscriber_id when it agrees with tenant_id; and
//   - refuses ambiguous/mismatched rows instead of guessing ownership.
//
// Every write uses the values observed during the scan as a compare-and-swap
// predicate, so a concurrent contact update cannot be overwritten. Rollback is
// deterministic: subscriber_id can be regenerated as tenant_id.Hex(); no
// ownership information is destroyed.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/josephalai/sentanyl/pkg/db"
	"github.com/josephalai/sentanyl/pkg/models"
	mgo "gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

type report struct {
	Mode       string `json:"mode"`
	Scanned    int    `json:"scanned"`
	Backfilled int    `json:"backfilled"`
	Retired    int    `json:"retired"`
	Canonical  int    `json:"canonical"`
	Conflicts  int    `json:"conflicts"`
	Raced      int    `json:"raced"`
}

func main() {
	var host, port, dbName, tenantHex string
	var apply bool
	flag.StringVar(&host, "mongo-host", envOr("MONGO_HOST", "localhost"), "Mongo host")
	flag.StringVar(&port, "mongo-port", envOr("MONGO_PORT", "27017"), "Mongo port")
	flag.StringVar(&dbName, "mongo-db", envOr("MONGO_DB", "sentanyl_db"), "Mongo db")
	flag.StringVar(&tenantHex, "tenant", "", "optional tenant ObjectId scope")
	flag.BoolVar(&apply, "apply", false, "apply safe backfill/retirement writes")
	flag.Parse()

	db.MongoHost, db.MongoPort, db.MongoDB, db.UsingLocalMongo = host, port, dbName, true
	db.InitMongoConnection()
	col := db.GetCollection(models.UserCollection)
	query := bson.M{}
	if tenantHex != "" {
		if !bson.IsObjectIdHex(tenantHex) {
			log.Fatal("-tenant must be a 24-character ObjectId")
		}
		tenantID := bson.ObjectIdHex(tenantHex)
		query["$or"] = []bson.M{{"tenant_id": tenantID}, {"subscriber_id": tenantHex}}
	}
	var rows []struct {
		ID           bson.ObjectId `bson:"_id"`
		TenantID     bson.ObjectId `bson:"tenant_id"`
		SubscriberID string        `bson:"subscriber_id"`
	}
	if err := col.Find(query).Select(bson.M{"_id": 1, "tenant_id": 1, "subscriber_id": 1}).All(&rows); err != nil {
		log.Fatalf("scan contacts: %v", err)
	}

	r := report{Mode: "dry-run", Scanned: len(rows)}
	if apply {
		r.Mode = "applied"
	}
	for _, row := range rows {
		legacyValid := bson.IsObjectIdHex(row.SubscriberID)
		if row.TenantID == "" {
			if !legacyValid {
				r.Conflicts++
				continue
			}
			r.Backfilled++
			if apply {
				err := col.Update(bson.M{"_id": row.ID, "tenant_id": bson.M{"$exists": false}, "subscriber_id": row.SubscriberID}, bson.M{
					"$set": bson.M{"tenant_id": bson.ObjectIdHex(row.SubscriberID)}, "$unset": bson.M{"subscriber_id": ""},
				})
				if err == mgo.ErrNotFound {
					r.Raced++
				} else if err != nil {
					log.Fatalf("backfill %s: %v", row.ID.Hex(), err)
				}
			}
			continue
		}
		if row.SubscriberID == "" {
			r.Canonical++
			continue
		}
		if !legacyValid || bson.ObjectIdHex(row.SubscriberID) != row.TenantID {
			r.Conflicts++
			continue
		}
		r.Retired++
		if apply {
			err := col.Update(bson.M{"_id": row.ID, "tenant_id": row.TenantID, "subscriber_id": row.SubscriberID}, bson.M{"$unset": bson.M{"subscriber_id": ""}})
			if err == mgo.ErrNotFound {
				r.Raced++
			} else if err != nil {
				log.Fatalf("retire %s: %v", row.ID.Hex(), err)
			}
		}
	}
	out, _ := json.Marshal(r)
	fmt.Println(string(out))
	if r.Conflicts > 0 {
		os.Exit(2)
	}
}

func envOr(k, fallback string) string {
	if value := os.Getenv(k); value != "" {
		return value
	}
	return fallback
}
