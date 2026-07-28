package database

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestPhase2CSessionContext(t *testing.T) {
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

	pool, err := NewPool(ctx, databaseURL)
	if err != nil {
		t.Fatal("could not prepare integration API pool")
	}
	defer pool.Close()

	fixture := createPhase2CFixture(t, adminPool, ctx)
	defer fixture.remove(t, adminPool, ctx)

	t.Run("structure and revocation constraint", func(t *testing.T) {
		var exists, rlsEnabled, forceRLS bool
		scanRow(t, adminPool, ctx, "select to_regclass('app.auth_sessions') is not null", &exists)
		scanRow(t, adminPool, ctx, `
			select relrowsecurity, relforcerowsecurity
			from pg_class
			where oid = 'app.auth_sessions'::regclass
		`, &rlsEnabled, &forceRLS)
		if !exists || !rlsEnabled || !forceRLS {
			t.Fatal("auth_sessions must exist with RLS and FORCE ROW LEVEL SECURITY")
		}

		columnTypes := map[string]string{}
		rows, err := adminPool.Query(ctx, `
			select column_name, data_type
			from information_schema.columns
			where table_schema = 'app'
			  and table_name = 'auth_sessions'
			  and column_name in ('session_id', 'profile_id', 'assurance_level', 'registered_at', 'revoked_at', 'revocation_reason')
		`)
		if err != nil {
			t.Fatal("could not inspect auth_sessions columns")
		}
		defer rows.Close()
		for rows.Next() {
			var name, dataType string
			if err := rows.Scan(&name, &dataType); err != nil {
				t.Fatal("could not scan auth_sessions column")
			}
			columnTypes[name] = dataType
		}
		if rows.Err() != nil {
			t.Fatal("could not finish auth_sessions column inspection")
		}
		for column, want := range map[string]string{
			"session_id":        "uuid",
			"profile_id":        "uuid",
			"assurance_level":   "text",
			"registered_at":     "timestamp with time zone",
			"revoked_at":        "timestamp with time zone",
			"revocation_reason": "text",
		} {
			if got := columnTypes[column]; got != want {
				t.Fatalf("auth_sessions column %q type = %q, want %q", column, got, want)
			}
		}

		assertAuthSessionInsert(t, adminPool, ctx, fixture.profileA, "aal1", nil, nil, false)
		revokedAt := "2026-07-26T12:00:00Z"
		reason := "logout"
		assertAuthSessionInsert(t, adminPool, ctx, fixture.profileA, "aal2", &revokedAt, &reason, false)
		assertAuthSessionInsert(t, adminPool, ctx, fixture.profileA, "aal1", &revokedAt, nil, true)
		assertAuthSessionInsert(t, adminPool, ctx, fixture.profileA, "aal1", nil, &reason, true)
		assertAuthSessionInsert(t, adminPool, ctx, fixture.profileA, "aal1", &revokedAt, stringPointer(" "), true)
		assertAuthSessionInsert(t, adminPool, ctx, fixture.profileA, "invalid", nil, nil, true)
	})

	t.Run("privileges are least privilege", func(t *testing.T) {
		for _, privilege := range []string{"select", "insert", "update", "delete", "truncate", "references", "trigger"} {
			var granted bool
			scanRowWithArguments(t, adminPool, ctx,
				"select has_table_privilege('sysap_api', 'app.auth_sessions', $1)",
				[]any{privilege}, &granted,
			)
			if granted != (privilege == "select" || privilege == "insert") {
				t.Fatalf("sysap_api privilege %q on auth_sessions = %v", privilege, granted)
			}
		}

		for _, table := range []string{"profiles", "organization_memberships", "auth_sessions"} {
			for _, role := range clientRoles {
				for _, privilege := range []string{"select", "insert", "update", "delete"} {
					var granted bool
					scanRowWithArguments(t, adminPool, ctx,
						"select has_table_privilege($1, 'app.' || $2::text, $3)",
						[]any{role, table, privilege}, &granted,
					)
					if granted {
						t.Fatalf("client role %q has %s on %s", role, privilege, table)
					}
				}
			}
			assertNoPublicTablePrivilege(t, adminPool, ctx, "app."+table)
		}
		for _, function := range []string{"current_auth_subject_id()", "current_auth_session_id()"} {
			var apiCanExecute bool
			scanRowWithArguments(t, adminPool, ctx,
				"select has_function_privilege('sysap_api', 'app.' || $1::text, 'execute')",
				[]any{function}, &apiCanExecute,
			)
			if !apiCanExecute {
				t.Fatalf("sysap_api must execute %s", function)
			}
			for _, role := range clientRoles {
				var clientCanExecute bool
				scanRowWithArguments(t, adminPool, ctx,
					"select has_function_privilege($1, 'app.' || $2::text, 'execute')",
					[]any{role, function}, &clientCanExecute,
				)
				if clientCanExecute {
					t.Fatalf("client role %q must not execute %s", role, function)
				}
			}
			assertNoPublicFunctionExecute(t, adminPool, ctx, "app."+function)
		}
	})

	t.Run("context functions are safe and correctly configured", func(t *testing.T) {
		transaction, err := pool.pool.Begin(ctx)
		if err != nil {
			t.Fatal("could not begin context function assertion")
		}
		defer func() { _ = transaction.Rollback(ctx) }()

		var subjectIsNull, sessionIsNull bool
		scanRow(t, transaction, ctx, "select app.current_auth_subject_id() is null", &subjectIsNull)
		scanRow(t, transaction, ctx, "select app.current_auth_session_id() is null", &sessionIsNull)
		if !subjectIsNull || !sessionIsNull {
			t.Fatal("identity context must be NULL without GUCs")
		}

		setLocalGUC(t, transaction, ctx, "app.current_auth_subject_id", fixture.subjectA)
		setLocalGUC(t, transaction, ctx, "app.current_auth_session_id", fixture.sessionA)
		var subjectID, sessionID string
		scanRow(t, transaction, ctx, "select app.current_auth_subject_id()::text", &subjectID)
		scanRow(t, transaction, ctx, "select app.current_auth_session_id()::text", &sessionID)
		if subjectID != fixture.subjectA || sessionID != fixture.sessionA {
			t.Fatal("identity context did not return configured UUIDs")
		}

		setLocalGUC(t, transaction, ctx, "app.current_auth_subject_id", "invalid")
		setLocalGUC(t, transaction, ctx, "app.current_auth_session_id", "invalid")
		scanRow(t, transaction, ctx, "select app.current_auth_subject_id() is null", &subjectIsNull)
		scanRow(t, transaction, ctx, "select app.current_auth_session_id() is null", &sessionIsNull)
		if !subjectIsNull || !sessionIsNull {
			t.Fatal("invalid identity context must be NULL")
		}

		for _, function := range []string{"current_auth_subject_id", "current_auth_session_id"} {
			var securityDefiner bool
			var volatility string
			var configuration []string
			scanRowWithArguments(t, adminPool, ctx, `
				select prosecdef, provolatile::text, coalesce(proconfig, array[]::text[])
				from pg_proc
				where oid = ('app.' || $1 || '()')::regprocedure
			`, []any{function}, &securityDefiner, &volatility, &configuration)
			if securityDefiner || volatility != "s" || !containsString(configuration, "search_path=pg_catalog") {
				t.Fatalf("function %s does not have SECURITY INVOKER, STABLE, and secure search_path", function)
			}
		}
	})

	t.Run("RLS resolves only the authenticated subject", func(t *testing.T) {
		transaction, err := pool.pool.Begin(ctx)
		if err != nil {
			t.Fatal("could not begin no-context assertion")
		}
		defer func() { _ = transaction.Rollback(ctx) }()
		for _, table := range []string{"profiles", "organization_memberships", "auth_sessions"} {
			var count int
			scanRow(t, transaction, ctx, "select count(*) from app."+table, &count)
			if count != 0 {
				t.Fatalf("table %s exposed rows without authenticated context", table)
			}
		}

		identity := AuthenticatedContext{
			SubjectID: mustPGUUID(t, fixture.subjectA),
			SessionID: mustPGUUID(t, fixture.sessionA),
		}
		if err := pool.WithAuthenticatedContext(ctx, identity, func(transaction pgx.Tx) error {
			var profileID string
			if err := transaction.QueryRow(ctx, "select id::text from app.profiles").Scan(&profileID); err != nil {
				return err
			}
			if profileID != fixture.profileA {
				return errors.New("authenticated subject resolved another profile")
			}

			var membershipCount int
			if err := transaction.QueryRow(ctx, "select count(*) from app.organization_memberships").Scan(&membershipCount); err != nil {
				return err
			}
			if membershipCount != 1 {
				return errors.New("authenticated subject saw another membership")
			}

			var sessionID string
			if err := transaction.QueryRow(ctx, "select session_id::text from app.auth_sessions").Scan(&sessionID); err != nil {
				return err
			}
			if sessionID != fixture.sessionA {
				return errors.New("authenticated subject resolved another session")
			}
			return nil
		}); err != nil {
			t.Fatalf("authenticated RLS query failed: %v", err)
		}

		identity.SessionID = mustPGUUID(t, fixture.sessionB)
		if err := pool.WithAuthenticatedContext(ctx, identity, func(transaction pgx.Tx) error {
			var sessionCount int
			if err := transaction.QueryRow(ctx, "select count(*) from app.auth_sessions").Scan(&sessionCount); err != nil {
				return err
			}
			if sessionCount != 0 {
				return errors.New("mismatched session was visible")
			}
			return nil
		}); err != nil {
			t.Fatalf("mismatched session RLS assertion failed: %v", err)
		}
	})

	t.Run("transaction-local settings do not leak", func(t *testing.T) {
		connection, err := pool.pool.Acquire(ctx)
		if err != nil {
			t.Fatal("could not acquire API connection")
		}
		defer connection.Release()

		transaction, err := connection.Begin(ctx)
		if err != nil {
			t.Fatal("could not begin context leak assertion")
		}
		setLocalGUC(t, transaction, ctx, "app.current_auth_subject_id", fixture.subjectA)
		setLocalGUC(t, transaction, ctx, "app.current_auth_session_id", fixture.sessionA)
		setLocalGUC(t, transaction, ctx, "app.current_organization_id", fixture.organizationA)
		if err := transaction.Commit(ctx); err != nil {
			t.Fatal("could not commit local context assertion")
		}

		transaction, err = connection.Begin(ctx)
		if err != nil {
			t.Fatal("could not begin fresh context assertion")
		}
		defer func() { _ = transaction.Rollback(ctx) }()
		var subjectIsNull, sessionIsNull, organizationIsNull bool
		scanRow(t, transaction, ctx, "select app.current_auth_subject_id() is null", &subjectIsNull)
		scanRow(t, transaction, ctx, "select app.current_auth_session_id() is null", &sessionIsNull)
		scanRow(t, transaction, ctx, "select app.current_tenant_id() is null", &organizationIsNull)
		if !subjectIsNull || !sessionIsNull || !organizationIsNull {
			t.Fatal("transaction-local identity context leaked to a later transaction")
		}
	})
}

