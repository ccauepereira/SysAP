package database

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/ccauepereira/SysAP/apps/api/internal/platform/httpserver"
)

var clientRoles = []string{"anon", "authenticated", "service_role"}

func TestPostgresFoundation(t *testing.T) {
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
	if err := adminPool.Ping(ctx); err != nil {
		t.Fatal("integration database did not respond to administration ping")
	}

	pool, err := NewPool(ctx, databaseURL)
	if err != nil {
		t.Fatal("could not prepare integration API pool")
	}
	defer pool.Close()
	if err := pool.Ping(ctx); err != nil {
		t.Fatal("integration database did not respond to API ping")
	}

	apiConnection, err := pool.pool.Acquire(ctx)
	if err != nil {
		t.Fatal("could not acquire integration API connection")
	}
	defer apiConnection.Release()

	t.Run("uses the restricted API role", func(t *testing.T) {
		var currentUser string
		scanRow(t, apiConnection, ctx, "select current_user", &currentUser)
		if currentUser != "sysap_api" {
			t.Fatalf("current database role = %q, want sysap_api", currentUser)
		}

		var superuser, createDatabase, createRole, inherit, login, replication, bypassRLS bool
		scanRow(t, adminPool, ctx, `
			select rolsuper, rolcreatedb, rolcreaterole, rolinherit,
			       rolcanlogin, rolreplication, rolbypassrls
			from pg_roles
			where rolname = 'sysap_api'
		`, &superuser, &createDatabase, &createRole, &inherit, &login, &replication, &bypassRLS)
		if superuser || createDatabase || createRole || inherit || login || replication || bypassRLS {
			t.Fatal("sysap_api has privileges beyond the approved restricted role")
		}

		var membershipCount int
		scanRow(t, adminPool, ctx, `
			select count(*)
			from pg_auth_members
			join pg_roles on pg_roles.oid = pg_auth_members.member
			where pg_roles.rolname = 'sysap_api'
		`, &membershipCount)
		if membershipCount != 0 {
			t.Fatal("sysap_api retains an unexpected role membership")
		}
	})

	t.Run("creates only the approved private foundation", func(t *testing.T) {
		var schemaExists, tableExists, rowLevelSecurityEnabled bool
		scanRow(t, adminPool, ctx,
			"select exists(select 1 from pg_namespace where nspname = 'app')",
			&schemaExists,
		)
		scanRow(t, adminPool, ctx,
			"select to_regclass('app.bootstrap_metadata') is not null",
			&tableExists,
		)
		scanRow(t, adminPool, ctx, `
			select c.relrowsecurity
			from pg_class c
			join pg_namespace n on n.oid = c.relnamespace
			where n.nspname = 'app' and c.relname = 'bootstrap_metadata'
		`, &rowLevelSecurityEnabled)
		if !schemaExists || !tableExists || !rowLevelSecurityEnabled {
			t.Fatal("database foundation objects are missing")
		}

		var rowCount, minimumVersion, maximumVersion int
		var everyRowIsSingleton bool
		var initializedAtType string
		scanRow(t, apiConnection, ctx, `
			select count(*), min(schema_version), max(schema_version),
			       bool_and(singleton), pg_typeof(min(initialized_at))::text
			from app.bootstrap_metadata
		`, &rowCount, &minimumVersion, &maximumVersion, &everyRowIsSingleton, &initializedAtType)
		if rowCount != 1 || minimumVersion != 1 || maximumVersion != 1 ||
			!everyRowIsSingleton || initializedAtType != "timestamp with time zone" {
			t.Fatal("bootstrap metadata does not match the singleton foundation contract")
		}

		var primaryKeyCount, checkConstraintCount int
		scanRow(t, adminPool, ctx, `
			select count(*) filter (where contype = 'p'),
			       count(*) filter (where contype = 'c')
			from pg_constraint
			where conrelid = 'app.bootstrap_metadata'::regclass
		`, &primaryKeyCount, &checkConstraintCount)
		if primaryKeyCount != 1 || checkConstraintCount < 2 {
			t.Fatal("bootstrap metadata constraints differ from the foundation contract")
		}
	})

	t.Run("grants exactly the approved schema and table privileges", func(t *testing.T) {
		assertPrivileges(t, adminPool, ctx, "sysap_api", true, true)
		for _, role := range clientRoles {
			assertPrivileges(t, adminPool, ctx, role, false, false)
		}

		var unexpectedAPIGrants, unexpectedClientGrants int
		scanRow(t, adminPool, ctx, `
			select count(*)
			from information_schema.table_privileges
			where table_schema = 'app'
			  and grantee = 'sysap_api'
			  and not (
			    (table_name = 'bootstrap_metadata' and privilege_type = 'SELECT') or
			    (table_name = 'organizations' and privilege_type in ('SELECT', 'UPDATE')) or
			    (table_name = 'profiles' and privilege_type in ('SELECT', 'UPDATE')) or
			    (table_name = 'organization_memberships' and privilege_type in ('SELECT', 'INSERT', 'UPDATE')) or
			    (table_name = 'athletes' and privilege_type in ('SELECT', 'INSERT', 'UPDATE')) or
			    (table_name = 'trainer_athlete_assignments' and privilege_type in ('SELECT', 'INSERT', 'UPDATE')) or
			    (table_name = 'athlete_invitations' and privilege_type in ('SELECT', 'INSERT', 'UPDATE')) or
			    (table_name = 'identity_operations' and privilege_type in ('SELECT', 'INSERT', 'UPDATE')) or
			    (table_name = 'outbox_events' and privilege_type in ('SELECT', 'INSERT', 'UPDATE')) or
			    (table_name = 'idempotency_records' and privilege_type in ('SELECT', 'INSERT', 'UPDATE')) or
			    (table_name = 'security_audit_events' and privilege_type in ('SELECT', 'INSERT'))
			  )
		`, &unexpectedAPIGrants)
		scanRowWithArguments(t, adminPool, ctx, `
			select count(*)
			from information_schema.table_privileges
			where table_schema = 'app'
			  and grantee = any($1::text[])
		`, []any{clientRoles}, &unexpectedClientGrants)
		if unexpectedAPIGrants != 0 || unexpectedClientGrants != 0 {
			t.Fatal("private schema contains an unexpected explicit grant")
		}
	})

	t.Run("client roles receive an actual permission denial", func(t *testing.T) {
		for _, role := range clientRoles {
			assertClientSelectDenied(t, adminPool, ctx, role)
		}
	})

	t.Run("has exactly the approved API select policy", func(t *testing.T) {
		var totalPolicies, approvedPolicies int
		scanRow(t, adminPool, ctx, `
			select count(*),
			       count(*) filter (
			         where cmd = 'SELECT'
			           and roles = array['sysap_api']::name[]
			           and with_check is null
			       )
			from pg_policies
			where schemaname = 'app'
			  and tablename = 'bootstrap_metadata'
		`, &totalPolicies, &approvedPolicies)
		if totalPolicies != 1 || approvedPolicies != 1 {
			t.Fatal("bootstrap metadata policies differ from the exact approved policy set")
		}
	})

	t.Run("owns and applies safe default privileges", func(t *testing.T) {
		var executor string
		scanRow(t, adminPool, ctx, "select current_user", &executor)
		if executor != "postgres" {
			t.Fatal("local migration executor differs from the expected role")
		}

		var tableOwner string
		scanRow(t, adminPool, ctx, `
			select pg_get_userbyid(pg_class.relowner)
			from pg_class
			where pg_class.oid = 'app.bootstrap_metadata'::regclass
		`, &tableOwner)
		if tableOwner != executor {
			t.Fatal("foundation objects are not owned by the expected migration executor")
		}

		var defaultACLCount, forbiddenGrantCount int
		scanRow(t, adminPool, ctx, `
			select count(*)
			from pg_default_acl
			join pg_roles as owner_role on owner_role.oid = pg_default_acl.defaclrole
			left join pg_namespace on pg_namespace.oid = pg_default_acl.defaclnamespace
			where owner_role.rolname = current_user
			  and (pg_default_acl.defaclnamespace = 0 or pg_namespace.nspname = 'app')
			  and pg_default_acl.defaclobjtype in ('r', 'S', 'f')
		`, &defaultACLCount)
		scanRowWithArguments(t, adminPool, ctx, `
			select count(*)
			from pg_default_acl
			join pg_roles as owner_role on owner_role.oid = pg_default_acl.defaclrole
			left join pg_namespace on pg_namespace.oid = pg_default_acl.defaclnamespace
			cross join lateral aclexplode(pg_default_acl.defaclacl) as acl
			left join pg_roles as grantee_role on grantee_role.oid = acl.grantee
			where owner_role.rolname = current_user
			  and (pg_default_acl.defaclnamespace = 0 or pg_namespace.nspname = 'app')
			  and pg_default_acl.defaclobjtype in ('r', 'S', 'f')
			  and (acl.grantee = 0 or grantee_role.rolname = any($1::text[]))
		`, []any{clientRoles}, &forbiddenGrantCount)
		if defaultACLCount == 0 {
			t.Fatal("migration executor has no inspectable effective default privilege entries")
		}
		if forbiddenGrantCount != 0 {
			t.Fatal("default privileges grant access to PUBLIC or a client role")
		}

		var unexpectedAppDefaultACLOwners int
		scanRow(t, adminPool, ctx, `
			select count(*)
			from pg_default_acl
			join pg_roles as owner_role on owner_role.oid = pg_default_acl.defaclrole
			join pg_namespace on pg_namespace.oid = pg_default_acl.defaclnamespace
			where pg_namespace.nspname = 'app'
			  and owner_role.rolname <> current_user
		`, &unexpectedAppDefaultACLOwners)
		if unexpectedAppDefaultACLOwners != 0 {
			t.Fatal("app default privileges have an unexpected owner")
		}

		assertDefaultPrivilegesOnNewObjects(t, adminPool, ctx)
	})

	t.Run("does not expose app through the Data API", func(t *testing.T) {
		assertAppSchemaNotExposed(t)
	})

	t.Run("returns the exact ready response without connection details", func(t *testing.T) {
		var logOutput bytes.Buffer
		logger := slog.New(slog.NewJSONHandler(&logOutput, nil))
		handler := httpserver.New(pool, logger, 2*time.Second)
		response := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodGet, "/readyz", nil)

		handler.ServeHTTP(response, request)

		if response.Code != http.StatusOK {
			t.Fatal("readiness did not return HTTP 200")
		}
		wantBody := "{\"status\":\"ready\",\"service\":\"sysap-api\",\"checks\":{\"database\":\"up\"}}\n"
		if response.Body.String() != wantBody {
			t.Fatal("readiness body does not match the approved contract")
		}
		assertNoConnectionDetails(t, response.Body.Bytes(), databaseURL)
		assertNoConnectionDetails(t, logOutput.Bytes(), databaseURL)
	})
}

