package routes

import (
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"

	"github.com/josephalai/sentanyl/pkg/audit"
	"github.com/josephalai/sentanyl/pkg/auth"
	"github.com/josephalai/sentanyl/pkg/db"
	"github.com/josephalai/sentanyl/pkg/models"
)

func EnsureMachineCredentialIndexes() {
	col := db.GetCollection(models.MachineCredentialCollection)
	for _, idx := range []mgo.Index{
		{Key: []string{"key_hash"}, Unique: true, Background: true},
		{Key: []string{"tenant_id", "name"}, Unique: true, Background: true},
		{Key: []string{"tenant_id", "public_id"}, Unique: true, Background: true},
	} {
		if err := col.EnsureIndex(idx); err != nil {
			log.Printf("machine credentials index %v: %v", idx.Key, err)
		}
	}
	if err := db.GetCollection(models.MachineCommandCollection).EnsureIndex(mgo.Index{
		Key: []string{"credential_id", "idempotency_key"}, Unique: true, Background: true,
	}); err != nil {
		log.Printf("machine commands index: %v", err)
	}
}

type machineCredentialRequest struct {
	Name             string   `json:"name"`
	AllowedTools     []string `json:"allowed_tools"`
	PermissionScopes []string `json:"permission_scopes"`
	CanApprove       bool     `json:"can_approve"`
}

func validateMachineCredential(req machineCredentialRequest) (machineCredentialRequest, string) {
	req.Name = strings.TrimSpace(req.Name)
	if req.Name == "" || len(req.Name) > 80 {
		return req, "name is required (max 80 characters)"
	}
	tools, msg := validateAllowedTools(req.AllowedTools)
	if msg != "" {
		return req, msg
	}
	req.AllowedTools = tools
	allowedScopes := map[string]bool{
		string(auth.PermContentManage): true, string(auth.PermSendManage): true,
		string(auth.PermGrantManage): true, string(auth.PermDomainManage): true,
		string(auth.PermSettingsManage): true, string(auth.PermReportsView): true,
	}
	if len(req.PermissionScopes) == 0 {
		req.PermissionScopes = []string{string(auth.PermContentManage)}
	}
	seen := map[string]bool{}
	scopes := []string{}
	for _, scope := range req.PermissionScopes {
		if !allowedScopes[scope] {
			return req, "unsupported or owner-only permission scope: " + scope
		}
		if !seen[scope] {
			seen[scope] = true
			scopes = append(scopes, scope)
		}
	}
	req.PermissionScopes = scopes
	return req, ""
}

func HandleListMachineCredentials(c *gin.Context) {
	tenantID := auth.GetTenantObjectID(c)
	var rows []models.MachineCredential
	if err := db.GetCollection(models.MachineCredentialCollection).Find(bson.M{"tenant_id": tenantID}).Sort("name").All(&rows); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list credentials"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"credentials": rows})
}