type phase2CFixture struct {
	organizationA string
	organizationB string
	profileA      string
	profileB      string
	subjectA      string
	subjectB      string
	sessionA      string
	sessionB      string
}

func createPhase2CFixture(t *testing.T, pool *pgxpool.Pool, ctx context.Context) phase2CFixture {
	t.Helper()
	transaction, err := pool.Begin(ctx)
	if err != nil {
		t.Fatal("could not begin phase 2C fixture")
	}
	defer func() { _ = transaction.Rollback(ctx) }()

	if _, err := transaction.Exec(ctx, "set local session_replication_role = replica"); err != nil {
		t.Fatal("could not prepare phase 2C fictional identity fixture")
	}

	fixture := phase2CFixture{}
	if err := transaction.QueryRow(ctx, `
		select gen_random_uuid()::text, gen_random_uuid()::text,
		       gen_random_uuid()::text, gen_random_uuid()::text,
		       gen_random_uuid()::text, gen_random_uuid()::text,
		       gen_random_uuid()::text, gen_random_uuid()::text
	`).Scan(
		&fixture.organizationA,
		&fixture.organizationB,
		&fixture.profileA,
		&fixture.profileB,
		&fixture.subjectA,
		&fixture.subjectB,
		&fixture.sessionA,
		&fixture.sessionB,
	); err != nil {
		t.Fatal("could not create phase 2C fixture IDs")
	}

	for _, organization := range []string{fixture.organizationA, fixture.organizationB} {
		if _, err := transaction.Exec(ctx,
			"insert into app.organizations (id, name, timezone, status) values ($1, $2, 'UTC', 'active')",
			organization, "Phase 2C fictional organization",
		); err != nil {
			t.Fatal("could not create fictional organization")
		}
	}
	for _, profile := range []struct {
		id      string
		subject string
	}{
		{id: fixture.profileA, subject: fixture.subjectA},
		{id: fixture.profileB, subject: fixture.subjectB},
	} {
		if _, err := transaction.Exec(ctx,
			"insert into app.profiles (id, auth_user_id, full_name) values ($1, $2, 'Fictional profile')",
			profile.id, profile.subject,
		); err != nil {
			t.Fatal("could not create fictional profile")
		}
	}
	for _, membership := range []struct {
		organization string
		profile      string
	}{
		{organization: fixture.organizationA, profile: fixture.profileA},
		{organization: fixture.organizationB, profile: fixture.profileB},
	} {
		if _, err := transaction.Exec(ctx, `
			insert into app.organization_memberships (organization_id, profile_id, role, status)
			values ($1, $2, 'athlete', 'active')
		`, membership.organization, membership.profile); err != nil {
			t.Fatal("could not create fictional membership")
		}
	}
	for _, session := range []struct {
		id      string
		profile string
	}{
		{id: fixture.sessionA, profile: fixture.profileA},
		{id: fixture.sessionB, profile: fixture.profileB},
	} {
		if _, err := transaction.Exec(ctx,
			"insert into app.auth_sessions (session_id, profile_id, assurance_level) values ($1, $2, 'aal1')",
			session.id, session.profile,
		); err != nil {
			t.Fatal("could not create fictional session")
		}
	}
	if err := transaction.Commit(ctx); err != nil {
		t.Fatal("could not commit phase 2C fixture")
	}

	return fixture
}

