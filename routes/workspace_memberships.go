package routes

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/josephalai/sentanyl/pkg/audit"
	"github.com/josephalai/sentanyl/pkg/auth"
	"github.com/josephalai/sentanyl/pkg/db"
	"github.com/josephalai/sentanyl/pkg/models"
	mgo "gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

func workspaceAudit(c *gin.Context, action, target, outcome, reason string) {
	e := audit.FromContext(c)
	e.Action, e.TargetType, e.TargetID, e.Outcome, e.Reason = action, "workspace_membership", target, outcome, reason
	audit.Record(e)
}

func validDelegatedRole(role string) bool { return role == auth.RoleAdmin || role == auth.RoleEditor }

func invitationToken() (string, string) {
	raw := make([]byte, 32)
	_, _ = rand.Read(raw)
	token := hex.EncodeToString(raw)
	digest := sha256.Sum256([]byte(token))
	return token, hex.EncodeToString(digest[:])
}

func tokenDigest(token string) string {
	digest := sha256.Sum256([]byte(strings.TrimSpace(token)))
	return hex.EncodeToString(digest[:])
}

// HandleListWorkspaceMembers exposes active/suspended membership state without
// credential material. The identity email/name is joined for operator use.
func HandleListWorkspaceMembers(c *gin.Context) {
	tenantID := auth.GetTenantObjectID(c)
	var memberships []models.WorkspaceMembership
	if err := db.GetCollection(models.WorkspaceMembershipCollection).Find(bson.M{
		"tenant_id": tenantID, "timestamps.deleted_at": nil,
	}).Sort("_id").All(&memberships); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list members"})
		return
	}
	rows := make([]gin.H, 0, len(memberships))
	for _, membership := range memberships {
		var identity models.AccountUser
		_ = db.GetCollection(models.AccountUserCollection).FindId(membership.IdentityID).Select(bson.M{"email": 1, "name": 1}).One(&identity)
		role, _, active := auth.WorkspaceRole(&identity, tenantID)
		if !active {
			role = membership.Role
		}
		rows = append(rows, gin.H{"membership": membership, "email": identity.Email, "name": identity.Name, "effective_role": role})
	}
	var invitations []models.WorkspaceInvitation
	_ = db.GetCollection(models.WorkspaceInvitationCollection).Find(bson.M{
		"tenant_id": tenantID, "status": models.InvitationPending,
	}).Sort("-created_at").All(&invitations)
	c.JSON(http.StatusOK, gin.H{"members": rows, "invitations": invitations})
}

func HandleInviteWorkspaceMember(c *gin.Context) {
	tenantID := auth.GetTenantObjectID(c)
	inviterID := auth.GetAccountUserID(c)
	var req struct {
		Email string `json:"email" binding:"required"`
		Role  string `json:"role" binding:"required"`
	}
	if c.ShouldBindJSON(&req) != nil || !validDelegatedRole(req.Role) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "email and role (admin or editor) are required"})
		return
	}
	req.Email = strings.ToLower(strings.TrimSpace(req.Email))
	if !strings.Contains(req.Email, "@") {
		c.JSON(http.StatusBadRequest, gin.H{"error": "valid email required"})
		return
	}
	if n, _ := db.GetCollection(models.WorkspaceInvitationCollection).Find(bson.M{
		"tenant_id": tenantID, "email": req.Email, "status": models.InvitationPending, "expires_at": bson.M{"$gt": time.Now().UTC()},
	}).Count(); n > 0 {
		c.JSON(http.StatusConflict, gin.H{"error": "a pending invitation already exists"})
		return
	}
	var existing models.AccountUser
	if err := db.GetCollection(models.AccountUserCollection).Find(bson.M{"email": req.Email}).One(&existing); err == nil {
		if n, _ := db.GetCollection(models.WorkspaceMembershipCollection).Find(bson.M{"tenant_id": tenantID, "identity_id": existing.Id}).Count(); n > 0 {
			c.JSON(http.StatusConflict, gin.H{"error": "identity is already a workspace member"})
			return
		}
	}
	raw, digest := invitationToken()
	now := time.Now().UTC()
	inviterOID := bson.ObjectId("")
	if bson.IsObjectIdHex(inviterID) {
		inviterOID = bson.ObjectIdHex(inviterID)
	}
	invite := &models.WorkspaceInvitation{
		Id: bson.NewObjectId(), PublicId: digest[:16], TenantID: tenantID,
		Email: req.Email, Role: req.Role, Status: models.InvitationPending, TokenDigest: digest,
		InvitedBy: inviterOID, ExpiresAt: now.Add(7 * 24 * time.Hour), SoftDeletes: models.SoftDeletes{CreatedAt: &now},
	}
	if err := db.GetCollection(models.WorkspaceInvitationCollection).Insert(invite); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create invitation"})
		return
	}
	workspaceAudit(c, "workspace.invitation.create", invite.Id.Hex(), "success", "")
	c.JSON(http.StatusCreated, gin.H{"invitation": invite, "token": raw})
}

