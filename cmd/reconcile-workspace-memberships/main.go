// Command reconcile-workspace-memberships backfills authoritative workspace
// memberships for legacy AccountUsers. Dry-run is the default. Applied rows are
// marked source=legacy, making rollback bounded and inspectable.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/josephalai/sentanyl/pkg/auth"
	"github.com/josephalai/sentanyl/pkg/db"
	"github.com/josephalai/sentanyl/pkg/models"
	"gopkg.in/mgo.v2/bson"
)

type report struct {
	Mode       string `json:"mode"`
	Scanned    int    `json:"scanned"`
	Existing   int    `json:"existing"`
	Backfilled int    `json:"backfilled"`
	Invalid    int    `json:"invalid"`
}

func main() {
	var host, port, dbName, tenantHex string
	var apply bool
	flag.StringVar(&host, "mongo-host", envOr("MONGO_HOST", "localhost"), "Mongo host")
	flag.StringVar(&port, "mongo-port", envOr("MONGO_PORT", "27017"), "Mongo port")
	flag.StringVar(&dbName, "mongo-db", envOr("MONGO_DB", "sentanyl_db"), "Mongo db")
	flag.StringVar(&tenantHex, "tenant", "", "optional tenant ObjectId scope")
	flag.BoolVar(&apply, "apply", false, "create missing legacy memberships")
	flag.Parse()
	db.MongoHost, db.MongoPort, db.MongoDB, db.UsingLocalMongo = host, port, dbName, true
	db.InitMongoConnection()
	auth.EnsureWorkspaceIndexes()
	query := bson.M{"timestamps.deleted_at": nil}
	if tenantHex != "" {
		if !bson.IsObjectIdHex(tenantHex) {
			log.Fatal("-tenant must be a 24-character ObjectId")
		}
		query["tenant_id"] = bson.ObjectIdHex(tenantHex)
	}
	var users []models.AccountUser
	if err := db.GetCollection(models.AccountUserCollection).Find(query).Sort("_id").All(&users); err != nil {
		log.Fatalf("scan identities: %v", err)
	}
	r := report{Mode: "dry-run", Scanned: len(users)}
	if apply {
		r.Mode = "applied"
	}
	for i := range users {
		user := &users[i]
		if user.TenantID == "" {
			r.Invalid++
			continue
		}
		n, err := db.GetCollection(models.WorkspaceMembershipCollection).Find(bson.M{
			"tenant_id": user.TenantID, "identity_id": user.Id,
		}).Count()
		if err != nil {
			log.Fatalf("inspect identity %s: %v", user.Id.Hex(), err)
		}
		if n > 0 {
			r.Existing++
			continue
		}
		r.Backfilled++
		if apply {
			if _, err := auth.EnsureLegacyWorkspaceMembership(user, user.TenantID); err != nil {
				log.Fatalf("backfill identity %s: %v", user.Id.Hex(), err)
			}
		}
	}
	out, _ := json.Marshal(r)
	fmt.Println(string(out))
	if r.Invalid > 0 {
		os.Exit(2)
	}
}

func envOr(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
