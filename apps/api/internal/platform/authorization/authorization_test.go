package authorization_test

import (
	"testing"

	"github.com/ccauepereira/SysAP/apps/api/internal/platform/authorization"
)

func TestMatrixCapabilities(t *testing.T) {
	tests := []struct {
		name    string
		role    authorization.Role
		aal     authorization.AAL
		cap     authorization.Capability
		wantErr error
	}{
		// Owner tests
		{"owner AAL1 can read own data", authorization.RoleOwner, authorization.AAL1, authorization.CapReadOwnData, nil},
		{"owner AAL1 can read organization", authorization.RoleOwner, authorization.AAL1, authorization.CapReadOrganization, nil},
		{"owner AAL1 can read athlete", authorization.RoleOwner, authorization.AAL1, authorization.CapReadAthlete, nil},
		{"owner AAL1 can manage plan", authorization.RoleOwner, authorization.AAL1, authorization.CapManagePlanFuture, nil},
		{"owner AAL1 blocked on manage role", authorization.RoleOwner, authorization.AAL1, authorization.CapManageRoleStatus, authorization.ErrUnauthorized},
		{"owner AAL1 blocked on manage finance", authorization.RoleOwner, authorization.AAL1, authorization.CapManageAccessFinanceFuture, authorization.ErrUnauthorized},
		{"owner AAL2 can manage role", authorization.RoleOwner, authorization.AAL2, authorization.CapManageRoleStatus, nil},
		{"owner AAL2 can manage finance", authorization.RoleOwner, authorization.AAL2, authorization.CapManageAccessFinanceFuture, nil},

		// Trainer tests
		{"trainer AAL1 can read own data", authorization.RoleTrainer, authorization.AAL1, authorization.CapReadOwnData, nil},
		{"trainer AAL1 can read organization", authorization.RoleTrainer, authorization.AAL1, authorization.CapReadOrganization, nil},
		{"trainer AAL1 can read athlete", authorization.RoleTrainer, authorization.AAL1, authorization.CapReadAthlete, nil},
		{"trainer AAL1 can manage plan", authorization.RoleTrainer, authorization.AAL1, authorization.CapManagePlanFuture, nil},
		{"trainer AAL1 blocked on manage role", authorization.RoleTrainer, authorization.AAL1, authorization.CapManageRoleStatus, authorization.ErrUnauthorized},
		{"trainer AAL2 blocked on manage role", authorization.RoleTrainer, authorization.AAL2, authorization.CapManageRoleStatus, authorization.ErrUnauthorized},

		// Athlete tests
		{"athlete AAL1 can read own data", authorization.RoleAthlete, authorization.AAL1, authorization.CapReadOwnData, nil},
		{"athlete AAL1 can read athlete", authorization.RoleAthlete, authorization.AAL1, authorization.CapReadAthlete, nil},
		{"athlete AAL1 blocked on read organization", authorization.RoleAthlete, authorization.AAL1, authorization.CapReadOrganization, authorization.ErrUnauthorized},
		{"athlete AAL1 blocked on manage plan", authorization.RoleAthlete, authorization.AAL1, authorization.CapManagePlanFuture, authorization.ErrUnauthorized},
		{"athlete AAL2 blocked on manage role", authorization.RoleAthlete, authorization.AAL2, authorization.CapManageRoleStatus, authorization.ErrUnauthorized},

		// Unknown role
		{"unknown role fails closed", authorization.Role("hacker"), authorization.AAL1, authorization.CapReadOwnData, authorization.ErrUnknownRole},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := authorization.Matrix{Role: tt.role, AAL: tt.aal}
			err := m.Can(tt.cap)
			if err != tt.wantErr {
				t.Errorf("Can() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