func (fixture phase2CFixture) remove(t *testing.T, pool *pgxpool.Pool, ctx context.Context) {
	t.Helper()
	for _, statement := range []struct {
		query string
		id    string
	}{
		{"delete from app.auth_sessions where session_id = $1", fixture.sessionA},
		{"delete from app.auth_sessions where session_id = $1", fixture.sessionB},
		{"delete from app.organization_memberships where profile_id = $1", fixture.profileA},
		{"delete from app.organization_memberships where profile_id = $1", fixture.profileB},
		{"delete from app.organizations where id = $1", fixture.organizationA},
		{"delete from app.organizations where id = $1", fixture.organizationB},
		{"delete from app.profiles where id = $1", fixture.profileA},
		{"delete from app.profiles where id = $1", fixture.profileB},
	} {
		if _, err := pool.Exec(ctx, statement.query, statement.id); err != nil {
			t.Fatal("could not remove phase 2C fictional fixture")
		}
	}
}

func assertAuthSessionInsert(
	t *testing.T,
	pool *pgxpool.Pool,
	ctx context.Context,
	profileID string,
	assuranceLevel string,
	revokedAt *string,
	revocationReason *string,
	wantError bool,
) {
	t.Helper()
	transaction, err := pool.Begin(ctx)
	if err != nil {
		t.Fatal("could not begin auth session constraint assertion")
	}
	defer func() { _ = transaction.Rollback(ctx) }()

	_, err = transaction.Exec(ctx, `
		insert into app.auth_sessions (session_id, profile_id, assurance_level, revoked_at, revocation_reason)
		values (gen_random_uuid(), $1, $2, $3::timestamptz, $4)
	`, profileID, assuranceLevel, revokedAt, revocationReason)
	if (err != nil) != wantError {
		t.Fatalf("auth session insert error = %v, want error = %v", err, wantError)
	}
}

