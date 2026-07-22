package database

import (
	"context"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func TestPhase2BIdentity(t *testing.T) {
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

	t.Run("Structure", func(t *testing.T) {
		tables := []string{
			"organizations",
			"profiles",
			"organization_memberships",
			"athletes",
			"trainer_athlete_assignments",
			"athlete_invitations",
			"identity_operations",
			"outbox_events",
			"idempotency_records",
			"security_audit_events",
		}
		for _, table := range tables {
			var rlsEnabled, forceRls bool
			scanRow(t, adminPool, ctx, `
				select c.relrowsecurity, c.relforcerowsecurity
				from pg_class c
				join pg_namespace n on n.oid = c.relnamespace
				where n.nspname = 'app' and c.relname = $1
			`, []any{table}, &rlsEnabled, &forceRls)
			if !rlsEnabled || !forceRls {
				t.Fatalf("Table %s must have RLS and FORCE RLS enabled", table)
			}
		}

		var secretColumns int
		scanRow(t, adminPool, ctx, `
			select count(*)
			from information_schema.columns
			where table_schema = 'app'
			  and column_name in ('password', 'passwd', 'otp', 'totp', 'token', 'access_token', 'refresh_token', 'secret')
		`, nil, &secretColumns)
		if secretColumns > 0 {
			t.Fatal("Schema app contains forbidden secret columns")
		}
	})

	t.Run("Sysap API Profiles Insert Blocked", func(t *testing.T) {
		tx, _ := pool.pool.Begin(ctx)
		defer tx.Rollback(ctx)

		// Try to insert a profile directly via sysap_api
		_, err := tx.Exec(ctx, "insert into app.profiles (full_name) values ('Orphan')")
		if err == nil {
			t.Fatal("Expected error when inserting profile via sysap_api directly")
		}
		if !strings.Contains(err.Error(), "permission denied") {
			t.Fatalf("Expected permission denied error, got %v", err)
		}
	})

	t.Run("Tenant Context Invalid", func(t *testing.T) {
		tx, _ := pool.pool.Begin(ctx)
		defer tx.Rollback(ctx)

		_, err := tx.Exec(ctx, "set local app.current_organization_id = 'invalid-uuid'")
		if err == nil {
			// If it somehow allows setting invalid text, reading it should fail the uuid cast.
			var name string
			err = tx.QueryRow(ctx, "select name from app.organizations").Scan(&name)
			if err == nil {
				t.Fatal("Expected failure when using invalid tenant ID")
			}
		}
	})

	t.Run("Enrollment", func(t *testing.T) {
		tx, _ := adminPool.Begin(ctx)
		defer tx.Rollback(ctx)

		var orgId, profileId, memberId string
		tx.QueryRow(ctx, "insert into app.organizations (name, timezone, status) values ('Test', 'America/Fortaleza', 'active') returning id").Scan(&orgId)
		tx.QueryRow(ctx, "insert into app.profiles (full_name) values ('Athlete') returning id").Scan(&profileId)
		tx.QueryRow(ctx, "insert into app.organization_memberships (organization_id, profile_id, role, status) values ($1, $2, 'athlete', 'active') returning id", orgId, profileId).Scan(&memberId)

		// Test valid enrollments
		for _, valid := range []string{"2026000001", "2026999999"} {
			_, err := tx.Exec(ctx, "insert into app.athletes (organization_id, membership_id, enrollment_number) values ($1, $2, $3)", orgId, memberId, valid)
			if err != nil {
				t.Fatalf("Failed to insert valid enrollment %s: %v", valid, err)
			}
			tx.Exec(ctx, "delete from app.athletes where enrollment_number = $1", valid)
		}

		// Test invalid enrollments
		for _, invalid := range []string{"202600001", "20260000001", "2026ABC001", "2026-000001"} {
			_, err := tx.Exec(ctx, "insert into app.athletes (organization_id, membership_id, enrollment_number) values ($1, $2, $3)", orgId, memberId, invalid)
			if err == nil {
				t.Fatalf("Expected constraint violation for invalid enrollment %s", invalid)
			}
		}
	})

	t.Run("Tenant Isolation", func(t *testing.T) {
		txAdmin, _ := adminPool.Begin(ctx)
		defer txAdmin.Rollback(ctx)

		var orgA, orgB string
		txAdmin.QueryRow(ctx, "insert into app.organizations (name, timezone, status) values ('Org A', 'America/Fortaleza', 'active') returning id").Scan(&orgA)
		txAdmin.QueryRow(ctx, "insert into app.organizations (name, timezone, status) values ('Org B', 'America/Fortaleza', 'active') returning id").Scan(&orgB)

		tx, _ := pool.pool.Begin(ctx)
		defer tx.Rollback(ctx)

		// Test reading only Tenant A
		tx.Exec(ctx, "set local app.current_organization_id = '"+orgA+"'")
		var count int
		tx.QueryRow(ctx, "select count(*) from app.organizations").Scan(&count)
		if count != 1 {
			t.Fatal("Tenant A should only see Org A")
		}

		// Test modifying Tenant A to Tenant B
		_, err := tx.Exec(ctx, "update app.organizations set id = $1", orgB)
		if err == nil {
			t.Fatal("Tenant A should not be able to modify its ID to Tenant B")
		}

		// Test no leakage after Commit/Rollback
		tx.Rollback(ctx)
		tx2, _ := pool.pool.Begin(ctx)
		defer tx2.Rollback(ctx)
		err = tx2.QueryRow(ctx, "select count(*) from app.organizations").Scan(&count)
		if err != nil || count != 0 {
			t.Fatalf("Context leaked or query failed: %v, count: %d", err, count)
		}
	})

	t.Run("Security Audit Recursive JSON", func(t *testing.T) {
		txAdmin, _ := adminPool.Begin(ctx)
		defer txAdmin.Rollback(ctx)
		var orgA string
		txAdmin.QueryRow(ctx, "insert into app.organizations (name, timezone, status) values ('Org A', 'UTC', 'active') returning id").Scan(&orgA)

		tx, _ := pool.pool.Begin(ctx)
		defer tx.Rollback(ctx)
		tx.Exec(ctx, "set local app.current_organization_id = '"+orgA+"'")

		// Valid metadata
		_, err := tx.Exec(ctx, "insert into app.security_audit_events (organization_id, event_type, result, reason_code, request_id, network_fingerprint, metadata) values ($1, 'test', 'success', 'none', 'req', 'net', '{\"safe\": \"value\"}'::jsonb)", orgA)
		if err != nil {
			t.Fatalf("Expected success for valid JSON, got %v", err)
		}

		// Invalid metadata tests
		invalidJSONs := []string{
			`{"password": "123"}`,
			`{"nested": {"access_token": "abc"}}`,
			`{"items": [{"RefreshToken": "x"}]}`,
			`{"AUTHORIZATION": "Bearer x"}`,
			`{"user": {"phone_number": "+000000000"}}`,
		}
		for _, invalid := range invalidJSONs {
			_, err = tx.Exec(ctx, "insert into app.security_audit_events (organization_id, event_type, result, reason_code, request_id, network_fingerprint, metadata) values ($1, 'test', 'success', 'none', 'req', 'net', $2::jsonb)", orgA, invalid)
			if err == nil {
				t.Fatalf("Expected failure for JSON payload containing secrets: %s", invalid)
			}
			if strings.Contains(err.Error(), "123") || strings.Contains(err.Error(), "abc") {
				t.Fatal("Error message must not echo the secret value")
			}
		}

		// Verify Append Only
		_, err = tx.Exec(ctx, "update app.security_audit_events set event_type = 'hack'")
		if err == nil {
			t.Fatal("Expected error when updating audit event")
		}
		_, err = tx.Exec(ctx, "delete from app.security_audit_events")
		if err == nil {
			t.Fatal("Expected error when deleting audit event")
		}
	})

	t.Run("Concurrent Last Owner Protection", func(t *testing.T) {
		// Create org and 2 active owners via admin
		txAdmin, _ := adminPool.Begin(ctx)
		var orgA, profileA, profileB, ownerA, ownerB string
		txAdmin.QueryRow(ctx, "insert into app.organizations (name, timezone, status) values ('Org A', 'UTC', 'active') returning id").Scan(&orgA)
		txAdmin.QueryRow(ctx, "insert into app.profiles (full_name) values ('User A') returning id").Scan(&profileA)
		txAdmin.QueryRow(ctx, "insert into app.profiles (full_name) values ('User B') returning id").Scan(&profileB)
		txAdmin.QueryRow(ctx, "insert into app.organization_memberships (organization_id, profile_id, role, status) values ($1, $2, 'owner', 'active') returning id", orgA, profileA).Scan(&ownerA)
		txAdmin.QueryRow(ctx, "insert into app.organization_memberships (organization_id, profile_id, role, status) values ($1, $2, 'owner', 'active') returning id", orgA, profileB).Scan(&ownerB)
		txAdmin.Commit(ctx)

		// Setup 2 concurrent transactions attempting to suspend/demote
		var wg sync.WaitGroup
		wg.Add(2)
		
		results := make(chan error, 2)
		ready := make(chan struct{})

		// Transaction 1
		go func() {
			defer wg.Done()
			conn1, _ := adminPool.Acquire(ctx)
			defer conn1.Release()
			tx1, _ := conn1.Begin(ctx)
			defer tx1.Rollback(ctx)
			
			<-ready // Wait for coordination
			_, err := tx1.Exec(ctx, "update app.organization_memberships set status = 'suspended' where id = $1", ownerA)
			if err == nil {
				tx1.Commit(ctx)
			}
			results <- err
		}()

		// Transaction 2
		go func() {
			defer wg.Done()
			conn2, _ := adminPool.Acquire(ctx)
			defer conn2.Release()
			tx2, _ := conn2.Begin(ctx)
			defer tx2.Rollback(ctx)
			
			<-ready // Wait for coordination
			_, err := tx2.Exec(ctx, "update app.organization_memberships set role = 'trainer' where id = $1", ownerB)
			if err == nil {
				tx2.Commit(ctx)
			}
			results <- err
		}()

		// Fire both
		close(ready)
		wg.Wait()
		close(results)

		successCount := 0
		errorCount := 0
		for err := range results {
			if err == nil {
				successCount++
			} else {
				errorCount++
			}
		}

		if successCount != 1 || errorCount != 1 {
			t.Fatalf("Expected exactly 1 success and 1 error due to serialization, got %d successes and %d errors", successCount, errorCount)
		}

		// Ensure at least one active owner remains
		var activeCount int
		adminPool.QueryRow(ctx, "select count(*) from app.organization_memberships where organization_id = $1 and role = 'owner' and status = 'active'", orgA).Scan(&activeCount)
		if activeCount < 1 {
			t.Fatal("Last owner protection failed, organization has no active owners left")
		}
	})
}
