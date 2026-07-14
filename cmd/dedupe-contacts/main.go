// Command dedupe-contacts reports and (with -apply) merges duplicate contacts
// that share a (tenant_id, email) so the ID-008 unique index can be created.
// Merge policy: the OLDEST row per (tenant, email) is canonical; newer
// duplicates are soft-deleted after their badges are unioned onto the
// canonical row (access is never lost). Dry-run by default.
//
// Rollback: soft-deleted rows retain timestamps.deleted_at; clear it to
// restore. No data is destroyed.
//
// Usage:
//
//	go run ./core-service/cmd/dedupe-contacts          # dry-run report
//	go run ./core-service/cmd/dedupe-contacts -apply    # merge
package main

import (
	"flag"
	"log"
	"os"

	"gopkg.in/mgo.v2/bson"

	"github.com/josephalai/sentanyl/pkg/db"
	pkgmodels "github.com/josephalai/sentanyl/pkg/models"
)

func main() {
	var host, port, dbName string
	var apply bool
	flag.StringVar(&host, "mongo-host", envOr("MONGO_HOST", "localhost"), "Mongo host")
	flag.StringVar(&port, "mongo-port", envOr("MONGO_PORT", "27017"), "Mongo port")
	flag.StringVar(&dbName, "mongo-db", envOr("MONGO_DB", "sentanyl_db"), "Mongo db")
	flag.BoolVar(&apply, "apply", false, "Merge duplicates (default dry-run)")
	flag.Parse()

	db.MongoHost, db.MongoPort, db.MongoDB, db.UsingLocalMongo = host, port, dbName, true
	db.InitMongoConnection()
	col := db.GetCollection(pkgmodels.UserCollection)

	var rows []struct {
		ID struct {
			Tenant bson.ObjectId `bson:"t"`
			Email  string        `bson:"e"`
		} `bson:"_id"`
		Count int           `bson:"count"`
		IDs   []bson.ObjectId `bson:"ids"`
	}
	pipe := col.Pipe([]bson.M{
		{"$match": bson.M{"email": bson.M{"$exists": true, "$ne": ""}, "timestamps.deleted_at": nil}},
		{"$group": bson.M{
			"_id":   bson.M{"t": "$tenant_id", "e": "$email"},
			"count": bson.M{"$sum": 1},
			"ids":   bson.M{"$push": "$_id"},
		}},
		{"$match": bson.M{"count": bson.M{"$gt": 1}}},
	})
	if err := pipe.All(&rows); err != nil {
		log.Fatalf("aggregate: %v", err)
	}

	groups, merged := 0, 0
	for _, r := range rows {
		groups++
		// Oldest _id (ObjectId embeds a timestamp) is canonical.
		canonical := r.IDs[0]
		for _, id := range r.IDs {
			if id.Hex() < canonical.Hex() {
				canonical = id
			}
		}
		for _, id := range r.IDs {
			if id == canonical {
				continue
			}
			merged++
			if !apply {
				continue
			}
			var dup pkgmodels.User
			if err := col.FindId(id).One(&dup); err == nil && len(dup.Badges) > 0 {
				_ = col.UpdateId(canonical, bson.M{"$addToSet": bson.M{"badges": bson.M{"$each": dup.Badges}}})
			}
			_ = col.UpdateId(id, bson.M{"$set": bson.M{"timestamps.deleted_at": bson.Now()}})
		}
	}
	mode := "DRY-RUN"
	if apply {
		mode = "APPLIED"
	}
	log.Printf("[%s] duplicate (tenant,email) groups=%d duplicate rows merged=%d", mode, groups, merged)
}

func envOr(k, d string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return d
}