func HandleCreateMachineCredential(c *gin.Context) {
	tenantID := auth.GetTenantObjectID(c)
	var req machineCredentialRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
		return
	}
	var msg string
	req, msg = validateMachineCredential(req)
	if msg != "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": msg})
		return
	}
	key, err := auth.GenerateAPIKey()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "key generation failed"})
		return
	}
	row := models.NewMachineCredential(tenantID, req.Name, auth.HashAPIKey(key), auth.APIKeyPrefix(key))
	row.AllowedTools, row.PermissionScopes, row.CanApprove = req.AllowedTools, req.PermissionScopes, req.CanApprove
	if id := auth.GetAccountUserID(c); bson.IsObjectIdHex(id) {
		row.CreatedBy = bson.ObjectIdHex(id)
	}
	if err := db.GetCollection(models.MachineCredentialCollection).Insert(row); err != nil {
		if mgo.IsDup(err) {
			c.JSON(http.StatusConflict, gin.H{"error": "credential name already exists"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create credential"})
		}
		return
	}
	e := audit.FromContext(c)
	e.Action, e.Outcome, e.TargetType, e.TargetID = "machine_credential.create", "success", "machine_credential", row.PublicID
	audit.Record(e)
	c.JSON(http.StatusCreated, gin.H{"credential": row, "api_key": key})
}

func HandleRotateMachineCredential(c *gin.Context) {
	tenantID := auth.GetTenantObjectID(c)
	key, err := auth.GenerateAPIKey()
	if err != nil {
		c.JSON(500, gin.H{"error": "key generation failed"})
		return
	}
	now := time.Now().UTC()
	err = db.GetCollection(models.MachineCredentialCollection).Update(bson.M{"tenant_id": tenantID, "public_id": c.Param("id"), "status": models.MachineCredentialActive}, bson.M{"$set": bson.M{"key_hash": auth.HashAPIKey(key), "key_prefix": auth.APIKeyPrefix(key), "rotated_at": now, "updated_at": now}})
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "credential not found"})
		return
	}
	var row models.MachineCredential
	_ = db.GetCollection(models.MachineCredentialCollection).Find(bson.M{"tenant_id": tenantID, "public_id": c.Param("id")}).One(&row)
	_, _ = auth.RevokeMachineSessionsForPrincipal(tenantID, "mcp:"+row.PublicID)
	e := audit.FromContext(c)
	e.Action, e.Outcome, e.TargetType, e.TargetID = "machine_credential.rotate", "success", "machine_credential", row.PublicID
	audit.Record(e)
	c.JSON(http.StatusOK, gin.H{"credential": row, "api_key": key})
}

func HandleUpdateMachineCredential(c *gin.Context) {
	tenantID := auth.GetTenantObjectID(c)
	var req machineCredentialRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": "invalid body"})
		return
	}
	var msg string
	req, msg = validateMachineCredential(req)
	if msg != "" {
		c.JSON(400, gin.H{"error": msg})
		return
	}
	now := time.Now().UTC()
	err := db.GetCollection(models.MachineCredentialCollection).Update(bson.M{"tenant_id": tenantID, "public_id": c.Param("id"), "status": models.MachineCredentialActive}, bson.M{"$set": bson.M{"name": req.Name, "allowed_tools": req.AllowedTools, "permission_scopes": req.PermissionScopes, "can_approve": req.CanApprove, "updated_at": now}})
	if err != nil {
		if mgo.IsDup(err) {
			c.JSON(409, gin.H{"error": "credential name already exists"})
		} else {
			c.JSON(404, gin.H{"error": "credential not found"})
		}
		return
	}
	var row models.MachineCredential
	_ = db.GetCollection(models.MachineCredentialCollection).Find(bson.M{"tenant_id": tenantID, "public_id": c.Param("id")}).One(&row)
	_, _ = auth.RevokeMachineSessionsForPrincipal(tenantID, "mcp:"+row.PublicID)
	e := audit.FromContext(c)
	e.Action, e.Outcome, e.TargetType, e.TargetID = "machine_credential.update", "success", "machine_credential", row.PublicID
	audit.Record(e)
	c.JSON(200, gin.H{"credential": row})
}

func HandleRevokeMachineCredential(c *gin.Context) {
	tenantID := auth.GetTenantObjectID(c)
	var row models.MachineCredential
	if err := db.GetCollection(models.MachineCredentialCollection).Find(bson.M{"tenant_id": tenantID, "public_id": c.Param("id"), "status": models.MachineCredentialActive}).One(&row); err != nil {
		c.JSON(404, gin.H{"error": "credential not found"})
		return
	}
	now := time.Now().UTC()
	_ = db.GetCollection(models.MachineCredentialCollection).UpdateId(row.Id, bson.M{"$set": bson.M{"status": models.MachineCredentialRevoked, "revoked_at": now, "updated_at": now}, "$unset": bson.M{"key_hash": ""}})
	_, _ = auth.RevokeMachineSessionsForPrincipal(tenantID, "mcp:"+row.PublicID)
	e := audit.FromContext(c)
	e.Action, e.Outcome, e.TargetType, e.TargetID = "machine_credential.revoke", "success", "machine_credential", row.PublicID
	audit.Record(e)
	c.JSON(200, gin.H{"status": "revoked"})
}
