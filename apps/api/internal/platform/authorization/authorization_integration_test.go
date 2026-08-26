package authorization_test

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/ccauepereira/SysAP/apps/api/internal/platform/authorization"
	"github.com/ccauepereira/SysAP/apps/api/internal/platform/database"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestAuthorizationIntegration(t *testing.T) {
	databaseURL := os.Getenv("SYSAP_TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("integration database is not configured")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	adminPool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		t.Fatal("could not prepare integration administration pool")
	}
	defer adminPool.Close()

	pool, err := database.NewPool(ctx, databaseURL)
	if err != nil {
		t.Fatal("could not prepare integration API pool")
	}
	defer pool.Close()

	fixture := createAuthFixture(t, adminPool, ctx)
	defer fixture.remove(t, adminPool, ctx)

	t.Run("suspended membership blocks next request", func(t *testing.T) {
		identity := database.AuthenticatedContext{
			SubjectID: mustPGUUID(t, fixture.subjectSuspended),
			SessionID: mustPGUUID(t, fixture.sessionSuspended),
		}
		err := authorization.WithTenantContext(ctx, pool, identity, fixture.org1, func(tx pgx.Tx, m authorization.Membership) error {
			return nil
		})
		if !errors.Is(err, authorization.ErrUnauthorized) {
			t.Fatalf("expected ErrUnauthorized for suspended membership, got %v", err)
		}
	})

	t.Run("owner sees only own tenant data", func(t *testing.T) {
		identity := database.AuthenticatedContext{
			SubjectID: mustPGUUID(t, fixture.subjectOwner1),
			SessionID: mustPGUUID(t, fixture.sessionOwner1),
		}
		err := authorization.WithTenantContext(ctx, pool, identity, fixture.org1, func(tx pgx.Tx, m authorization.Membership) error {
			if m.Role != authorization.RoleOwner {
				t.Fatalf("expected owner role")
			}

			var count int
			err := tx.QueryRow(ctx, "select count(*) from app.athletes").Scan(&count)
			if err != nil {
				return err
			}
			if count != 2 { // Athlete1 and Athlete2 are in Org1
				t.Fatalf("expected 2 athletes, got %d", count)
			}

			// Can access any athlete in the org
			errAthlete1 := authorization.CanAccessAthlete(ctx, tx, m, fixture.athlete1)
			errAthlete2 := authorization.CanAccessAthlete(ctx, tx, m, fixture.athlete2)
			if errAthlete1 != nil || errAthlete2 != nil {
				t.Fatalf("owner should access any athlete in tenant")
			}
			return nil
		})
		if err != nil {
			t.Fatalf("WithTenantContext error: %v", err)
		}

		// Attempt to use org2 with org1 owner (they don't have membership in org2)
		err2 := authorization.WithTenantContext(ctx, pool, identity, fixture.org2, func(tx pgx.Tx, m authorization.Membership) error {
			return nil
		})
		if !errors.Is(err2, authorization.ErrUnauthorized) {
			t.Fatalf("expected ErrUnauthorized for owner accessing other org")
		}
	})

	t.Run("trainer sees only assigned athlete", func(t *testing.T) {
		identity := database.AuthenticatedContext{
			SubjectID: mustPGUUID(t, fixture.subjectTrainer1),
			SessionID: mustPGUUID(t, fixture.sessionTrainer1),
		}
		err := authorization.WithTenantContext(ctx, pool, identity, fixture.org1, func(tx pgx.Tx, m authorization.Membership) error {
			if m.Role != authorization.RoleTrainer {
				t.Fatalf("expected trainer role")
			}

			// Trainer has assignment to athlete1, but not athlete2
			if err := authorization.CanAccessAthlete(ctx, tx, m, fixture.athlete1); err != nil {
				t.Fatalf("trainer should access assigned athlete, got %v", err)
			}
			if err := authorization.CanAccessAthlete(ctx, tx, m, fixture.athlete2); !errors.Is(err, authorization.ErrUnauthorized) {
				t.Fatalf("trainer should not access unassigned athlete")
			}
			return nil
		})
		if err != nil {
			t.Fatalf("WithTenantContext error: %v", err)
		}
	})

	t.Run("athlete sees only own record", func(t *testing.T) {
		identity := database.AuthenticatedContext{
			SubjectID: mustPGUUID(t, fixture.subjectAthlete1),
			SessionID: mustPGUUID(t, fixture.sessionAthlete1),
		}
		err := authorization.WithTenantContext(ctx, pool, identity, fixture.org1, func(tx pgx.Tx, m authorization.Membership) error {
			if m.Role != authorization.RoleAthlete {
				t.Fatalf("expected athlete role")
			}

			if err := authorization.CanAccessAthlete(ctx, tx, m, fixture.athlete1); err != nil {
				t.Fatalf("athlete should access own record, got %v", err)
			}
			if err := authorization.CanAccessAthlete(ctx, tx, m, fixture.athlete2); !errors.Is(err, authorization.ErrUnauthorized) {
				t.Fatalf("athlete should not access other athlete record")
			}
			return nil
		})
		if err != nil {
			t.Fatalf("WithTenantContext error: %v", err)
		}
	})

	t.Run("RLS context does not leak", func(t *testing.T) {
		connection, err := adminPool.Acquire(ctx)
		if err != nil {
			t.Fatal(err)
		}
		defer connection.Release()

		tx1, _ := connection.Begin(ctx)
		tx1.Exec(ctx, "select set_config('app.current_auth_subject_id', $1, true)", fixture.subjectOwner1)
		tx1.Exec(ctx, "select set_config('app.current_organization_id', $1, true)", fixture.org1)
		tx1.Commit(ctx)

		tx2, _ := connection.Begin(ctx)
		defer tx2.Rollback(ctx)
		var orgID *string
		tx2.QueryRow(ctx, "select current_setting('app.current_organization_id', true)").Scan(&orgID)
		if orgID != nil && *orgID != "" {
			t.Fatalf("context leaked between transactions: %v", *orgID)
		}
	})

	t.Run("RLS requires context", func(t *testing.T) {
		tx, _ := adminPool.Begin(ctx)
		defer tx.Rollback(ctx)

		tx.Exec(ctx, "set role sysap_api")

		var count int
		tx.QueryRow(ctx, "select count(*) from app.athletes").Scan(&count)
		if count != 0 {
			t.Fatalf("could read athletes without context")
		}
	})

	t.Run("privileges", func(t *testing.T) {
		tx, _ := adminPool.Begin(ctx)
		defer tx.Rollback(ctx)

		for _, role := range []string{"public", "anon", "authenticated", "service_role"} {
			var canSelect bool
			tx.QueryRow(ctx, "select has_table_privilege($1, 'app.athletes', 'select')", role).Scan(&canSelect)
			if canSelect {
				t.Fatalf("role %s should not have select on athletes", role)
			}
		}

		var bypassRLS, superUser bool
		tx.QueryRow(ctx, "select rolbypassrls, rolsuper from pg_roles where rolname = 'sysap_api'").Scan(&bypassRLS, &superUser)
		if bypassRLS || superUser {
			t.Fatalf("sysap_api has dangerous privileges")
		}
	})
}

