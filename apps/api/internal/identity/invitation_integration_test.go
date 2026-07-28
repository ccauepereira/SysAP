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

// A stub enrollment generator to ensure uniqueness and predictability in tests.
type testEnrollmentGenerator struct {
	counter int
}

func (g *testEnrollmentGenerator) Generate() (string, error) {
	g.counter++
	return "2026" + "00000" + string(rune('0'+g.counter)), nil
}

func setupInvitationIntegrationTest(t *testing.T) (*database.Pool, func(), uuid.UUID, uuid.UUID, uuid.UUID, uuid.UUID, uuid.UUID, uuid.UUID, uuid.UUID) {
	t.Helper()

	dbURL := os.Getenv("SYSAP_TEST_DATABASE_URL")
	if dbURL == "" {
		t.Skip("SYSAP_TEST_DATABASE_URL is not set")
	}

	ctx := context.Background()
	pool, err := database.NewPool(ctx, dbURL)
	if err != nil {
		t.Fatalf("failed to connect to database: %v", err)
	}

	orgID := uuid.New()
	profileOwner := uuid.New()
	profileTrainer := uuid.New()
	profileAthlete := uuid.New()

	ownerAuthID := uuid.New()
	trainerAuthID := uuid.New()
	athleteAuthID := uuid.New()

	membershipOwner := uuid.New()
	membershipTrainer := uuid.New()
	membershipAthlete := uuid.New()

	queries := []struct {
		query string
		args  []any
	}{
		{"insert into auth.users (id, aud, role, email, raw_app_meta_data, raw_user_meta_data, created_at, updated_at) values ($1, 'authenticated', 'authenticated', $2, '{}'::jsonb, '{}'::jsonb, now(), now())", []any{ownerAuthID, "owner-" + ownerAuthID.String() + "@example.test"}},
		{"insert into auth.users (id, aud, role, email, raw_app_meta_data, raw_user_meta_data, created_at, updated_at) values ($1, 'authenticated', 'authenticated', $2, '{}'::jsonb, '{}'::jsonb, now(), now())", []any{trainerAuthID, "trainer-" + trainerAuthID.String() + "@example.test"}},
		{"insert into auth.users (id, aud, role, email, raw_app_meta_data, raw_user_meta_data, created_at, updated_at) values ($1, 'authenticated', 'authenticated', $2, '{}'::jsonb, '{}'::jsonb, now(), now())", []any{athleteAuthID, "athlete-" + athleteAuthID.String() + "@example.test"}},
		{"insert into app.organizations (id, name, timezone, status) values ($1, 'Test Org', 'UTC', 'active')", []any{orgID}},
		{"insert into app.profiles (id, auth_user_id, full_name) values ($1, $2, 'Owner')", []any{profileOwner, ownerAuthID}},
		{"insert into app.profiles (id, auth_user_id, full_name) values ($1, $2, 'Trainer')", []any{profileTrainer, trainerAuthID}},
		{"insert into app.profiles (id, auth_user_id, full_name) values ($1, $2, 'Athlete')", []any{profileAthlete, athleteAuthID}},
		{"insert into app.organization_memberships (id, organization_id, profile_id, role, status) values ($1, $2, $3, 'owner', 'active')", []any{membershipOwner, orgID, profileOwner}},
		{"insert into app.organization_memberships (id, organization_id, profile_id, role, status) values ($1, $2, $3, 'trainer', 'active')", []any{membershipTrainer, orgID, profileTrainer}},
		{"insert into app.organization_memberships (id, organization_id, profile_id, role, status) values ($1, $2, $3, 'athlete', 'active')", []any{membershipAthlete, orgID, profileAthlete}},
	}

	adminPool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		t.Fatalf("failed to connect admin pool: %v", err)
	}

	for _, q := range queries {
		if _, err := adminPool.Exec(ctx, q.query, q.args...); err != nil {
			t.Fatalf("failed to setup test fixture: %v (%s)", err, q.query)
		}
	}

	teardown := func() {
		adminPool.Exec(ctx, "delete from app.idempotency_records")
		adminPool.Exec(ctx, "delete from app.security_audit_events")
		adminPool.Exec(ctx, "delete from app.activation_invitations")
		adminPool.Exec(ctx, "delete from app.athlete_profiles")
		adminPool.Exec(ctx, "delete from app.organization_memberships")
		adminPool.Exec(ctx, "delete from app.profiles")
		adminPool.Exec(ctx, "delete from app.organizations")
		adminPool.Exec(ctx, "delete from auth.users where id = any($1::uuid[])", []uuid.UUID{ownerAuthID, trainerAuthID, athleteAuthID})
		adminPool.Close()
		pool.Close()
	}

	return pool, teardown, orgID, profileOwner, profileTrainer, profileAthlete, ownerAuthID, trainerAuthID, athleteAuthID
}

