package identity

import (
	"context"
	"errors"
	"fmt"
	"net/mail"
	"regexp"
	"strings"
	"time"

	"github.com/ccauepereira/SysAP/apps/api/internal/platform/database"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

var (
	phoneRegex = regexp.MustCompile(`^\+[1-9]\d{1,14}$`)
)

type BootstrapOwnerParams struct {
	OrgName  string
	Timezone string
	FullName string
	Email    string
	Phone    string
	Password string
}

func (p BootstrapOwnerParams) Validate() error {
	if strings.TrimSpace(p.OrgName) == "" {
		return errors.New("organization name cannot be empty")
	}
	if strings.TrimSpace(p.Timezone) == "" {
		return errors.New("timezone cannot be empty")
	}
	if strings.TrimSpace(p.FullName) == "" {
		return errors.New("full name cannot be empty")
	}
	if _, err := mail.ParseAddress(p.Email); err != nil {
		return errors.New("invalid email address format")
	}
	if !phoneRegex.MatchString(p.Phone) {
		return errors.New("invalid E.164 phone format (e.g. +5585999999999)")
	}
	if !IsValidPassword(p.Password) {
		return errors.New("password does not meet policy requirements (minimum 15 characters, 4 digits, 2 special characters)")
	}
	return nil
}

func BootstrapOwner(
	ctx context.Context,
	pool *database.Pool,
	authAdmin SupabaseAuthAdmin,
	generator EnrollmentNumberGenerator,
	params BootstrapOwnerParams,
	now func() time.Time,
) (string, error) {
	if err := params.Validate(); err != nil {
		return "", err
	}

	if now == nil {
		now = func() time.Time { return time.Now().UTC() }
	}

	var enrollment string
	var authUserID uuid.UUID
	var authCreated bool

	txErr := pool.WithTransaction(ctx, func(tx pgx.Tx) error {
		// Reset role to superuser (postgres) to bypass RLS policies
		if _, err := tx.Exec(ctx, "reset role"); err != nil {
			return fmt.Errorf("failed to reset role to superuser: %w", err)
		}

		// Check if owner already exists for organization
		var existingOwners int
		err := tx.QueryRow(ctx, `
			select count(*)
			from app.organization_memberships m
			join app.organizations o on o.id = m.organization_id
			where o.name = $1 and m.role = 'owner'
		`, params.OrgName).Scan(&existingOwners)
		if err != nil {
			return fmt.Errorf("failed to check for existing owners: %w", err)
		}
		if existingOwners > 0 {
			return errors.New("owner_already_exists")
		}

		// Generate enrollment number
		enrollment, err = generator.Generate()
		if err != nil {
			return fmt.Errorf("failed to generate enrollment: %w", err)
		}

		// Create Auth user in Supabase Auth
		authUserID, err = authAdmin.CreateUserWithEmail(ctx, params.Email, params.Phone, params.Password)
		if err != nil {
			return fmt.Errorf("auth_failed: %w", err)
		}
		authCreated = true

		nowUTC := now().UTC()

		// Insert Organization
		orgID := uuid.New()
		_, err = tx.Exec(ctx, `
			insert into app.organizations (id, name, timezone, status, created_at, updated_at)
			values ($1, $2, $3, 'active', $4, $4)
		`, orgID, params.OrgName, params.Timezone, nowUTC)
		if err != nil {
			return fmt.Errorf("failed to insert organization: %w", err)
		}

		// Insert Profile
		profileID := uuid.New()
		_, err = tx.Exec(ctx, `
			insert into app.profiles (id, auth_user_id, full_name, created_at, updated_at)
			values ($1, $2, $3, $4, $4)
		`, profileID, authUserID, params.FullName, nowUTC)
		if err != nil {
			return fmt.Errorf("failed to insert profile: %w", err)
		}

		// Insert Membership
		membershipID := uuid.New()
		_, err = tx.Exec(ctx, `
			insert into app.organization_memberships (id, organization_id, profile_id, role, status, activated_at, created_at, updated_at)
			values ($1, $2, $3, 'owner', 'active', $4, $4, $4)
		`, membershipID, orgID, profileID, nowUTC)
		if err != nil {
			return fmt.Errorf("failed to insert membership: %w", err)
		}

		// Insert Login Enrollment
		_, err = tx.Exec(ctx, `
			insert into app.login_enrollments (enrollment_number, profile_id, organization_id, created_at)
			values ($1, $2, $3, $4)
		`, enrollment, profileID, orgID, nowUTC)
		if err != nil {
			return fmt.Errorf("failed to insert login enrollment: %w", err)
		}

		return nil
	})

	if txErr != nil {
		// Compensation: delete Auth user if created
		if authCreated && authUserID != uuid.Nil {
			compErr := authAdmin.DeleteUser(ctx, authUserID)
			if compErr != nil {
				// Record repair task if compensation fails
				repairErr := pool.WithTransaction(ctx, func(tx pgx.Tx) error {
					if _, err := tx.Exec(ctx, "reset role"); err != nil {
						return err
					}
					_, err := tx.Exec(ctx, `
						insert into app.identity_repair_tasks (auth_user_id, operation, status)
						values ($1, 'delete_auth_user', 'pending')
						on conflict (auth_user_id, operation) where status = 'pending' do nothing
					`, authUserID)
					return err
				})
				if repairErr != nil {
					// Secure error logging (no secrets, no PII)
					return "", fmt.Errorf("transaction failed (%w), compensation failed (%v), repair task record failed: %v", txErr, compErr, repairErr)
				}
				return "", fmt.Errorf("transaction failed (%w), compensation failed: %v (repair task recorded)", txErr, compErr)
			}
		}
		return "", txErr
	}

	return enrollment, nil
}