// HandleAcceptWorkspaceInvitation accepts a bearer invitation. Existing human
// identities gain another membership; new identities must establish a password.
// It never mints a session, so normal credential authentication still follows.
func HandleAcceptWorkspaceInvitation(c *gin.Context) {
	var req struct {
		Token     string `json:"token" binding:"required"`
		Password  string `json:"password"`
		FirstName string `json:"first_name"`
		LastName  string `json:"last_name"`
	}
	if c.ShouldBindJSON(&req) != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "token is required"})
		return
	}
	var invite models.WorkspaceInvitation
	if err := db.GetCollection(models.WorkspaceInvitationCollection).Find(bson.M{
		"token_digest": tokenDigest(req.Token), "status": models.InvitationPending, "expires_at": bson.M{"$gt": time.Now().UTC()},
	}).One(&invite); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "invitation is invalid or expired"})
		return
	}
	var identity models.AccountUser
	createdIdentity := false
	err := db.GetCollection(models.AccountUserCollection).Find(bson.M{"email": invite.Email}).One(&identity)
	if err == mgo.ErrNotFound {
		if len(req.Password) < 8 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "password must be at least 8 characters for a new identity"})
			return
		}
		hash, hashErr := auth.HashPassword(req.Password)
		if hashErr != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create identity"})
			return
		}
		identity = *models.NewAccountUser(invite.Email, invite.TenantID)
		identity.Role, identity.PasswordHash = invite.Role, hash
		identity.Name.FirstName, identity.Name.LastName = req.FirstName, req.LastName
		if err := db.GetCollection(models.AccountUserCollection).Insert(&identity); err != nil {
			if !mgo.IsDup(err) || db.GetCollection(models.AccountUserCollection).Find(bson.M{"email": invite.Email}).One(&identity) != nil {
				c.JSON(http.StatusConflict, gin.H{"error": "identity could not be created"})
				return
			}
		} else {
			createdIdentity = true
		}
	} else if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to resolve identity"})
		return
	}
	membership := models.NewWorkspaceMembership(invite.TenantID, identity.Id, invite.Role, invite.InvitedBy)
	if err := db.GetCollection(models.WorkspaceMembershipCollection).Insert(membership); err != nil {
		if createdIdentity {
			_ = db.GetCollection(models.AccountUserCollection).RemoveId(identity.Id)
		}
		c.JSON(http.StatusConflict, gin.H{"error": "workspace membership already exists"})
		return
	}
	now := time.Now().UTC()
	if err := db.GetCollection(models.WorkspaceInvitationCollection).Update(
		bson.M{"_id": invite.Id, "status": models.InvitationPending},
		bson.M{"$set": bson.M{"status": models.InvitationAccepted, "accepted_by": identity.Id, "accepted_at": now}},
	); err != nil {
		_ = db.GetCollection(models.WorkspaceMembershipCollection).RemoveId(membership.Id)
		if createdIdentity {
			_ = db.GetCollection(models.AccountUserCollection).RemoveId(identity.Id)
		}
		c.JSON(http.StatusConflict, gin.H{"error": "invitation was already consumed"})
		return
	}
	e := audit.FromContext(c)
	e.TenantID, e.ActorKind, e.ActorID = invite.TenantID, "human", identity.Id.Hex()
	e.Action, e.TargetType, e.TargetID, e.Outcome = "workspace.invitation.accept", "workspace_membership", membership.Id.Hex(), "success"
	audit.Record(e)
	c.JSON(http.StatusOK, gin.H{"accepted": true, "workspace_id": invite.TenantID.Hex(), "identity_id": identity.Id.Hex()})
}

func HandleUpdateWorkspaceMember(c *gin.Context) {
	tenantID := auth.GetTenantObjectID(c)
	if !bson.IsObjectIdHex(c.Param("id")) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid membership id"})
		return
	}
	membershipID := bson.ObjectIdHex(c.Param("id"))
	var membership models.WorkspaceMembership
	if err := db.GetCollection(models.WorkspaceMembershipCollection).Find(bson.M{"_id": membershipID, "tenant_id": tenantID}).One(&membership); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "membership not found"})
		return
	}
	var tenant models.Tenant
	_ = db.GetCollection(models.TenantCollection).FindId(tenantID).One(&tenant)
	if tenant.OwnerMembershipID == membership.Id {
		c.JSON(http.StatusConflict, gin.H{"error": "transfer ownership before changing or suspending the owner"})
		return
	}
	var req struct {
		Role   string `json:"role"`
		Status string `json:"status"`
	}
	if c.ShouldBindJSON(&req) != nil || (req.Role == "" && req.Status == "") {
		c.JSON(http.StatusBadRequest, gin.H{"error": "role or status required"})
		return
	}
	set := bson.M{}
	if req.Role != "" {
		if !validDelegatedRole(req.Role) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "role must be admin or editor"})
			return
		}
		set["role"] = req.Role
	}
	if req.Status != "" {
		if req.Status != models.MembershipActive && req.Status != models.MembershipSuspended {
			c.JSON(http.StatusBadRequest, gin.H{"error": "status must be active or suspended"})
			return
		}
		set["status"] = req.Status
		if req.Status == models.MembershipSuspended {
			now := time.Now().UTC()
			set["suspended_at"] = now
		} else {
			set["suspended_at"] = nil
		}
	}
	if err := db.GetCollection(models.WorkspaceMembershipCollection).Update(
		bson.M{"_id": membership.Id, "tenant_id": tenantID}, bson.M{"$set": set},
	); err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": "membership changed concurrently"})
		return
	}
	_, _ = auth.RevokeSessionsForPrincipal(models.AuthSessionKindTenant, membership.IdentityID.Hex())
	workspaceAudit(c, "workspace.membership.update", membership.Id.Hex(), "success", "")
	c.JSON(http.StatusOK, gin.H{"updated": true})
}