func assertNoPublicTablePrivilege(t *testing.T, pool *pgxpool.Pool, ctx context.Context, relation string) {
	t.Helper()
	var publicPrivilegeCount int
	scanRowWithArguments(t, pool, ctx, `
		select count(*)
		from pg_class as relation
		cross join lateral aclexplode(coalesce(relation.relacl, acldefault('r', relation.relowner))) as privilege
		where relation.oid = $1::regclass
		  and privilege.grantee = 0
	`, []any{relation}, &publicPrivilegeCount)
	if publicPrivilegeCount != 0 {
		t.Fatalf("%s grants table privileges to PUBLIC", relation)
	}
}

func assertNoPublicFunctionExecute(t *testing.T, pool *pgxpool.Pool, ctx context.Context, function string) {
	t.Helper()
	var publicExecuteCount int
	scanRowWithArguments(t, pool, ctx, `
		select count(*)
		from pg_proc as procedure
		cross join lateral aclexplode(coalesce(procedure.proacl, acldefault('f', procedure.proowner))) as privilege
		where procedure.oid = $1::regprocedure
		  and privilege.grantee = 0
		  and privilege.privilege_type = 'EXECUTE'
	`, []any{function}, &publicExecuteCount)
	if publicExecuteCount != 0 {
		t.Fatalf("%s grants EXECUTE to PUBLIC", function)
	}
}

func setLocalGUC(t *testing.T, transaction pgx.Tx, ctx context.Context, name, value string) {
	t.Helper()
	if _, err := transaction.Exec(ctx, "select set_config($1, $2, true)", name, value); err != nil {
		t.Fatal("could not set transaction-local context")
	}
}

func mustPGUUID(t *testing.T, value string) pgtype.UUID {
	t.Helper()
	var identifier pgtype.UUID
	if err := identifier.Scan(value); err != nil {
		t.Fatal("could not parse fictional UUID")
	}
	return identifier
}

func containsString(values []string, target string) bool {
	return strings.Contains(strings.Join(values, ","), target)
}

func stringPointer(value string) *string {
	return &value
}
