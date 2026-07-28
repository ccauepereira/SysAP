package identity

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/ccauepereira/SysAP/apps/api/internal/platform/database"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type mockOTPProvider struct {
	code string
	err  error
}
func (m *mockOTPProvider) NewCode() (string, error) { return m.code, m.err }

func setupActivationIntegrationTest(t *testing.T) (*database.Pool, func(), string, uuid.UUID, uuid.UUID) {
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

	adminPool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		t.Fatalf("failed to connect admin pool: %v", err)
	}

	orgID := uuid.New()
	profileID := uuid.New()
	inviterProfileID := uuid.New()
	ownerAuthID := uuid.New()
	invitationID := uuid.New()
	enrollment := "1234567890"

	// Reutiliza a fixture de usuários do 2D para o convidador
	queries := []struct {
		query string
		args  []any
	}{
		{"insert into app.organizations (id, name, timezone, status) values ($1, 'Test Org', 'UTC', 'active')", []any{orgID}},
		{"insert into auth.users (id, aud, role, email, raw_app_meta_data, raw_user_meta_data, created_at, updated_at) values ($1, 'authenticated', 'authenticated', $2, '{}'::jsonb, '{}'::jsonb, now(), now())", []any{ownerAuthID, "owner-" + ownerAuthID.String() + "@example.test"}},
		{"insert into app.profiles (id, auth_user_id, full_name) values ($1, $2, 'Owner')", []any{inviterProfileID, ownerAuthID}},
		{"insert into app.organization_memberships (id, organization_id, profile_id, role, status) values ($1, $2, $3, 'owner', 'active')", []any{uuid.New(), orgID, inviterProfileID}},
		{"insert into app.athlete_profiles (id, organization_id, enrollment_number, display_name, phone_e164, status) values ($1, $2, $3, 'Test Athlete', '+15555550101', 'pending_activation')", []any{profileID, orgID, enrollment}},
		{"insert into app.activation_invitations (id, organization_id, profile_id, role, status, invited_by_profile_id, expires_at) values ($1, $2, $3, 'athlete', 'pending', $4, $5)", []any{invitationID, orgID, profileID, inviterProfileID, time.Now().Add(7 * 24 * time.Hour)}},
	}

	for _, q := range queries {
		if _, err := adminPool.Exec(ctx, q.query, q.args...); err != nil {
			t.Fatalf("failed to setup test fixture: %v (%s)", err, q.query)
		}
	}

	teardown := func() {
		adminPool.Exec(ctx, "delete from app.activation_challenges")
		adminPool.Exec(ctx, "delete from app.activation_invitations")
		adminPool.Exec(ctx, "delete from app.athlete_profiles")
		adminPool.Exec(ctx, "delete from app.organization_memberships")
		adminPool.Exec(ctx, "delete from app.profiles")
		adminPool.Exec(ctx, "delete from app.organizations")
		adminPool.Exec(ctx, "delete from auth.users where id = $1", ownerAuthID)
		adminPool.Close()
		pool.Close()
	}

	return pool, teardown, enrollment, profileID, invitationID
}

