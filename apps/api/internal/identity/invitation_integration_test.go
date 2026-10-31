package identity_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
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
	"github.com/ccauepereira/SysAP/apps/api/internal/platform/authorization"
	"github.com/ccauepereira/SysAP/apps/api/internal/platform/database"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
)

// A stub enrollment generator to ensure uniqueness and predictability in tests.
type testEnrollmentGenerator struct {
	counter int
}

type recordingEnrollmentDelivery struct {
	calls      int
	enrollment string
}

func (p *recordingEnrollmentDelivery) SendEnrollment(_ context.Context, _, _, enrollment string) error {
	p.calls++
	p.enrollment = enrollment
	return nil
}

type failedEnrollmentDelivery struct{}

func (failedEnrollmentDelivery) SendEnrollment(context.Context, string, string, string) error {
	return knownDeliveryFailure{}
}

type knownDeliveryFailure struct{}

func (knownDeliveryFailure) Error() string           { return "safe fake delivery rejection" }
func (knownDeliveryFailure) DeliveryOutcome() string { return "failed" }

type countingFailedEnrollmentDelivery struct{ calls int }

func (p *countingFailedEnrollmentDelivery) SendEnrollment(context.Context, string, string, string) error {
	p.calls++
	return knownDeliveryFailure{}
}

