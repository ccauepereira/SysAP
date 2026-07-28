package authorization

import (
	"context"
	"errors"

	"github.com/ccauepereira/SysAP/apps/api/internal/platform/database"
	"github.com/jackc/pgx/v5"
)

type Role string

const (
	RoleOwner   Role = "owner"
	RoleTrainer Role = "trainer"
	RoleAthlete Role = "athlete"
)

type AAL string

const (
	AAL1 AAL = "aal1"
	AAL2 AAL = "aal2"
)

type Capability string

const (
	CapReadOwnData               Capability = "read_own_data"
	CapReadOrganization          Capability = "read_organization"
	CapReadAthlete               Capability = "read_athlete"
	CapManagePlanFuture          Capability = "manage_plan_future"
	CapManageRoleStatus          Capability = "manage_role_status"
	CapManageAccessFinanceFuture Capability = "manage_access_finance_future"
)

var (
	ErrUnauthorized = errors.New("unauthorized access")
	ErrUnknownRole  = errors.New("unknown role")
)

type Matrix struct {
	Role Role
	AAL  AAL
}

func (m Matrix) Can(cap Capability) error {
	switch m.Role {
	case RoleOwner:
		return m.evaluateOwner(cap)
	case RoleTrainer:
		return m.evaluateTrainer(cap)
	case RoleAthlete:
		return m.evaluateAthlete(cap)
	default:
		return ErrUnknownRole
	}
}

func (m Matrix) evaluateOwner(cap Capability) error {
	switch cap {
	case CapReadOwnData, CapReadOrganization, CapReadAthlete, CapManagePlanFuture:
		if m.AAL == AAL1 || m.AAL == AAL2 {
			return nil
		}
	case CapManageRoleStatus, CapManageAccessFinanceFuture:
		if m.AAL == AAL2 {
			return nil
		}
	}
	return ErrUnauthorized
}

func (m Matrix) evaluateTrainer(cap Capability) error {
	switch cap {
	case CapReadOwnData, CapReadOrganization, CapReadAthlete, CapManagePlanFuture:
		if m.AAL == AAL1 || m.AAL == AAL2 {
			return nil
		}
	}
	return ErrUnauthorized
}

func (m Matrix) evaluateAthlete(cap Capability) error {
	switch cap {
	case CapReadOwnData, CapReadAthlete:
		if m.AAL == AAL1 || m.AAL == AAL2 {
			return nil
		}
	}
	return ErrUnauthorized
}

type Membership struct {
	ID             string
	OrganizationID string
	Role           Role
	Status         string
}

// WithTenantContext validates active membership before setting the organization context.
func WithTenantContext(
	ctx context.Context,
	pool *database.Pool,
	identity database.AuthenticatedContext,
	organizationID string,
	action func(pgx.Tx, Membership) error,
) error {
	baseIdentity := identity
	// Disable organization context in the beginning to avoid bypassing membership read restriction
	baseIdentity.OrganizationID.Valid = false

	return pool.WithAuthenticatedContext(ctx, baseIdentity, func(tx pgx.Tx) error {
		var membership Membership
		membership.OrganizationID = organizationID

		err := tx.QueryRow(ctx, `
			select id, role, status
			from app.organization_memberships
			where organization_id = $1
			  and status = 'active'
		`, organizationID).Scan(&membership.ID, &membership.Role, &membership.Status)

		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return ErrUnauthorized
			}
			return err
		}

		if _, err := tx.Exec(ctx, "select set_config('app.current_organization_id', $1, true)", organizationID); err != nil {
			return err
		}

		return action(tx, membership)
	})
}

// CanAccessAthlete checks if the current membership role allows accessing a specific athlete.
func CanAccessAthlete(ctx context.Context, tx pgx.Tx, membership Membership, athleteID string) error {
	switch membership.Role {
	case RoleOwner:
		// Owner can access any athlete in the tenant, and RLS already isolates by tenant.
		return nil

	case RoleAthlete:
		// Athlete can only access themselves.
		var selfAthleteID string
		err := tx.QueryRow(ctx, `
			select id from app.athletes
			where organization_id = $1 and membership_id = $2
		`, membership.OrganizationID, membership.ID).Scan(&selfAthleteID)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return ErrUnauthorized
			}
			return err
		}
		if selfAthleteID != athleteID {
			return ErrUnauthorized
		}
		return nil

	case RoleTrainer:
		// Trainer accesses only athletes with active assignment.
		var assigned int
		err := tx.QueryRow(ctx, `
			select 1 from app.trainer_athlete_assignments
			where organization_id = $1
			  and trainer_membership_id = $2
			  and athlete_id = $3
			  and revoked_at is null
		`, membership.OrganizationID, membership.ID, athleteID).Scan(&assigned)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return ErrUnauthorized
			}
			return err
		}
		return nil

	default:
		return ErrUnknownRole
	}
}
