package routes

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"gopkg.in/mgo.v2/bson"

	"github.com/josephalai/sentanyl/pkg/audit"
	"github.com/josephalai/sentanyl/pkg/auth"
	"github.com/josephalai/sentanyl/pkg/jobs"
)

// Operator job console (OPS-002): owner-gated read + replay over the durable
// jobs kernel's dead-letter queue. Surfaces what the kernel already tracks —
// status counts and dead jobs — and lets an operator replay a dead job after
// a fix. These are platform-operator actions, gated PermDataDestroy (owner).

// HandleOpsJobOverview returns a status→count summary across all jobs.
func HandleOpsJobOverview(c *gin.Context) {
	counts, err := jobs.StatusCounts()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to read job stats"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status_counts": counts})
}

// HandleOpsDeadLetters lists dead-lettered jobs (exhausted retries) for
// operator inspection.
func HandleOpsDeadLetters(c *gin.Context) {
	dead, err := jobs.DeadLettered(200)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to read dead jobs"})
		return
	}
	if dead == nil {
		dead = []jobs.Job{}
	}
	c.JSON(http.StatusOK, gin.H{"dead_jobs": dead})
}

// HandleOpsReplayJob resets a dead job to pending so the worker retries it.
func HandleOpsReplayJob(c *gin.Context) {
	id := c.Param("id")
	if !bson.IsObjectIdHex(id) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid job id"})
		return
	}
	if err := jobs.Replay(bson.ObjectIdHex(id)); err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": "job is not dead-lettered or no longer exists"})
		return
	}
	e := audit.FromContext(c)
	e.Action, e.Outcome = "ops.job.replay", "success"
	e.TargetType, e.TargetID = "job", id
	audit.Record(e)
	c.JSON(http.StatusOK, gin.H{"status": "replayed"})
}

// HandleOpsAuditList is the tenant-isolated, permission-gated read API over
// the audit ledger (OPS-005). Filters: action prefix, actor, target, from/to
// (unix seconds), limit/skip pagination.
func HandleOpsAuditList(c *gin.Context) {
	tenantID := auth.GetTenantObjectID(c)
	if tenantID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	q := audit.Query{
		TenantID: tenantID,
		Action:   c.Query("action"),
		ActorID:  c.Query("actor"),
		Target:   c.Query("target"),
	}
	if v, err := strconv.ParseInt(c.Query("from"), 10, 64); err == nil && v > 0 {
		q.From = time.Unix(v, 0)
	}
	if v, err := strconv.ParseInt(c.Query("to"), 10, 64); err == nil && v > 0 {
		q.To = time.Unix(v, 0)
	}
	if v, err := strconv.Atoi(c.Query("limit")); err == nil {
		q.Limit = v
	}
	if v, err := strconv.Atoi(c.Query("skip")); err == nil {
		q.Skip = v
	}
	events, err := audit.List(q)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to read audit events"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"events": events})
}
