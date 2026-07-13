package routes

import (
	"testing"

	"gopkg.in/mgo.v2/bson"

	"github.com/josephalai/sentanyl/pkg/db"
)

// ID-002 tenant resolution (JWT → API key → verified domain/channel) is
// DB-backed and is exercised end-to-end by the lifecycle harness. These unit
// tests cover the DB-free security invariants of the ID-002/ID-003 slice.

// ID-003: the contact update allowlist must strip identity, tenancy, security,
// and engine-owned fields even when a caller includes them.
func TestContactUpdateAllowlist_StripsProtectedFields(t *testing.T) {
	attackerTenant := bson.NewObjectId().Hex()
	set, err := db.SanitizeUpdate(bson.M{
		"name":          bson.M{"first_name": "Jane"},
		"email":         "jane@example.com",
		"subscriber_id": attackerTenant,
		"tenant_id":     attackerTenant,
		"badges":        []string{"vip"},
		"password_hash": "x",
		"story_status":  "active",
	}, contactUpdateFields)
	if err != nil {
		t.Fatalf("sanitize: %v", err)
	}
	for _, forbidden := range []string{"subscriber_id", "tenant_id", "badges", "password_hash", "story_status"} {
		if _, present := set[forbidden]; present {
			t.Fatalf("protected field %q survived sanitization", forbidden)
		}
	}
	if _, ok := set["name"]; !ok {
		t.Fatal("legitimate field name was dropped")
	}
	if _, ok := set["email"]; !ok {
		t.Fatal("legitimate field email was dropped")
	}
}

// ID-003: operator injection ($-prefixed / dotted keys) in an update body is
// rejected outright for every story-graph entity allowlist.
func TestUpdateAllowlists_RejectOperatorInjection(t *testing.T) {
	lists := map[string][]string{
		"story":     storyUpdateFields,
		"storyline": storylineUpdateFields,
		"enactment": enactmentUpdateFields,
		"scene":     sceneUpdateFields,
		"trigger":   triggerUpdateFields,
		"action":    actionUpdateFields,
		"badge":     badgeUpdateFields,
		"tag":       tagUpdateFields,
		"contact":   contactUpdateFields,
	}
	for name, fields := range lists {
		if _, err := db.SanitizeUpdate(bson.M{"$set": bson.M{"x": 1}}, fields); err == nil {
			t.Fatalf("%s: operator key $set was not rejected", name)
		}
	}
}
