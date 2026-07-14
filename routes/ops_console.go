package routes

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"gopkg.in/mgo.v2/bson"

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
	c.JSON(http.StatusOK, gin.H{"status": "replayed"})
}