type rowQuerier interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}

func assertPrivileges(
	t *testing.T,
	querier rowQuerier,
	ctx context.Context,
	role string,
	wantSchemaUsage bool,
	wantTableSelect bool,
) {
	t.Helper()

	for _, privilege := range []string{"usage", "create"} {
		want := wantSchemaUsage && privilege == "usage"
		var got bool
		scanRowWithArguments(t, querier, ctx,
			"select has_schema_privilege($1, 'app', $2)",
			[]any{role, privilege},
			&got,
		)
		if got != want {
			t.Fatalf("role %q has an unexpected schema privilege", role)
		}
	}

	for _, privilege := range []string{
		"select", "insert", "update", "delete", "truncate", "references", "trigger",
	} {
		want := wantTableSelect && privilege == "select"
		var got bool
		scanRowWithArguments(t, querier, ctx,
			"select has_table_privilege($1, 'app.bootstrap_metadata', $2)",
			[]any{role, privilege},
			&got,
		)
		if got != want {
			t.Fatalf("role %q has an unexpected table privilege", role)
		}
	}
}

func assertClientSelectDenied(
	t *testing.T,
	pool *pgxpool.Pool,
	ctx context.Context,
	role string,
) {
	t.Helper()

	transaction, err := pool.Begin(ctx)
	if err != nil {
		t.Fatal("could not begin client privilege assertion")
	}
	defer func() { _ = transaction.Rollback(ctx) }()

	if _, err := transaction.Exec(ctx, "set local role "+pgx.Identifier{role}.Sanitize()); err != nil {
		t.Fatal("could not assume local client role")
	}
	if _, err := transaction.Exec(ctx, "select * from app.bootstrap_metadata"); err == nil {
		t.Fatalf("client role %q unexpectedly read private metadata", role)
	} else {
		var postgresError *pgconn.PgError
		if !errors.As(err, &postgresError) {
			t.Fatal("client role denial failed because the connection or query transport broke")
		}
		switch {
		case postgresError.Code == "42501":
			return
		case postgresError.Code == "42P01":
			t.Fatal("client role denial queried a missing object")
		case postgresError.Code == "42601":
			t.Fatal("client role denial query has invalid syntax")
		case strings.HasPrefix(postgresError.Code, "08"):
			t.Fatal("client role denial failed because the database connection broke")
		default:
			t.Fatal("client role denial returned an unexpected PostgreSQL error category")
		}
	}
}