func TestActivationStartAndVerify(t *testing.T) {
	pepper := "test_pepper_123"

	t.Run("invalid format", func(t *testing.T) {
		pool, teardown, _, _, _ := setupActivationIntegrationTest(t)
		defer teardown()
		handler := newActivationHandler(pool, []byte(pepper), &mockOTPProvider{code: "123456"}, time.Now)
		
		reqBody := map[string]string{"enrollment_number": "123"}
		b, _ := json.Marshal(reqBody)
		req := httptest.NewRequest(http.MethodPost, "/v1/activation/start", bytes.NewReader(b))
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)
		if rr.Code != http.StatusAccepted {
			t.Fatalf("expected 202 Accepted, got %d", rr.Code)
		}
		
		reqVerifyBody := map[string]string{"enrollment_number": "123", "code": "abc"}
		b2, _ := json.Marshal(reqVerifyBody)
		reqVerify := httptest.NewRequest(http.MethodPost, "/v1/activation/verify", bytes.NewReader(b2))
		rrVerify := httptest.NewRecorder()
		handler.ServeHTTP(rrVerify, reqVerify)
		if rrVerify.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401 Unauthorized for invalid verify, got %d", rrVerify.Code)
		}
	})

	t.Run("start returns 202 without enumerating", func(t *testing.T) {
		pool, teardown, _, _, _ := setupActivationIntegrationTest(t)
		defer teardown()
		handler := newActivationHandler(pool, []byte(pepper), &mockOTPProvider{code: "123456"}, time.Now)

		reqBody := map[string]string{"enrollment_number": "0000000000"}
		b, _ := json.Marshal(reqBody)
		req := httptest.NewRequest(http.MethodPost, "/v1/activation/start", bytes.NewReader(b))
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)

		if rr.Code != http.StatusAccepted {
			t.Fatalf("expected 202 Accepted, got %d", rr.Code)
		}
	})

	t.Run("start and verify success", func(t *testing.T) {
		pool, teardown, enrollment, _, _ := setupActivationIntegrationTest(t)
		defer teardown()
		
		mockProvider := &mockOTPProvider{code: "654321"}
		handler := newActivationHandler(pool, []byte(pepper), mockProvider, time.Now)

		reqBody := map[string]string{"enrollment_number": enrollment}
		b, _ := json.Marshal(reqBody)
		req := httptest.NewRequest(http.MethodPost, "/v1/activation/start", bytes.NewReader(b))
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)

		if rr.Code != http.StatusAccepted {
			t.Fatalf("expected 202 Accepted, got %d", rr.Code)
		}

		reqVerifyBody := map[string]string{"enrollment_number": enrollment, "code": "654321"}
		b2, _ := json.Marshal(reqVerifyBody)
		reqVerify := httptest.NewRequest(http.MethodPost, "/v1/activation/verify", bytes.NewReader(b2))
		rrVerify := httptest.NewRecorder()
		handler.ServeHTTP(rrVerify, reqVerify)

		if rrVerify.Code != http.StatusOK {
			t.Fatalf("expected 200 OK, got %d", rrVerify.Code)
		}

		var resp map[string]string
		if err := json.Unmarshal(rrVerify.Body.Bytes(), &resp); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}

		if resp["activation_proof"] == "" {
			t.Fatalf("expected activation_proof in response")
		}
	})

	t.Run("concurrency and resend limits", func(t *testing.T) {
		pool, teardown, enrollment, _, invitationID := setupActivationIntegrationTest(t)
		defer teardown()

		// 1. First request
		now := time.Now()
		timeMock := func() time.Time { return now }
		mockProvider := &mockOTPProvider{code: "111111"}
		handler := newActivationHandler(pool, []byte(pepper), mockProvider, timeMock)

		reqBody := map[string]string{"enrollment_number": enrollment}
		b, _ := json.Marshal(reqBody)
		
		rr1 := httptest.NewRecorder()
		handler.ServeHTTP(rr1, httptest.NewRequest(http.MethodPost, "/v1/activation/start", bytes.NewReader(b)))
		
		// 2. Second request immediately (should be blocked by 60s rule but return 202)
		rr2 := httptest.NewRecorder()
		handler.ServeHTTP(rr2, httptest.NewRequest(http.MethodPost, "/v1/activation/start", bytes.NewReader(b)))

		adminPool, _ := pgxpool.New(context.Background(), os.Getenv("SYSAP_TEST_DATABASE_URL"))
		defer adminPool.Close()

		var count int
		adminPool.QueryRow(context.Background(), "select count(*) from app.activation_challenges where invitation_id=$1", invitationID).Scan(&count)
		if count != 1 {
			t.Fatalf("expected 1 challenge, got %d (second was not throttled by 60s)", count)
		}
		
		var resendCount int
		adminPool.QueryRow(context.Background(), "select resend_count from app.activation_challenges where invitation_id=$1", invitationID).Scan(&resendCount)
		if resendCount != 0 {
			t.Fatalf("expected resend_count 0, got %d", resendCount)
		}

		// 3. Request after 61s
		now = now.Add(61 * time.Second)
		mockProvider.code = "222222"
		rr3 := httptest.NewRecorder()
		handler.ServeHTTP(rr3, httptest.NewRequest(http.MethodPost, "/v1/activation/start", bytes.NewReader(b)))

		adminPool.QueryRow(context.Background(), "select count(*) from app.activation_challenges where invitation_id=$1", invitationID).Scan(&count)
		if count != 2 {
			t.Fatalf("expected 2 challenges, got %d", count)
		}
		adminPool.QueryRow(context.Background(), "select resend_count from app.activation_challenges where invitation_id=$1 and invalidated_at is null", invitationID).Scan(&resendCount)
		if resendCount != 1 {
			t.Fatalf("expected resend_count 1, got %d", resendCount)
		}

		// 4. Request after 61s again
		now = now.Add(61 * time.Second)
		mockProvider.code = "333333"
		rr4 := httptest.NewRecorder()
		handler.ServeHTTP(rr4, httptest.NewRequest(http.MethodPost, "/v1/activation/start", bytes.NewReader(b)))
		adminPool.QueryRow(context.Background(), "select resend_count from app.activation_challenges where invitation_id=$1 and invalidated_at is null", invitationID).Scan(&resendCount)
		if resendCount != 2 {
			t.Fatalf("expected resend_count 2, got %d", resendCount)
		}

		// 5. Request after 61s again
		now = now.Add(61 * time.Second)
		mockProvider.code = "444444"
		rr5 := httptest.NewRecorder()
		handler.ServeHTTP(rr5, httptest.NewRequest(http.MethodPost, "/v1/activation/start", bytes.NewReader(b)))
		adminPool.QueryRow(context.Background(), "select resend_count from app.activation_challenges where invitation_id=$1 and invalidated_at is null", invitationID).Scan(&resendCount)
		if resendCount != 3 {
			t.Fatalf("expected resend_count 3, got %d", resendCount)
		}

		// 6. Request after 61s again (4th resend! Should be blocked by 24h limit)
		now = now.Add(61 * time.Second)
		mockProvider.code = "555555"
		rr6 := httptest.NewRecorder()
		handler.ServeHTTP(rr6, httptest.NewRequest(http.MethodPost, "/v1/activation/start", bytes.NewReader(b)))

		adminPool.QueryRow(context.Background(), "select count(*) from app.activation_challenges where invitation_id=$1", invitationID).Scan(&count)
		if count != 4 { // The 5th request should not have created a new challenge
			t.Fatalf("expected 4 challenges total after block, got %d", count)
		}

		// 7. Request after 24h+ (Should reset counter and create)
		now = now.Add(24 * time.Hour)
		mockProvider.code = "666666"
		rr7 := httptest.NewRecorder()
		handler.ServeHTTP(rr7, httptest.NewRequest(http.MethodPost, "/v1/activation/start", bytes.NewReader(b)))
		
		adminPool.QueryRow(context.Background(), "select count(*) from app.activation_challenges where invitation_id=$1", invitationID).Scan(&count)
		if count != 5 {
			t.Fatalf("expected 5 challenges after 24h reset, got %d", count)
		}
		adminPool.QueryRow(context.Background(), "select resend_count from app.activation_challenges where invitation_id=$1 and invalidated_at is null", invitationID).Scan(&resendCount)
		if resendCount != 0 {
			t.Fatalf("expected resend_count 0 after 24h reset, got %d", resendCount)
		}
	})
}
