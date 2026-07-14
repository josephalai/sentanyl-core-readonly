package routes

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"gopkg.in/mgo.v2/bson"

	"github.com/josephalai/sentanyl/pkg/auth"
	"github.com/josephalai/sentanyl/pkg/db"
)

// Per-entity update allowlists (ID-003). Only these client-editable fields may
// flow into a $set; identity, tenancy, ownership, security, and engine state
// are server-owned and stripped by db.SanitizeUpdate regardless of this list.
var (
	storyUpdateFields = []string{
		"name", "storyline_ids", "priority", "allow_interruption",
		"on_complete", "on_fail", "on_begin", "must_have", "must_not_have",
		"required_user_badges", "start_trigger", "complete_trigger",
		"next_story", "next_story_id",
	}
	storylineUpdateFields = []string{
		"name", "natural_order", "act_ids", "must_have", "must_not_have",
		"required_user_badges", "badge_transaction", "next_storyline",
		"next_storyline_id", "conditional_routes", "on_complete_begin",
		"on_fail_begin", "on_begin",
	}
	enactmentUpdateFields = []string{
		"name", "level", "natural_order", "badge_transaction",
		"next_storyline_id", "send_scene", "send_scenes_ids",
		"skip_storyline_on_expiry", "trigger_ids",
	}
	sceneUpdateFields = []string{
		"name", "message_id", "tags_ids", "subject", "body",
		"from_email", "from_name", "reply_to", "vars",
	}
	messageUpdateFields = []string{"name", "content", "vars"}

	messageContentUpdateFields = []string{
		"subject", "reply_to", "from_email", "from_name", "body",
		"template_id", "given_vars", "email_gen_config",
	}
	triggerUpdateFields = []string{
		"name", "trigger_type", "user_action_type", "user_action_value",
		"then_do_this_action_id", "priority", "mark_complete", "mark_failed",
		"persist_scope", "must_have", "must_not_have", "required_badges",
		"watch_block_id", "watch_operator", "watch_percent",
	}
	actionUpdateFields = []string{
		"action_name", "when", "next_enactment_id", "end_story",
		"advance_to_next_storyline", "send_immediate", "badge_transaction_ids",
		"unsubscribe", "extra_actions",
	}
	badgeUpdateFields    = []string{"name", "description", "kind"}
	tagUpdateFields      = []string{"name", "description"}
	templateVarFields    = []string{"name", "value", "description", "default_value"}
	contactUpdateFields  = []string{
		"name", "email", "phone_number", "personal_tags", "custom_fields",
		"preferred_locale", "subscribed", "email_list", "middle_name",
		"first_name", "last_name", "phone",
	}
)

// applySanitizedUpdate binds the request body as a generic document, filters it
// through the entity allowlist, and performs the tenant-scoped $set. It writes
// the HTTP response on error and returns false so callers can stop.
func applySanitizedUpdate(c *gin.Context, collection string, allowed []string) bool {
	var raw bson.M
	if err := c.ShouldBindJSON(&raw); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return false
	}
	set, err := db.SanitizeUpdate(raw, allowed)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return false
	}
	if err := db.GetCollection(collection).Update(
		bson.M{"public_id": c.Param("id"), "subscriber_id": auth.GetTenantID(c)},
		bson.M{"$set": set},
	); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return false
	}
	return true
}