func assertDefaultPrivilegesOnNewObjects(
	t *testing.T,
	pool *pgxpool.Pool,
	ctx context.Context,
) {
	transaction, err := pool.Begin(ctx)
	if err != nil {
		t.Fatal("could not begin default privilege assertion")
	}
	defer func() { _ = transaction.Rollback(ctx) }()

	statements := []string{
		"create table app.sysap_acl_probe_table (id integer primary key)",
		"create sequence app.sysap_acl_probe_sequence",
		"create function app.sysap_acl_probe_function() returns integer language sql as 'select 1'",
	}
	for _, statement := range statements {
		if _, err := transaction.Exec(ctx, statement); err != nil {
			t.Fatal("could not create transactional default privilege probe")
		}
	}

	for _, role := range clientRoles {
		for _, privilege := range []string{
			"select", "insert", "update", "delete", "truncate", "references", "trigger",
		} {
			var granted bool
			scanRowWithArguments(t, transaction, ctx,
				"select has_table_privilege($1, 'app.sysap_acl_probe_table', $2)",
				[]any{role, privilege},
				&granted,
			)
			if granted {
				t.Fatalf("role %q inherited an unexpected table default privilege", role)
			}
		}

		for _, privilege := range []string{"usage", "select", "update"} {
			var granted bool
			scanRowWithArguments(t, transaction, ctx,
				"select has_sequence_privilege($1, 'app.sysap_acl_probe_sequence', $2)",
				[]any{role, privilege},
				&granted,
			)
			if granted {
				t.Fatalf("role %q inherited an unexpected sequence default privilege", role)
			}
		}

		var functionExecute bool
		scanRowWithArguments(t, transaction, ctx,
			"select has_function_privilege($1, 'app.sysap_acl_probe_function()', 'execute')",
			[]any{role},
			&functionExecute,
		)
		if functionExecute {
			t.Fatalf("role %q inherited an unexpected function default privilege", role)
		}
	}

	var publicOrClientGrantCount int
	scanRowWithArguments(t, transaction, ctx, `
		select count(*)
		from (
		  select acl.grantee
		  from pg_class
		  join pg_namespace on pg_namespace.oid = pg_class.relnamespace
		  cross join lateral aclexplode(
		    coalesce(
		      pg_class.relacl,
			      acldefault(
			        (case when pg_class.relkind = 'S' then 's' else 'r' end)::"char",
			        pg_class.relowner
			      )
		    )
		  ) as acl
		  where pg_namespace.nspname = 'app'
		    and pg_class.relname in ('sysap_acl_probe_table', 'sysap_acl_probe_sequence')
		  union all
		  select acl.grantee
		  from pg_proc
		  join pg_namespace on pg_namespace.oid = pg_proc.pronamespace
		  cross join lateral aclexplode(coalesce(pg_proc.proacl, acldefault('f', pg_proc.proowner))) as acl
		  where pg_namespace.nspname = 'app'
		    and pg_proc.proname = 'sysap_acl_probe_function'
		) as grants
	left join pg_roles on pg_roles.oid = grants.grantee
	where grants.grantee = 0 or pg_roles.rolname = any($1::text[])
	`, []any{clientRoles}, &publicOrClientGrantCount)
	if publicOrClientGrantCount != 0 {
		t.Fatal("new private objects grant privileges to PUBLIC or client roles")
	}

	if err := transaction.Rollback(ctx); err != nil {
		t.Fatal("could not roll back default privilege probes")
	}
	for _, query := range []string{
		"select to_regclass('app.sysap_acl_probe_table') is null",
		"select to_regclass('app.sysap_acl_probe_sequence') is null",
		"select to_regprocedure('app.sysap_acl_probe_function()') is null",
	} {
		var removed bool
		scanRow(t, pool, ctx, query, &removed)
		if !removed {
			t.Fatal("transactional default privilege probe was not removed")
		}
	}
}