func HandleTransferWorkspaceOwnership(c *gin.Context) {
	tenantID := auth.GetTenantObjectID(c)
	currentIdentity := auth.GetAccountUserID(c)
	var req struct {
		MembershipID string `json:"membership_id" binding:"required"`
	}
	if c.ShouldBindJSON(&req) != nil || !bson.IsObjectIdHex(req.MembershipID) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "valid membership_id required"})
		return
	}
	var tenant models.Tenant
	if err := db.GetCollection(models.TenantCollection).FindId(tenantID).One(&tenant); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "workspace not found"})
		return
	}
	var target models.WorkspaceMembership
	if err := db.GetCollection(models.WorkspaceMembershipCollection).Find(bson.M{
		"_id": bson.ObjectIdHex(req.MembershipID), "tenant_id": tenantID, "status": models.MembershipActive,
	}).One(&target); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "active target membership not found"})
		return
	}
	if target.Id == tenant.OwnerMembershipID {
		c.JSON(http.StatusOK, gin.H{"transferred": false})
		return
	}
	if err := db.GetCollection(models.TenantCollection).Update(
		bson.M{"_id": tenantID, "owner_membership_id": tenant.OwnerMembershipID},
		bson.M{"$set": bson.M{"owner_membership_id": target.Id}},
	); err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": "ownership changed concurrently"})
		return
	}
	_ = db.GetCollection(models.WorkspaceMembershipCollection).UpdateId(target.Id, bson.M{"$set": bson.M{"role": auth.RoleOwner}})
	_ = db.GetCollection(models.WorkspaceMembershipCollection).UpdateId(tenant.OwnerMembershipID, bson.M{"$set": bson.M{"role": auth.RoleAdmin}})
	_, _ = auth.RevokeSessionsForPrincipal(models.AuthSessionKindTenant, target.IdentityID.Hex())
	if bson.IsObjectIdHex(currentIdentity) {
		_, _ = auth.RevokeSessionsForPrincipal(models.AuthSessionKindTenant, currentIdentity)
	}
	workspaceAudit(c, "workspace.ownership.transfer", target.Id.Hex(), "success", "")
	c.JSON(http.StatusOK, gin.H{"transferred": true, "owner_membership_id": target.Id.Hex()})
}

func HandleSelectWorkspace(c *gin.Context) {
	if !bson.IsObjectIdHex(c.Param("tenantId")) || !bson.IsObjectIdHex(auth.GetAccountUserID(c)) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid workspace"})
		return
	}
	var identity models.AccountUser
	if err := db.GetCollection(models.AccountUserCollection).FindId(bson.ObjectIdHex(auth.GetAccountUserID(c))).One(&identity); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "identity not found"})
		return
	}
	workspaceUser, ok := auth.UserForWorkspace(&identity, bson.ObjectIdHex(c.Param("tenantId")))
	if !ok {
		c.JSON(http.StatusForbidden, gin.H{"error": "no active membership in workspace"})
		return
	}
	token, err := auth.GenerateTenantToken(workspaceUser)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to select workspace"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"token": token, "workspace_id": workspaceUser.TenantID.Hex(), "role": workspaceUser.Role})
}

func HandleRevokeWorkspaceInvitation(c *gin.Context) {
	tenantID := auth.GetTenantObjectID(c)
	if !bson.IsObjectIdHex(c.Param("id")) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid invitation id"})
		return
	}
	now := time.Now().UTC()
	if err := db.GetCollection(models.WorkspaceInvitationCollection).Update(
		bson.M{"_id": bson.ObjectIdHex(c.Param("id")), "tenant_id": tenantID, "status": models.InvitationPending},
		bson.M{"$set": bson.M{"status": models.InvitationRevoked, "revoked_at": now}, "$unset": bson.M{"token_digest": ""}},
	); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "pending invitation not found"})
		return
	}
	workspaceAudit(c, "workspace.invitation.revoke", c.Param("id"), "success", "")
	c.JSON(http.StatusOK, gin.H{"revoked": true})
}
