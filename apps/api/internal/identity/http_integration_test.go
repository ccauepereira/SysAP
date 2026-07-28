package identity_test

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/ccauepereira/SysAP/apps/api/internal/identity"
	"github.com/ccauepereira/SysAP/apps/api/internal/platform/auth"
	"github.com/ccauepereira/SysAP/apps/api/internal/platform/database"
)

type stubTokenVerifier struct {
	token auth.VerifiedToken
	err   error
}

func (s *stubTokenVerifier) Verify(ctx context.Context, raw string) (auth.VerifiedToken, error) {
	return s.token, s.err
}

type stubSessionResolver struct {
	context auth.AuthenticatedContext
	err     error
}

func (s *stubSessionResolver) Resolve(ctx context.Context, token auth.VerifiedToken, reqID string) (auth.AuthenticatedContext, error) {
	return s.context, s.err
}

type fixture struct {
	organizationA string
	organizationB string
	profileA      string
	profileB      string
	subjectA      string
	subjectB      string
	sessionA      string
	sessionB      string
}

func createFixture(t *testing.T, pool *pgxpool.Pool, ctx context.Context) fixture {
	t.Helper()
	transaction, err := pool.Begin(ctx)
	if err != nil {
		t.Fatal("could not begin fixture")
	}
	defer func() { _ = transaction.Rollback(ctx) }()

	if _, err := transaction.Exec(ctx, "set local session_replication_role = replica"); err != nil {
		t.Fatal("could not prepare fictional identity fixture")
	}

	f := fixture{}
	if err := transaction.QueryRow(ctx, `
		select gen_random_uuid()::text, gen_random_uuid()::text,
		       gen_random_uuid()::text, gen_random_uuid()::text,
		       gen_random_uuid()::text, gen_random_uuid()::text,
		       gen_random_uuid()::text, gen_random_uuid()::text
	`).Scan(
		&f.organizationA,
		&f.organizationB,
		&f.profileA,
		&f.profileB,
		&f.subjectA,
		&f.subjectB,
		&f.sessionA,
		&f.sessionB,
	); err != nil {
		t.Fatal("could not create fixture IDs")
	}

	for _, organization := range []string{f.organizationA, f.organizationB} {
		if _, err := transaction.Exec(ctx,
			"insert into app.organizations (id, name, timezone, status) values ($1, $2, 'UTC', 'active')",
			organization, "Fictional organization",
		); err != nil {
			t.Fatal("could not create fictional organization")
		}
	}

	if _, err := transaction.Exec(ctx,
		"insert into app.profiles (id, auth_user_id, full_name) values ($1, $2, 'Profile A')",
		f.profileA, f.subjectA,
	); err != nil {
		t.Fatal("could not create fictional profile A")
	}
	if _, err := transaction.Exec(ctx,
		"insert into app.profiles (id, auth_user_id, full_name) values ($1, $2, 'Profile B')",
		f.profileB, f.subjectB,
	); err != nil {
		t.Fatal("could not create fictional profile B")
	}

	if _, err := transaction.Exec(ctx, `
		insert into app.organization_memberships (organization_id, profile_id, role, status)
		values ($1, $2, 'athlete', 'active')
	`, f.organizationA, f.profileA); err != nil {
		t.Fatal("could not create fictional membership 1")
	}
	if _, err := transaction.Exec(ctx, `
		insert into app.organization_memberships (organization_id, profile_id, role, status)
		values ($1, $2, 'athlete', 'active')
	`, f.organizationB, f.profileA); err != nil {
		t.Fatal("could not create fictional membership 2")
	}
	if _, err := transaction.Exec(ctx, `
		insert into app.organization_memberships (organization_id, profile_id, role, status)
		values ($1, $2, 'trainer', 'active')
	`, f.organizationA, f.profileB); err != nil {
		t.Fatal("could not create fictional membership 3")
	}

	for _, session := range []struct {
		id      string
		profile string
	}{
		{id: f.sessionA, profile: f.profileA},
		{id: f.sessionB, profile: f.profileB},
	} {
		if _, err := transaction.Exec(ctx,
			"insert into app.auth_sessions (session_id, profile_id, assurance_level) values ($1, $2, 'aal1')",
			session.id, session.profile,
		); err != nil {
			t.Fatal("could not create fictional session")
		}
	}
	if err := transaction.Commit(ctx); err != nil {
		t.Fatal("could not commit fixture")
	}

	return f
}