func TestCreateAthleteInvitation(t *testing.T) {
	pool, teardown, orgID, profileOwner, profileTrainer, profileAthlete, ownerAuthID, trainerAuthID, athleteAuthID := setupInvitationIntegrationTest(t)
	defer teardown()

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))

	clock := func() time.Time {
		return time.Date(2026, 7, 28, 10, 0, 0, 0, time.UTC)
	}

	enrollGen := &testEnrollmentGenerator{}
	handler := identity.NewInvitationHandler(pool, logger, clock, enrollGen)

	t.Run("owner can create invitation successfully", func(t *testing.T) {
		reqBody := map[string]string{
			"display_name": "New Athlete",
			"phone_e164":   "+5511999999999",
			"email":        "test@example.com",
		}
		b, _ := json.Marshal(reqBody)

		req := httptest.NewRequest(http.MethodPost, "/v1/organizations/"+orgID.String()+"/athlete-invitations", bytes.NewReader(b))
		req.SetPathValue("organization_id", orgID.String())
		req.Header.Set("X-Organization-ID", orgID.String())
		req.Header.Set("Authorization", "Bearer stub")
		idempotencyKey := uuid.New().String()
		req.Header.Set("Idempotency-Key", idempotencyKey)

		// We'll wrap the handler with a mock middleware manually for testing since ContextWithAuthenticatedContext is unexported.
		sessionID := uuid.New()
		authCtx := auth.AuthenticatedContext{
			SubjectID: ownerAuthID,
			ProfileID: profileOwner,
			SessionID: sessionID,
		}
		// Since we can't create the context, let's just make a stub resolver
		stubResolver := &stubSessionResolver{context: authCtx}
		stubVerifier := &stubTokenVerifier{token: auth.VerifiedToken{}}
		middleware := auth.Middleware(stubVerifier, stubResolver)
		testHandler := middleware(handler)

		mux := http.NewServeMux()
		mux.Handle("POST /v1/organizations/{organization_id}/athlete-invitations", testHandler)

		rr := httptest.NewRecorder()
		mux.ServeHTTP(rr, req)

		if rr.Code != http.StatusCreated {
			t.Fatalf("expected 201 Created, got %d: %s", rr.Code, rr.Body.String())
		}

		var resp map[string]interface{}
		json.Unmarshal(rr.Body.Bytes(), &resp)

		if resp["status"] != "pending_activation" {
			t.Errorf("expected status pending_activation, got %v", resp["status"])
		}
		if resp["enrollment_number"] == "" {
			t.Errorf("expected enrollment_number, got empty")
		}

		// Verify idempotency works
		rr2 := httptest.NewRecorder()
		req2 := httptest.NewRequest(http.MethodPost, "/v1/organizations/"+orgID.String()+"/athlete-invitations", bytes.NewReader(b))
		req2.Header.Set("Authorization", "Bearer stub")
		req2.Header.Set("X-Organization-ID", orgID.String())
		req2.Header.Set("Idempotency-Key", idempotencyKey)
		mux.ServeHTTP(rr2, req2)

		if rr2.Code != http.StatusCreated {
			t.Fatalf("expected 201 for idempotent request, got %d", rr2.Code)
		}

		// Verify different payload fails
		reqBody["email"] = "other@example.com"
		b2, _ := json.Marshal(reqBody)
		rr3 := httptest.NewRecorder()
		req3 := httptest.NewRequest(http.MethodPost, "/v1/organizations/"+orgID.String()+"/athlete-invitations", bytes.NewReader(b2))
		req3.Header.Set("Authorization", "Bearer stub")
		req3.Header.Set("X-Organization-ID", orgID.String())
		req3.Header.Set("Idempotency-Key", idempotencyKey)

		mux.ServeHTTP(rr3, req3)

		if rr3.Code != http.StatusConflict {
			t.Fatalf("expected 409 for conflict, got %d", rr3.Code)
		}
	})

	t.Run("trainer cannot create invitation", func(t *testing.T) {
		reqBody := map[string]string{
			"display_name": "New Athlete 2",
			"phone_e164":   "+5511999999998",
		}
		b, _ := json.Marshal(reqBody)

		req := httptest.NewRequest(http.MethodPost, "/v1/organizations/"+orgID.String()+"/athlete-invitations", bytes.NewReader(b))
		req.SetPathValue("organization_id", orgID.String())
		req.Header.Set("X-Organization-ID", orgID.String())
		req.Header.Set("Idempotency-Key", uuid.New().String())

		req.Header.Set("Authorization", "Bearer stub")

		authCtx := auth.AuthenticatedContext{
			SubjectID: trainerAuthID,
			ProfileID: profileTrainer,
			SessionID: uuid.New(),
		}
		stubResolver := &stubSessionResolver{context: authCtx}
		stubVerifier := &stubTokenVerifier{token: auth.VerifiedToken{}}
		middleware := auth.Middleware(stubVerifier, stubResolver)
		testHandler := middleware(handler)

		rr := httptest.NewRecorder()
		testHandler.ServeHTTP(rr, req)

		if rr.Code != http.StatusForbidden {
			t.Fatalf("expected 403 Forbidden, got %d: %s", rr.Code, rr.Body.String())
		}
	})

	t.Run("missing idempotency key fails securely", func(t *testing.T) {
		reqBody := map[string]string{
			"display_name": "New Athlete 3",
			"phone_e164":   "+5511999999997",
		}
		b, _ := json.Marshal(reqBody)

		req := httptest.NewRequest(http.MethodPost, "/v1/organizations/"+orgID.String()+"/athlete-invitations", bytes.NewReader(b))
		req.SetPathValue("organization_id", orgID.String())
		req.Header.Set("X-Organization-ID", orgID.String())

		req.Header.Set("Authorization", "Bearer stub")

		authCtx := auth.AuthenticatedContext{
			SubjectID: ownerAuthID,
			ProfileID: profileOwner,
			SessionID: uuid.New(),
		}
		stubResolver := &stubSessionResolver{context: authCtx}
		stubVerifier := &stubTokenVerifier{token: auth.VerifiedToken{}}
		middleware := auth.Middleware(stubVerifier, stubResolver)
		testHandler := middleware(handler)

		rr := httptest.NewRecorder()
		testHandler.ServeHTTP(rr, req)

		if rr.Code != http.StatusBadRequest {
			t.Fatalf("expected 400 Bad Request, got %d", rr.Code)
		}
	})

	t.Run("athlete cannot create invitation", func(t *testing.T) {
		reqBody := map[string]string{
			"display_name": "New Athlete 4",
			"phone_e164":   "+5511999999996",
		}
		b, _ := json.Marshal(reqBody)

		req := httptest.NewRequest(http.MethodPost, "/v1/organizations/"+orgID.String()+"/athlete-invitations", bytes.NewReader(b))
		req.SetPathValue("organization_id", orgID.String())
		req.Header.Set("X-Organization-ID", orgID.String())
		req.Header.Set("Idempotency-Key", uuid.New().String())

		req.Header.Set("Authorization", "Bearer stub")

		authCtx := auth.AuthenticatedContext{
			SubjectID: athleteAuthID,
			ProfileID: profileAthlete,
			SessionID: uuid.New(),
		}
		stubResolver := &stubSessionResolver{context: authCtx}
		stubVerifier := &stubTokenVerifier{token: auth.VerifiedToken{}}
		middleware := auth.Middleware(stubVerifier, stubResolver)
		testHandler := middleware(handler)

		rr := httptest.NewRecorder()
		testHandler.ServeHTTP(rr, req)

		if rr.Code != http.StatusForbidden {
			t.Fatalf("expected 403 Forbidden, got %d: %s", rr.Code, rr.Body.String())
		}
	})

	t.Run("cross organization fails", func(t *testing.T) {
		reqBody := map[string]string{
			"display_name": "New Athlete 5",
			"phone_e164":   "+5511999999995",
		}
		b, _ := json.Marshal(reqBody)

		otherOrgID := uuid.New()

		req := httptest.NewRequest(http.MethodPost, "/v1/organizations/"+otherOrgID.String()+"/athlete-invitations", bytes.NewReader(b))
		req.SetPathValue("organization_id", otherOrgID.String())
		req.Header.Set("X-Organization-ID", otherOrgID.String())
		req.Header.Set("Idempotency-Key", uuid.New().String())

		req.Header.Set("Authorization", "Bearer stub")

		authCtx := auth.AuthenticatedContext{
			SubjectID: ownerAuthID,
			ProfileID: profileOwner,
			SessionID: uuid.New(),
		}
		stubResolver := &stubSessionResolver{context: authCtx}
		stubVerifier := &stubTokenVerifier{token: auth.VerifiedToken{}}
		middleware := auth.Middleware(stubVerifier, stubResolver)
		testHandler := middleware(handler)

		rr := httptest.NewRecorder()
		testHandler.ServeHTTP(rr, req)

		if rr.Code != http.StatusForbidden {
			t.Fatalf("expected 403 Forbidden for cross organization access, got %d", rr.Code)
		}
	})
}
