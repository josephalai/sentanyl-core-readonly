package routes

import (
	"testing"

	"github.com/josephalai/sentanyl/pkg/auth"
)

func TestValidateMachineCredentialRejectsOwnerScopeAndUnknownTool(t *testing.T) {
	_, msg := validateMachineCredential(machineCredentialRequest{Name: "billing robot", PermissionScopes: []string{string(auth.PermBillingManage)}})
	if msg == "" {
		t.Fatal("owner-only billing scope must be rejected")
	}
	_, msg = validateMachineCredential(machineCredentialRequest{Name: "unknown robot", AllowedTools: []string{"not_a_real_tool"}})
	if msg == "" {
		t.Fatal("unknown tool must be rejected")
	}
}

func TestValidateMachineCredentialDeduplicatesSafeScopes(t *testing.T) {
	req, msg := validateMachineCredential(machineCredentialRequest{
		Name: "  delivery operator  ", AllowedTools: []string{"courses_list", "course_enrollment_create"},
		PermissionScopes: []string{string(auth.PermContentManage), string(auth.PermContentManage), string(auth.PermGrantManage)},
	})
	if msg != "" {
		t.Fatalf("valid delegated credential rejected: %s", msg)
	}
	if req.Name != "delivery operator" || len(req.PermissionScopes) != 2 || len(req.AllowedTools) != 2 {
		t.Fatalf("credential normalization wrong: %+v", req)
	}
}