func (f fixture) remove(t *testing.T, pool *pgxpool.Pool, ctx context.Context) {
	t.Helper()
	for _, statement := range []struct {
		query string
		id    string
	}{
		{"delete from app.auth_sessions where session_id = $1", f.sessionA},
		{"delete from app.auth_sessions where session_id = $1", f.sessionB},
		{"delete from app.organization_memberships where profile_id = $1", f.profileA},
		{"delete from app.organization_memberships where profile_id = $1", f.profileB},
		{"delete from app.organizations where id = $1", f.organizationA},
		{"delete from app.organizations where id = $1", f.organizationB},
		{"delete from app.profiles where id = $1", f.profileA},
		{"delete from app.profiles where id = $1", f.profileB},
	} {
		if _, err := pool.Exec(ctx, statement.query, statement.id); err != nil {
			t.Fatalf("could not remove fictional fixture: %v", err)
		}
	}
}

func discardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(bytes.NewBuffer(nil), nil))
}

func TestGetMeIntegration(t *testing.T) {
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

	f := createFixture(t, adminPool, ctx)
	defer f.remove(t, adminPool, ctx)

	meHandler := identity.NewMeHandler(pool, discardLogger())

	profileAUUID, _ := uuid.Parse(f.profileA)
	subjectAUUID, _ := uuid.Parse(f.subjectA)
	sessionAUUID, _ := uuid.Parse(f.sessionA)

	t.Run("returns profile and all memberships when no tenant is requested", func(t *testing.T) {
		verifier := &stubTokenVerifier{
			token: auth.VerifiedToken{SubjectID: subjectAUUID, SessionID: sessionAUUID, AAL: auth.AAL1},
		}
		resolver := &stubSessionResolver{
			context: auth.AuthenticatedContext{
				SubjectID: subjectAUUID,
				ProfileID: profileAUUID,
				SessionID: sessionAUUID,
				AAL:       auth.AAL1,
				RequestID: "test-request",
			},
		}
		handler := auth.Middleware(verifier, resolver)(meHandler)

		req := httptest.NewRequest(http.MethodGet, "/v1/me", nil)
		req.Header.Set("Authorization", "Bearer fictional-token")
		res := httptest.NewRecorder()
		handler.ServeHTTP(res, req)

		if res.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d", res.Code, http.StatusOK)
		}

		var response struct {
			Profile struct {
				ID          string `json:"id"`
				DisplayName string `json:"display_name"`
			} `json:"profile"`
			Memberships []struct {
				OrganizationID string `json:"organization_id"`
				Role           string `json:"role"`
				Status         string `json:"status"`
			} `json:"memberships"`
		}
		if err := json.Unmarshal(res.Body.Bytes(), &response); err != nil {
			t.Fatalf("response is not valid JSON: %v", err)
		}

		if response.Profile.ID != f.profileA || response.Profile.DisplayName != "Profile A" {
			t.Errorf("got profile %+v", response.Profile)
		}
		if len(response.Memberships) != 2 {
			t.Errorf("got %d memberships, want 2", len(response.Memberships))
		}
	})

	t.Run("returns single membership when tenant is requested and valid", func(t *testing.T) {
		verifier := &stubTokenVerifier{
			token: auth.VerifiedToken{SubjectID: subjectAUUID, SessionID: sessionAUUID, AAL: auth.AAL1},
		}
		resolver := &stubSessionResolver{
			context: auth.AuthenticatedContext{
				SubjectID: subjectAUUID,
				ProfileID: profileAUUID,
				SessionID: sessionAUUID,
				AAL:       auth.AAL1,
				RequestID: "test-request",
			},
		}
		handler := auth.Middleware(verifier, resolver)(meHandler)

		req := httptest.NewRequest(http.MethodGet, "/v1/me", nil)
		req.Header.Set("Authorization", "Bearer fictional-token")
		req.Header.Set("X-Organization-ID", f.organizationB) // athlete role
		res := httptest.NewRecorder()
		handler.ServeHTTP(res, req)

		if res.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d (body: %s)", res.Code, http.StatusOK, res.Body.String())
		}

		var response struct {
			Profile struct {
				ID          string `json:"id"`
				DisplayName string `json:"display_name"`
			} `json:"profile"`
			Memberships []struct {
				OrganizationID string `json:"organization_id"`
				Role           string `json:"role"`
				Status         string `json:"status"`
			} `json:"memberships"`
		}
		if err := json.Unmarshal(res.Body.Bytes(), &response); err != nil {
			t.Fatalf("response is not valid JSON: %v", err)
		}

		if response.Profile.ID != f.profileA {
			t.Errorf("got profile ID %s, want %s", response.Profile.ID, f.profileA)
		}
		if len(response.Memberships) != 1 {
			t.Errorf("got %d memberships, want 1", len(response.Memberships))
		} else {
			if response.Memberships[0].OrganizationID != f.organizationB || response.Memberships[0].Role != "athlete" {
				t.Errorf("got membership %+v", response.Memberships[0])
			}
		}
	})

	t.Run("returns 403 when tenant is requested but user is not member", func(t *testing.T) {
		verifier := &stubTokenVerifier{
			token: auth.VerifiedToken{SubjectID: subjectAUUID, SessionID: sessionAUUID, AAL: auth.AAL1},
		}
		resolver := &stubSessionResolver{
			context: auth.AuthenticatedContext{
				SubjectID: subjectAUUID,
				ProfileID: profileAUUID,
				SessionID: sessionAUUID,
				AAL:       auth.AAL1,
				RequestID: "test-request",
			},
		}
		handler := auth.Middleware(verifier, resolver)(meHandler)

		req := httptest.NewRequest(http.MethodGet, "/v1/me", nil)
		req.Header.Set("Authorization", "Bearer fictional-token")
		randomOrg := uuid.New().String()
		req.Header.Set("X-Organization-ID", randomOrg)
		res := httptest.NewRecorder()
		handler.ServeHTTP(res, req)

		if res.Code != http.StatusForbidden {
			t.Fatalf("status = %d, want %d", res.Code, http.StatusForbidden)
		}
	})

	t.Run("returns 400 on invalid X-Organization-ID", func(t *testing.T) {
		verifier := &stubTokenVerifier{
			token: auth.VerifiedToken{SubjectID: subjectAUUID, SessionID: sessionAUUID, AAL: auth.AAL1},
		}
		resolver := &stubSessionResolver{
			context: auth.AuthenticatedContext{
				SubjectID: subjectAUUID,
				ProfileID: profileAUUID,
				SessionID: sessionAUUID,
				AAL:       auth.AAL1,
				RequestID: "test-request",
			},
		}
		handler := auth.Middleware(verifier, resolver)(meHandler)

		req := httptest.NewRequest(http.MethodGet, "/v1/me", nil)
		req.Header.Set("Authorization", "Bearer fictional-token")
		req.Header.Set("X-Organization-ID", "not-a-uuid")
		res := httptest.NewRecorder()
		handler.ServeHTTP(res, req)

		if res.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want %d", res.Code, http.StatusBadRequest)
		}
	})

	t.Run("returns 401 when token is missing", func(t *testing.T) {
		verifier := &stubTokenVerifier{}
		resolver := &stubSessionResolver{}
		handler := auth.Middleware(verifier, resolver)(meHandler)

		req := httptest.NewRequest(http.MethodGet, "/v1/me", nil)
		res := httptest.NewRecorder()
		handler.ServeHTTP(res, req)

		if res.Code != http.StatusUnauthorized {
			t.Fatalf("status = %d, want %d", res.Code, http.StatusUnauthorized)
		}
	})

	t.Run("returns 401 when token is invalid", func(t *testing.T) {
		verifier := &stubTokenVerifier{err: auth.ErrInvalidToken}
		resolver := &stubSessionResolver{}
		handler := auth.Middleware(verifier, resolver)(meHandler)

		req := httptest.NewRequest(http.MethodGet, "/v1/me", nil)
		req.Header.Set("Authorization", "Bearer invalid-token")
		res := httptest.NewRecorder()
		handler.ServeHTTP(res, req)

		if res.Code != http.StatusUnauthorized {
			t.Fatalf("status = %d, want %d", res.Code, http.StatusUnauthorized)
		}
	})
}