func TestInvitationDeliveryFailureKeepsPendingAccount(t *testing.T) {
	pool, teardown, orgID, profileOwner, _, _, ownerAuthID, _, _, trainingGroupID := setupInvitationIntegrationTest(t)
	defer teardown()
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	handler := identity.NewInvitationHandlerWithDelivery(pool, logger, time.Now, &testEnrollmentGenerator{}, failedEnrollmentDelivery{})
	authContext := auth.AuthenticatedContext{SubjectID: ownerAuthID, ProfileID: profileOwner, SessionID: uuid.New(), AAL: auth.AAL2}
	middleware := auth.Middleware(&stubTokenVerifier{token: auth.VerifiedToken{}}, &stubSessionResolver{context: authContext})
	requestBody, _ := json.Marshal(map[string]string{
		"full_name": "Fictional Pending Athlete", "phone_e164": "+15555550123", "email": "pending@example.test",
		"birth_date": "2010-01-15", "locality": "Fictional City", "team_id": trainingGroupID.String(),
		"football_position": "midfielder", "enrollment_delivery_channel": "email",
	})
	req := httptest.NewRequest(http.MethodPost, "/v1/organizations/"+orgID.String()+"/athlete-invitations", bytes.NewReader(requestBody))
	req.SetPathValue("organization_id", orgID.String())
	req.Header.Set("X-Organization-ID", orgID.String())
	req.Header.Set("Idempotency-Key", uuid.NewString())
	req.Header.Set("Authorization", "Bearer fake")
	recorder := httptest.NewRecorder()
	middleware(handler).ServeHTTP(recorder, req)
	if recorder.Code != http.StatusCreated {
		t.Fatalf("invitation create returned unexpected status %d", recorder.Code)
	}
	var response struct {
		InvitationID   uuid.UUID `json:"invitation_id"`
		ProfileID      uuid.UUID `json:"profile_id"`
		Status         string    `json:"status"`
		DeliveryStatus string    `json:"delivery_status"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatal("invitation response was not valid JSON")
	}
	if response.Status != "pending_activation" || response.DeliveryStatus != "failed" {
		t.Fatalf("expected pending account with failed delivery state; got status=%q delivery=%q", response.Status, response.DeliveryStatus)
	}
	dbIdentity := database.AuthenticatedContext{SubjectID: pgtype.UUID{Bytes: ownerAuthID, Valid: true}, SessionID: pgtype.UUID{Bytes: authContext.SessionID, Valid: true}}
	var membershipStatus, invitationStatus, deliveryStatus, athleteStatus string
	err := authorization.WithMembershipContext(context.Background(), pool, dbIdentity, orgID.String(), profileOwner.String(), func(tx pgx.Tx, _ authorization.Membership) error {
		return tx.QueryRow(context.Background(), `
			select membership.status, invitation.status, invitation.delivery_status, athlete.status
			from app.activation_invitations invitation
			join app.organization_memberships membership on membership.organization_id=invitation.organization_id and membership.profile_id=invitation.profile_id
			join app.athlete_profiles athlete on athlete.id=invitation.profile_id
			where invitation.id=$1 and invitation.organization_id=$2
		`, response.InvitationID, orgID).Scan(&membershipStatus, &invitationStatus, &deliveryStatus, &athleteStatus)
	})
	if err != nil {
		t.Fatal("could not verify persisted pending invitation")
	}
	if membershipStatus != "pending_activation" || invitationStatus != "pending" || deliveryStatus != "failed" || athleteStatus != "pending_activation" {
		t.Fatal("delivery failure did not preserve a pending invitation and membership")
	}
}

func TestInvitationMembershipRLSRejectsUninvitedActiveInsert(t *testing.T) {
	_, teardown, orgID, _, _, _, _, _, _, _ := setupInvitationIntegrationTest(t)
	defer teardown()
	dbURL := os.Getenv("SYSAP_TEST_DATABASE_URL")
	adminPool, err := pgxpool.New(context.Background(), dbURL)
	if err != nil {
		t.Fatal("could not open isolated integration database")
	}
	defer adminPool.Close()

	profileID := uuid.New()
	if _, err := adminPool.Exec(context.Background(), `insert into app.profiles (id, full_name) values ($1, 'Uninvited Fictional Profile')`, profileID); err != nil {
		t.Fatal("could not create isolated RLS fixture")
	}

	tx, err := adminPool.Begin(context.Background())
	if err != nil {
		t.Fatal("could not begin RLS assertion")
	}
	defer tx.Rollback(context.Background())
	if _, err := tx.Exec(context.Background(), `set local role sysap_api`); err != nil {
		t.Fatal("could not assume API role for RLS assertion")
	}
	if _, err := tx.Exec(context.Background(), `select set_config('app.current_organization_id', $1, true)`, orgID.String()); err != nil {
		t.Fatal("could not set tenant context for RLS assertion")
	}
	_, err = tx.Exec(context.Background(), `insert into app.organization_memberships (organization_id, profile_id, role, status) values ($1, $2, 'athlete', 'active')`, orgID, profileID)
	var postgresError *pgconn.PgError
	if !errors.As(err, &postgresError) || postgresError.Code != "42501" {
		t.Fatal("RLS did not reject an active membership without a valid invitation")
	}
}

func TestInvitationResendUsesFakeDeliveryAndRetryPolicy(t *testing.T) {
	pool, teardown, orgID, profileOwner, _, _, ownerAuthID, _, _, trainingGroupID := setupInvitationIntegrationTest(t)
	defer teardown()

	clock := time.Date(2026, 7, 28, 10, 0, 0, 0, time.UTC)
	delivery := &countingFailedEnrollmentDelivery{}
	handler := identity.NewInvitationHandlerWithDelivery(pool, slog.Default(), func() time.Time { return clock }, &testEnrollmentGenerator{}, delivery)
	authContext := auth.AuthenticatedContext{SubjectID: ownerAuthID, ProfileID: profileOwner, SessionID: uuid.New(), AAL: auth.AAL2}
	middleware := auth.Middleware(&stubTokenVerifier{token: auth.VerifiedToken{}}, &stubSessionResolver{context: authContext})

	body, _ := json.Marshal(map[string]string{
		"full_name": "Resend Fixture Athlete", "phone_e164": "+15555550124", "email": "resend@example.test",
		"birth_date": "2010-01-15", "locality": "Fictional City", "team_id": trainingGroupID.String(),
		"football_position": "midfielder", "enrollment_delivery_channel": "email",
	})
	create := httptest.NewRequest(http.MethodPost, "/v1/organizations/"+orgID.String()+"/athlete-invitations", bytes.NewReader(body))
	create.SetPathValue("organization_id", orgID.String())
	create.Header.Set("X-Organization-ID", orgID.String())
	create.Header.Set("Authorization", "Bearer fake")
	create.Header.Set("Idempotency-Key", uuid.NewString())
	created := httptest.NewRecorder()
	middleware(handler).ServeHTTP(created, create)
	if created.Code != http.StatusCreated {
		t.Fatalf("create status = %d, body = %s", created.Code, created.Body.String())
	}
	var invitation struct {
		InvitationID uuid.UUID `json:"invitation_id"`
	}
	if err := json.Unmarshal(created.Body.Bytes(), &invitation); err != nil || invitation.InvitationID == uuid.Nil {
		t.Fatalf("create response did not contain invitation_id: %s", created.Body.String())
	}
	if delivery.calls != 1 {
		t.Fatalf("initial fake delivery calls = %d, want 1", delivery.calls)
	}

	clock = clock.Add(61 * time.Second)
	resend := httptest.NewRequest(http.MethodPost, "/v1/athlete-invitations/"+invitation.InvitationID.String()+"/resend", nil)
	resend.SetPathValue("invitation_id", invitation.InvitationID.String())
	resend.Header.Set("X-Organization-ID", orgID.String())
	resend.Header.Set("Authorization", "Bearer fake")
	resend.Header.Set("Idempotency-Key", uuid.NewString())
	resent := httptest.NewRecorder()
	middleware(handler).ServeHTTP(resent, resend)
	if resent.Code != http.StatusAccepted {
		t.Fatalf("resend status = %d, body = %s", resent.Code, resent.Body.String())
	}
	if delivery.calls != 2 {
		t.Fatalf("resend fake delivery calls = %d, want 2", delivery.calls)
	}
	var deliveryStatus string
	adminPool, err := pgxpool.New(context.Background(), os.Getenv("SYSAP_TEST_DATABASE_URL"))
	if err != nil {
		t.Fatal(err)
	}
	defer adminPool.Close()
	if err := adminPool.QueryRow(context.Background(), "select delivery_status from app.activation_invitations where id=$1", invitation.InvitationID).Scan(&deliveryStatus); err != nil {
		t.Fatal(err)
	}
	if deliveryStatus != "failed" {
		t.Fatalf("delivery_status = %q, want failed", deliveryStatus)
	}

	// Replaying the same key is idempotent and must not call the provider again.
	replay := httptest.NewRequest(http.MethodPost, "/v1/athlete-invitations/"+invitation.InvitationID.String()+"/resend", nil)
	replay.SetPathValue("invitation_id", invitation.InvitationID.String())
	replay.Header.Set("X-Organization-ID", orgID.String())
	replay.Header.Set("Authorization", "Bearer fake")
	replay.Header.Set("Idempotency-Key", resend.Header.Get("Idempotency-Key"))
	replayed := httptest.NewRecorder()
	middleware(handler).ServeHTTP(replayed, replay)
	if replayed.Code != http.StatusAccepted || delivery.calls != 2 {
		t.Fatalf("idempotent replay status=%d calls=%d; want 202 and 2 calls", replayed.Code, delivery.calls)
	}

	// A new key inside the cooldown is rejected without a provider attempt.
	limited := httptest.NewRequest(http.MethodPost, "/v1/athlete-invitations/"+invitation.InvitationID.String()+"/resend", nil)
	limited.SetPathValue("invitation_id", invitation.InvitationID.String())
	limited.Header.Set("X-Organization-ID", orgID.String())
	limited.Header.Set("Authorization", "Bearer fake")
	limited.Header.Set("Idempotency-Key", uuid.NewString())
	limitedResponse := httptest.NewRecorder()
	middleware(handler).ServeHTTP(limitedResponse, limited)
	if limitedResponse.Code != http.StatusTooManyRequests || delivery.calls != 2 {
		t.Fatalf("cooldown status=%d calls=%d; want 429 and 2 calls", limitedResponse.Code, delivery.calls)
	}
}

func (g *testEnrollmentGenerator) Generate() (string, error) {
	g.counter++
	return "2026" + "00000" + string(rune('0'+g.counter)), nil
}

func setupInvitationIntegrationTest(t *testing.T) (*database.Pool, func(), uuid.UUID, uuid.UUID, uuid.UUID, uuid.UUID, uuid.UUID, uuid.UUID, uuid.UUID, uuid.UUID) {
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
	ownerFactorID := uuid.New()
	trainingGroupID := uuid.New()

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
		{"insert into app.mfa_factors (profile_id, provider_factor_id, status, verified_at) values ($1, $2, 'verified', now())", []any{profileOwner, ownerFactorID}},
		{"insert into app.training_groups (id, organization_id, name) values ($1, $2, 'Fictional Group')", []any{trainingGroupID, orgID}},
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
		if tx, err := adminPool.Begin(ctx); err == nil {
			_, _ = tx.Exec(ctx, "set local session_replication_role = replica")
			_, _ = tx.Exec(ctx, "delete from app.training_group_athletes where organization_id = $1", orgID)
			_, _ = tx.Exec(ctx, "delete from app.idempotency_records where organization_id = $1", orgID)
			_, _ = tx.Exec(ctx, "delete from app.security_audit_events where organization_id = $1", orgID)
			_, _ = tx.Exec(ctx, "delete from app.mfa_factors where profile_id = any($1::uuid[])", []uuid.UUID{profileOwner, profileTrainer, profileAthlete})
			_, _ = tx.Exec(ctx, "delete from app.profiles where id in (select profile_id from app.activation_invitations where organization_id = $1)", orgID)
			_, _ = tx.Exec(ctx, "delete from app.activation_invitations where organization_id = $1", orgID)
			_, _ = tx.Exec(ctx, "delete from app.athlete_profiles where organization_id = $1", orgID)
			_, _ = tx.Exec(ctx, "delete from app.training_groups where organization_id = $1", orgID)
			_, _ = tx.Exec(ctx, "delete from app.organization_memberships where organization_id = $1", orgID)
			_, _ = tx.Exec(ctx, "delete from app.profiles where id = any($1::uuid[])", []uuid.UUID{profileOwner, profileTrainer, profileAthlete})
			_, _ = tx.Exec(ctx, "delete from app.organizations where id = $1", orgID)
			_, _ = tx.Exec(ctx, "delete from auth.users where id = any($1::uuid[])", []uuid.UUID{ownerAuthID, trainerAuthID, athleteAuthID})
			_ = tx.Commit(ctx)
		}
		adminPool.Close()
		pool.Close()
	}

	return pool, teardown, orgID, profileOwner, profileTrainer, profileAthlete, ownerAuthID, trainerAuthID, athleteAuthID, trainingGroupID
}

func TestCreateAthleteInvitation(t *testing.T) {
	pool, teardown, orgID, profileOwner, profileTrainer, profileAthlete, ownerAuthID, trainerAuthID, athleteAuthID, trainingGroupID := setupInvitationIntegrationTest(t)
	defer teardown()

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))

	clock := func() time.Time {
		return time.Date(2026, 7, 28, 10, 0, 0, 0, time.UTC)
	}

	enrollGen := &testEnrollmentGenerator{}
	delivery := &recordingEnrollmentDelivery{}
	handler := identity.NewInvitationHandlerWithDelivery(pool, logger, clock, enrollGen, delivery)

	t.Run("owner can create invitation successfully", func(t *testing.T) {
		reqBody := map[string]string{
			"full_name":                   "New Athlete",
			"phone_e164":                  "+5511999999999",
			"email":                       "test@example.com",
			"birth_date":                  "2010-01-15",
			"locality":                    "Fictional City",
			"team_id":                     trainingGroupID.String(),
			"football_position":           "midfielder",
			"enrollment_delivery_channel": "email",
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
			AAL:       auth.AAL2,
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
		if delivery.calls != 1 {
			t.Fatalf("expected one provider attempt, got %d", delivery.calls)
		}

		var resp map[string]interface{}
		json.Unmarshal(rr.Body.Bytes(), &resp)

		if resp["status"] != "pending_activation" {
			t.Errorf("expected status pending_activation, got %v", resp["status"])
		}
		if _, leaked := resp["enrollment_number"]; leaked {
			t.Errorf("enrollment_number must not be returned")
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
		if delivery.calls != 1 {
			t.Fatalf("idempotent replay repeated provider delivery; calls=%d", delivery.calls)
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
			"full_name":                   "New Athlete 2",
			"phone_e164":                  "+5511999999998",
			"email":                       "two@example.test",
			"birth_date":                  "2010-01-15",
			"locality":                    "Fictional City",
			"team_id":                     trainingGroupID.String(),
			"football_position":           "defender",
			"enrollment_delivery_channel": "email",
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
			"full_name":                   "New Athlete 3",
			"phone_e164":                  "+5511999999997",
			"email":                       "three@example.test",
			"birth_date":                  "2010-01-15",
			"locality":                    "Fictional City",
			"team_id":                     trainingGroupID.String(),
			"football_position":           "forward",
			"enrollment_delivery_channel": "email",
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
			"full_name":                   "New Athlete 4",
			"phone_e164":                  "+5511999999996",
			"email":                       "four@example.test",
			"birth_date":                  "2010-01-15",
			"locality":                    "Fictional City",
			"team_id":                     trainingGroupID.String(),
			"football_position":           "goalkeeper",
			"enrollment_delivery_channel": "email",
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
			"full_name":                   "New Athlete 5",
			"phone_e164":                  "+5511999999995",
			"email":                       "five@example.test",
			"birth_date":                  "2010-01-15",
			"locality":                    "Fictional City",
			"team_id":                     trainingGroupID.String(),
			"football_position":           "midfielder",
			"enrollment_delivery_channel": "email",
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

	t.Run("recovers from enrollment collision without aborting transaction", func(t *testing.T) {
		// Reset counter to 0 so the first generated enrollment ("2026000001") collides with the first test's athlete_profiles row!
		enrollGen.counter = 0
		reqBody := map[string]string{
			"full_name":                   "Collision Recovery Athlete",
			"phone_e164":                  "+5511999999996",
			"email":                       "collision@example.test",
			"birth_date":                  "2010-01-15",
			"locality":                    "Fictional City",
			"team_id":                     trainingGroupID.String(),
			"football_position":           "midfielder",
			"enrollment_delivery_channel": "email",
		}
		b, _ := json.Marshal(reqBody)

		req := httptest.NewRequest(http.MethodPost, "/v1/organizations/"+orgID.String()+"/athlete-invitations", bytes.NewReader(b))
		req.SetPathValue("organization_id", orgID.String())
		req.Header.Set("X-Organization-ID", orgID.String())
		req.Header.Set("Idempotency-Key", uuid.New().String())
		req.Header.Set("Authorization", "Bearer stub")

		authCtx := auth.AuthenticatedContext{
			SubjectID: ownerAuthID,
			ProfileID: profileOwner,
			SessionID: uuid.New(),
			AAL:       auth.AAL2,
		}
		middleware := auth.Middleware(&stubTokenVerifier{token: auth.VerifiedToken{}}, &stubSessionResolver{context: authCtx})
		rr := httptest.NewRecorder()
		middleware(handler).ServeHTTP(rr, req)

		if rr.Code != http.StatusCreated {
			t.Fatalf("expected 201 Created after enrollment collision recovery, got %d: %s", rr.Code, rr.Body.String())
		}
		if delivery.enrollment != "2026000002" {
			t.Fatalf("expected recovered enrollment 2026000002, got %s", delivery.enrollment)
		}
	})

	t.Run("lists active training groups for owner and denies athlete", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/v1/organizations/"+orgID.String()+"/training-groups", nil)
		req.SetPathValue("organization_id", orgID.String())
		req.Header.Set("X-Organization-ID", orgID.String())
		req.Header.Set("Authorization", "Bearer stub")

		ownerCtx := auth.AuthenticatedContext{
			SubjectID: ownerAuthID,
			ProfileID: profileOwner,
			SessionID: uuid.New(),
			AAL:       auth.AAL1,
		}
		ownerMiddleware := auth.Middleware(&stubTokenVerifier{token: auth.VerifiedToken{}}, &stubSessionResolver{context: ownerCtx})
		rr := httptest.NewRecorder()
		ownerMiddleware(handler).ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Fatalf("expected 200 OK for owner listing training groups, got %d: %s", rr.Code, rr.Body.String())
		}
		var listResp struct {
			Items []struct {
				ID   uuid.UUID `json:"id"`
				Name string    `json:"name"`
			} `json:"items"`
		}
		if err := json.Unmarshal(rr.Body.Bytes(), &listResp); err != nil || len(listResp.Items) != 1 || listResp.Items[0].ID != trainingGroupID {
			t.Fatalf("unexpected training groups response: %s", rr.Body.String())
		}

		athleteReq := httptest.NewRequest(http.MethodGet, "/v1/organizations/"+orgID.String()+"/training-groups", nil)
		athleteReq.SetPathValue("organization_id", orgID.String())
		athleteReq.Header.Set("X-Organization-ID", orgID.String())
		athleteReq.Header.Set("Authorization", "Bearer stub")
		athleteCtx := auth.AuthenticatedContext{
			SubjectID: athleteAuthID,
			ProfileID: profileAthlete,
			SessionID: uuid.New(),
			AAL:       auth.AAL1,
		}
		athleteMiddleware := auth.Middleware(&stubTokenVerifier{token: auth.VerifiedToken{}}, &stubSessionResolver{context: athleteCtx})
		athleteRR := httptest.NewRecorder()
		athleteMiddleware(handler).ServeHTTP(athleteRR, athleteReq)
		if athleteRR.Code != http.StatusForbidden {
			t.Fatalf("expected 403 Forbidden for athlete listing training groups, got %d", athleteRR.Code)
		}
	})
}