type authFixture struct {
	org1, org2 string

	subjectOwner1, sessionOwner1    string
	profileOwner1, membershipOwner1 string

	subjectTrainer1, sessionTrainer1    string
	profileTrainer1, membershipTrainer1 string

	subjectAthlete1, sessionAthlete1              string
	profileAthlete1, membershipAthlete1, athlete1 string

	subjectAthlete2, sessionAthlete2              string
	profileAthlete2, membershipAthlete2, athlete2 string

	subjectSuspended, sessionSuspended    string
	profileSuspended, membershipSuspended string
}

func createAuthFixture(t *testing.T, pool *pgxpool.Pool, ctx context.Context) authFixture {
	f := authFixture{
		org1: newUUID(t, pool, ctx), org2: newUUID(t, pool, ctx),
		subjectOwner1: newUUID(t, pool, ctx), sessionOwner1: newUUID(t, pool, ctx), profileOwner1: newUUID(t, pool, ctx), membershipOwner1: newUUID(t, pool, ctx),
		subjectTrainer1: newUUID(t, pool, ctx), sessionTrainer1: newUUID(t, pool, ctx), profileTrainer1: newUUID(t, pool, ctx), membershipTrainer1: newUUID(t, pool, ctx),
		subjectAthlete1: newUUID(t, pool, ctx), sessionAthlete1: newUUID(t, pool, ctx), profileAthlete1: newUUID(t, pool, ctx), membershipAthlete1: newUUID(t, pool, ctx), athlete1: newUUID(t, pool, ctx),
		subjectAthlete2: newUUID(t, pool, ctx), sessionAthlete2: newUUID(t, pool, ctx), profileAthlete2: newUUID(t, pool, ctx), membershipAthlete2: newUUID(t, pool, ctx), athlete2: newUUID(t, pool, ctx),
		subjectSuspended: newUUID(t, pool, ctx), sessionSuspended: newUUID(t, pool, ctx), profileSuspended: newUUID(t, pool, ctx), membershipSuspended: newUUID(t, pool, ctx),
	}

	tx, _ := pool.Begin(ctx)
	defer tx.Rollback(ctx)

	tx.Exec(ctx, "set local session_replication_role = replica")

	// Orgs
	tx.Exec(ctx, "insert into app.organizations(id, name, timezone, status) values ($1, 'Org1', 'UTC', 'active'), ($2, 'Org2', 'UTC', 'active')", f.org1, f.org2)

	// Profiles
	profiles := []struct{ id, sub string }{
		{f.profileOwner1, f.subjectOwner1},
		{f.profileTrainer1, f.subjectTrainer1},
		{f.profileAthlete1, f.subjectAthlete1},
		{f.profileAthlete2, f.subjectAthlete2},
		{f.profileSuspended, f.subjectSuspended},
	}
	for _, p := range profiles {
		tx.Exec(ctx, "insert into app.profiles(id, auth_user_id, full_name) values ($1, $2, 'Name')", p.id, p.sub)
	}

	// Memberships
	memberships := []struct{ id, org, prof, role, status string }{
		{f.membershipOwner1, f.org1, f.profileOwner1, "owner", "active"},
		{f.membershipTrainer1, f.org1, f.profileTrainer1, "trainer", "active"},
		{f.membershipAthlete1, f.org1, f.profileAthlete1, "athlete", "active"},
		{f.membershipAthlete2, f.org1, f.profileAthlete2, "athlete", "active"},
		{f.membershipSuspended, f.org1, f.profileSuspended, "owner", "suspended"},
	}
	for _, m := range memberships {
		tx.Exec(ctx, "insert into app.organization_memberships(id, organization_id, profile_id, role, status) values ($1, $2, $3, $4, $5)", m.id, m.org, m.prof, m.role, m.status)
	}

	// Sessions
	sessions := []struct{ id, prof string }{
		{f.sessionOwner1, f.profileOwner1},
		{f.sessionTrainer1, f.profileTrainer1},
		{f.sessionAthlete1, f.profileAthlete1},
		{f.sessionAthlete2, f.profileAthlete2},
		{f.sessionSuspended, f.profileSuspended},
	}
	for _, s := range sessions {
		tx.Exec(ctx, "insert into app.auth_sessions(session_id, profile_id, assurance_level) values ($1, $2, 'aal1')", s.id, s.prof)
	}

	// Athletes
	athletes := []struct{ id, org, mem, enr string }{
		{f.athlete1, f.org1, f.membershipAthlete1, "1111111111"},
		{f.athlete2, f.org1, f.membershipAthlete2, "2222222222"},
	}
	for _, a := range athletes {
		tx.Exec(ctx, "insert into app.athletes(id, organization_id, membership_id, enrollment_number) values ($1, $2, $3, $4)", a.id, a.org, a.mem, a.enr)
	}

	// Assignments
	tx.Exec(ctx, "insert into app.trainer_athlete_assignments(id, organization_id, trainer_membership_id, athlete_id, assigned_by_membership_id) values (gen_random_uuid(), $1, $2, $3, $4)", f.org1, f.membershipTrainer1, f.athlete1, f.membershipOwner1)

	tx.Commit(ctx)

	return f
}

func (f authFixture) remove(t *testing.T, pool *pgxpool.Pool, ctx context.Context) {
	tx, _ := pool.Begin(ctx)
	defer tx.Rollback(ctx)
	tx.Exec(ctx, "delete from app.trainer_athlete_assignments")
	tx.Exec(ctx, "delete from app.athletes")
	tx.Exec(ctx, "delete from app.auth_sessions")
	tx.Exec(ctx, "delete from app.organization_memberships")
	tx.Exec(ctx, "delete from app.profiles")
	tx.Exec(ctx, "delete from app.organizations")
	tx.Commit(ctx)
}

func newUUID(t *testing.T, pool *pgxpool.Pool, ctx context.Context) string {
	var id string
	pool.QueryRow(ctx, "select gen_random_uuid()").Scan(&id)
	return id
}

func mustPGUUID(t *testing.T, value string) pgtype.UUID {
	var identifier pgtype.UUID
	if err := identifier.Scan(value); err != nil {
		t.Fatal(err)
	}
	return identifier
}