func scanRow(
	t *testing.T,
	querier rowQuerier,
	ctx context.Context,
	query string,
	destination ...any,
) {
	t.Helper()
	if err := querier.QueryRow(ctx, query).Scan(destination...); err == nil {
		return
	}
	t.Fatal("database foundation assertion failed")
}

func scanRowWithArguments(
	t *testing.T,
	querier rowQuerier,
	ctx context.Context,
	query string,
	arguments []any,
	destination ...any,
) {
	t.Helper()
	if err := querier.QueryRow(ctx, query, arguments...).Scan(destination...); err == nil {
		return
	}
	t.Fatal("database foundation assertion failed")
}

func assertAppSchemaNotExposed(t *testing.T) {
	t.Helper()
	configPath := filepath.Join("..", "..", "..", "..", "..", "infra", "supabase", "config.toml")
	contents, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatal("could not read local Supabase configuration")
	}

	inAPISection := false
	foundSchemas := false
	for _, line := range strings.Split(string(contents), "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "[") {
			inAPISection = trimmed == "[api]"
			continue
		}
		if inAPISection && strings.HasPrefix(trimmed, "schemas =") {
			foundSchemas = true
			if strings.Contains(trimmed, "\"app\"") {
				t.Fatal("private schema is exposed by the Data API configuration")
			}
		}
	}
	if !foundSchemas {
		t.Fatal("Data API schemas are not explicitly configured")
	}
}

func assertNoConnectionDetails(t *testing.T, output []byte, databaseURL string) {
	t.Helper()
	candidates := []string{databaseURL}
	parsed, err := url.Parse(databaseURL)
	if err == nil {
		candidates = append(candidates, parsed.Host, parsed.User.Username())
		if password, exists := parsed.User.Password(); exists {
			candidates = append(candidates, password)
		}
	}

	for _, candidate := range candidates {
		if candidate != "" && bytes.Contains(output, []byte(candidate)) {
			t.Fatal("response or log exposed connection details")
		}
	}
}
