package identity

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/ccauepereira/SysAP/apps/api/internal/platform/database"
)

func TestLoginIntegration(t *testing.T) {
	databaseURL := os.Getenv("SYSAP_TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("integration database is not configured")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()
	admin, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		t.Fatal("could not open local administration pool")
	}
	defer admin.Close()
	pool, err := database.NewPool(ctx, databaseURL)
	if err != nil {
		t.Fatal("could not open local API pool")
	}
	defer pool.Close()

	clock := time.Date(2026, time.July, 30, 12, 0, 0, 0, time.UTC)
	now := func() time.Time { return clock }

	t.Run("malformed enrollment is denied before provider use", func(t *testing.T) {
		provider := &loginProviderStub{}
		handler := newLoginHandler(pool, []byte("12345678901234567890123456789012"), provider, now)
		response := loginRequestForTest(handler, "short", "fixture-password", "127.0.0.1:1000")
		assertInvalidCredentials(t, response)
		if provider.calls.Load() != 0 {
			t.Fatal("malformed enrollment called the provider")
		}
	})

	t.Run("unknown enrollment has the same public failure as a bad password", func(t *testing.T) {
		fixture := createLoginFixture(t, admin, ctx, "athlete", "active", "active", false)
		defer fixture.remove(t, admin, ctx)
		provider := &loginProviderStub{err: errLoginDenied}
		handler := newLoginHandler(pool, []byte("12345678901234567890123456789012"), provider, now)
		unknown := loginRequestForTest(handler, "9000000000", "fixture-password", "127.0.0.1:1001")
		badPassword := loginRequestForTest(handler, fixture.enrollment, "fixture-password", "127.0.0.1:1002")
		assertInvalidCredentials(t, unknown)
		assertInvalidCredentials(t, badPassword)
		if unknown.Body.String() != badPassword.Body.String() {
			t.Fatalf("unknown and bad-password bodies differ: %q != %q", unknown.Body.String(), badPassword.Body.String())
		}
		if provider.calls.Load() != 1 {
			t.Fatalf("provider calls = %d, want one for the known enrollment only", provider.calls.Load())
		}
	})

	for _, role := range []string{"athlete", "trainer", "owner"} {
		t.Run(role+" receives an active local session", func(t *testing.T) {
			fixture := createLoginFixture(t, admin, ctx, role, "active", "active", false)
			defer fixture.remove(t, admin, ctx)
			provider := &loginProviderStub{}
			handler := newLoginHandler(pool, []byte("12345678901234567890123456789012"), provider, now)
			response := loginRequestForTest(handler, fixture.enrollment, "fixture-password", "127.0.0.1:1003")
			if response.Code != http.StatusOK {
				t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
			}
			var body loginSuccessResponse
			if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
				t.Fatal("success is not JSON")
			}
			if body.ProfileID != fixture.profileID || body.OrganizationID != fixture.organizationID || body.Role != role || body.AAL != "aal1" || body.SessionID == uuid.Nil || body.AccessToken == "" || body.RefreshToken == "" || body.ExpiresIn <= 0 {
				t.Fatalf("unexpected login response: %+v", body)
			}
			var sessions int
			if err := admin.QueryRow(ctx, "select count(*) from app.auth_sessions where session_id = $1 and profile_id = $2 and revoked_at is null", body.SessionID, fixture.profileID).Scan(&sessions); err != nil || sessions != 1 {
				t.Fatalf("local session was not created: count=%d err=%v", sessions, err)
			}
			var auditCount int
			if err := admin.QueryRow(ctx, "select count(*) from app.security_audit_events where target_profile_id = $1 and event_type = 'login_success'", fixture.profileID).Scan(&auditCount); err != nil || auditCount != 1 {
				t.Fatalf("success audit missing: count=%d err=%v", auditCount, err)
			}
		})
	}

	for _, state := range []struct {
		name              string
		organizationState string
		membershipState   string
		suspendedProfile  bool
	}{
		{"suspended profile", "active", "active", true},
		{"suspended organization", "suspended", "active", false},
		{"inactive membership", "active", "suspended", false},
		{"pending activation", "active", "pending_activation", false},
	} {
		t.Run(state.name+" is indistinguishable from invalid credentials", func(t *testing.T) {
			fixture := createLoginFixture(t, admin, ctx, "athlete", state.organizationState, state.membershipState, state.suspendedProfile)
			defer fixture.remove(t, admin, ctx)
			provider := &loginProviderStub{}
			handler := newLoginHandler(pool, []byte("12345678901234567890123456789012"), provider, now)
			response := loginRequestForTest(handler, fixture.enrollment, "fixture-password", "127.0.0.1:1004")
			assertInvalidCredentials(t, response)
			if provider.calls.Load() != 0 {
				t.Fatal("ineligible local identity called the provider")
			}
		})
	}

	t.Run("five failures allow provider checks, sixth is blocked, and expiry resets the window", func(t *testing.T) {
		fixture := createLoginFixture(t, admin, ctx, "athlete", "active", "active", false)
		defer fixture.remove(t, admin, ctx)
		provider := &loginProviderStub{err: errLoginDenied}
		handler := newLoginHandler(pool, []byte("12345678901234567890123456789012"), provider, now)
		for attempt := 0; attempt < loginMaxFailures; attempt++ {
			assertInvalidCredentials(t, loginRequestForTest(handler, fixture.enrollment, "fixture-password", "127.0.0.1:1005"))
		}
		assertInvalidCredentials(t, loginRequestForTest(handler, fixture.enrollment, "fixture-password", "127.0.0.1:1005"))
		if provider.calls.Load() != loginMaxFailures {
			t.Fatalf("provider calls = %d, want %d", provider.calls.Load(), loginMaxFailures)
		}
		clock = clock.Add(loginBlockPeriod + time.Second)
		assertInvalidCredentials(t, loginRequestForTest(handler, fixture.enrollment, "fixture-password", "127.0.0.1:1005"))
		if provider.calls.Load() != loginMaxFailures+1 {
			t.Fatal("expired block did not permit a new provider check")
		}
	})

	t.Run("concurrent failures cannot exceed the provider attempt limit", func(t *testing.T) {
		fixture := createLoginFixture(t, admin, ctx, "athlete", "active", "active", false)
		defer fixture.remove(t, admin, ctx)
		provider := &loginProviderStub{err: errLoginDenied}
		handler := newLoginHandler(pool, []byte("12345678901234567890123456789012"), provider, now)
		var group sync.WaitGroup
		for attempt := 0; attempt < loginMaxFailures+1; attempt++ {
			group.Add(1)
			go func() {
				defer group.Done()
				assertInvalidCredentials(t, loginRequestForTest(handler, fixture.enrollment, "fixture-password", "127.0.0.1:1006"))
			}()
		}
		group.Wait()
		if provider.calls.Load() != loginMaxFailures {
			t.Fatalf("provider calls = %d, want %d", provider.calls.Load(), loginMaxFailures)
		}
	})

	t.Run("RLS denies direct login tables outside the server login transaction", func(t *testing.T) {
		err := pool.WithTransaction(ctx, func(tx pgx.Tx) error {
			var visible int
			if err := tx.QueryRow(ctx, "select count(*) from app.login_enrollments").Scan(&visible); err != nil {
				return err
			}
			if visible != 0 {
				return errors.New("login enrollment rows leaked without login context")
			}
			if _, err := tx.Exec(ctx, "savepoint rate_limit_rls"); err != nil {
				return err
			}
			_, err := tx.Exec(ctx, "insert into app.security_rate_limits(key_fingerprint, window_started_at, failure_count) values (decode(repeat('00', 32), 'hex'), now(), 1)")
			if err == nil {
				return errors.New("rate limit insert bypassed RLS")
			}
			if _, err := tx.Exec(ctx, "rollback to savepoint rate_limit_rls"); err != nil {
				return err
			}
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
	})
}

type loginProviderStub struct {
	err   error
	calls atomic.Int32
}

func (s *loginProviderStub) Authenticate(_ context.Context, subject uuid.UUID, _ string) (ProviderSession, error) {
	s.calls.Add(1)
	if s.err != nil {
		return ProviderSession{}, s.err
	}
	return ProviderSession{
		SubjectID: subject, SessionID: uuid.New(), AAL: "aal1",
		AccessToken: uuid.NewString(), RefreshToken: uuid.NewString(), ExpiresIn: 3600,
	}, nil
}

type loginFixture struct {
	organizationID uuid.UUID
	profileID      uuid.UUID
	enrollment     string
}

func createLoginFixture(t *testing.T, pool *pgxpool.Pool, ctx context.Context, role, organizationStatus, membershipStatus string, suspended bool) loginFixture {
	t.Helper()
	fixture := loginFixture{organizationID: uuid.New(), profileID: uuid.New(), enrollment: uuid.NewString()[0:10]}
	// UUID text may contain letters; generate an exact ten-digit fictional enrollment.
	fixture.enrollment = strings.NewReplacer("a", "0", "b", "1", "c", "2", "d", "3", "e", "4", "f", "5", "-", "").Replace(uuid.NewString())[:10]
	authID := uuid.New()
	transaction, err := pool.Begin(ctx)
	if err != nil {
		t.Fatalf("could not begin fictional login fixture: %v", err)
	}
	defer func() { _ = transaction.Rollback(ctx) }()
	if _, err := transaction.Exec(ctx, "set local session_replication_role = replica"); err != nil {
		t.Fatalf("could not prepare fictional login fixture: %v", err)
	}
	for _, statement := range []struct {
		query string
		args  []any
	}{
		{"insert into app.organizations (id, name, timezone, status) values ($1, 'Fictional login organization', 'UTC', $2)", []any{fixture.organizationID, organizationStatus}},
		{"insert into app.profiles (id, auth_user_id, full_name, suspended_at) values ($1, $2, 'Fictional login profile', case when $3 then now() else null end)", []any{fixture.profileID, authID, suspended}},
		{"insert into app.organization_memberships (organization_id, profile_id, role, status) values ($1, $2, $3, $4)", []any{fixture.organizationID, fixture.profileID, role, membershipStatus}},
		{"insert into app.login_enrollments (enrollment_number, profile_id, organization_id) values ($1, $2, $3)", []any{fixture.enrollment, fixture.profileID, fixture.organizationID}},
	} {
		if _, err = transaction.Exec(ctx, statement.query, statement.args...); err != nil {
			break
		}
	}
	if err != nil {
		t.Fatalf("could not create fictional login fixture: %v", err)
	}
	if err := transaction.Commit(ctx); err != nil {
		t.Fatalf("could not commit fictional login fixture: %v", err)
	}
	return fixture
}

func (f loginFixture) remove(t *testing.T, pool *pgxpool.Pool, ctx context.Context) {
	t.Helper()
	transaction, err := pool.Begin(ctx)
	if err != nil {
		t.Fatalf("could not begin fictional login fixture cleanup: %v", err)
	}
	defer func() { _ = transaction.Rollback(ctx) }()
	if _, err := transaction.Exec(ctx, "set local session_replication_role = replica"); err != nil {
		t.Fatalf("could not prepare fictional login fixture cleanup: %v", err)
	}
	for _, query := range []struct {
		statement string
		id        uuid.UUID
	}{
		{"delete from app.auth_sessions where profile_id = $1", f.profileID},
		{"delete from app.security_audit_events where target_profile_id = $1", f.profileID},
		{"delete from app.login_enrollments where profile_id = $1", f.profileID},
		{"delete from app.organization_memberships where profile_id = $1", f.profileID},
		{"delete from app.profiles where id = $1", f.profileID},
		{"delete from app.organizations where id = $1", f.organizationID},
	} {
		if _, err := transaction.Exec(ctx, query.statement, query.id); err != nil {
			t.Fatalf("could not remove fictional login fixture: %v", err)
		}
	}
	if err := transaction.Commit(ctx); err != nil {
		t.Fatalf("could not commit fictional login fixture cleanup: %v", err)
	}
}

func loginRequestForTest(handler http.Handler, enrollment, password, remoteAddress string) *httptest.ResponseRecorder {
	body, _ := json.Marshal(loginRequest{EnrollmentNumber: enrollment, Password: password})
	request := httptest.NewRequest(http.MethodPost, "/v1/auth/login", bytes.NewReader(body))
	request.RemoteAddr = remoteAddress
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	return response
}

func assertInvalidCredentials(t *testing.T, response *httptest.ResponseRecorder) {
	t.Helper()
	if response.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401; body=%s", response.Code, response.Body.String())
	}
	const expected = "{\"error\":{\"code\":\"invalid_credentials\",\"message\":\"invalid credentials\",\"request_id\":\"\"}}\n"
	if response.Body.String() != expected {
		t.Fatalf("body = %q, want %q", response.Body.String(), expected)
	}
}
