package identity

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/ccauepereira/SysAP/apps/api/internal/platform/database"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type mockOTPProvider struct {
	code       string
	err        error
	isExternal bool
	startHook  func(ctx context.Context, phone string) error
}

func (m *mockOTPProvider) IsExternal() bool { return m.isExternal }
func (m *mockOTPProvider) Start(ctx context.Context, phone, code string) error {
	if m.startHook != nil {
		return m.startHook(ctx, phone)
	}
	return m.err
}
func (m *mockOTPProvider) Verify(ctx context.Context, phone, code string) error {
	if code != m.code {
		return errors.New("invalid_code")
	}
	return m.err
}
func (m *mockOTPProvider) NewCode() (string, error) { return m.code, m.err }

type mockAuthAdmin struct {
	createdID uuid.UUID
	err       error
	deleteErr error
	deleted   []uuid.UUID
	mu        sync.Mutex
}

func (m *mockAuthAdmin) CreateUser(ctx context.Context, phone, password string) (uuid.UUID, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.createdID, m.err
}
func (m *mockAuthAdmin) CreateUserWithEmail(ctx context.Context, email, phone, password string) (uuid.UUID, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.createdID, m.err
}
func (m *mockAuthAdmin) DeleteUser(ctx context.Context, id uuid.UUID) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.deleted = append(m.deleted, id)
	return m.deleteErr
}

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
		{"insert into app.athlete_profiles (id, organization_id, enrollment_number, display_name, phone_e164, email, status) values ($1, $2, $3, 'Test Athlete', '+15555550101', 'athlete@example.test', 'pending_activation')", []any{profileID, orgID, enrollment}},
		{"insert into app.activation_invitations (id, organization_id, profile_id, role, status, invited_by_profile_id, expires_at) values ($1, $2, $3, 'athlete', 'pending', $4, $5)", []any{invitationID, orgID, profileID, inviterProfileID, time.Now().Add(7 * 24 * time.Hour)}},
	}

	for _, q := range queries {
		if _, err := adminPool.Exec(ctx, q.query, q.args...); err != nil {
			t.Fatalf("failed to setup test fixture: %v (%s)", err, q.query)
		}
	}

	teardown := func() {
		adminPool.Exec(ctx, "delete from app.activation_challenges")
		adminPool.Exec(ctx, "delete from app.identity_repair_tasks")
		adminPool.Exec(ctx, "delete from app.auth_sessions")
		adminPool.Exec(ctx, "delete from app.security_audit_events")
		adminPool.Exec(ctx, "delete from app.login_enrollments")
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
		handler := newActivationHandler(pool, []byte("test-pepper"), &mockOTPProvider{}, &mockAuthAdmin{}, slog.Default(), func() time.Time {
			return time.Date(2025, 1, 1, 10, 0, 0, 0, time.UTC)
		})

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
		handler := newActivationHandler(pool, []byte(pepper), &mockOTPProvider{code: "123456"}, &mockAuthAdmin{}, slog.Default(), time.Now)

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

		now := time.Now()
		mockProvider := &mockOTPProvider{code: "654321"}
		handler := newActivationHandler(pool, []byte("test-pepper"), mockProvider, &mockAuthAdmin{}, slog.Default(), func() time.Time { return now })

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
		handler := newActivationHandler(pool, []byte(pepper), mockProvider, &mockAuthAdmin{}, slog.Default(), timeMock)

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

	t.Run("concurrent start leaves one active challenge", func(t *testing.T) {
		pool, teardown, enrollment, _, invitationID := setupActivationIntegrationTest(t)
		defer teardown()
		handler := newActivationHandler(pool, []byte("test-pepper"), &mockOTPProvider{code: "654321"}, &mockAuthAdmin{}, slog.Default(), time.Now)
		body, _ := json.Marshal(map[string]string{"enrollment_number": enrollment})
		ready := make(chan struct{})
		release := make(chan struct{})
		var wait sync.WaitGroup
		for range 2 {
			wait.Add(1)
			go func() {
				defer wait.Done()
				ready <- struct{}{}
				<-release
				rec := httptest.NewRecorder()
				handler.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/v1/activation/start", bytes.NewReader(body)))
				if rec.Code != http.StatusAccepted {
					t.Errorf("status = %d", rec.Code)
				}
			}()
		}
		<-ready
		<-ready
		close(release)
		wait.Wait()
		adminPool, err := pgxpool.New(context.Background(), os.Getenv("SYSAP_TEST_DATABASE_URL"))
		if err != nil {
			t.Fatal(err)
		}
		defer adminPool.Close()
		var active int
		if err := adminPool.QueryRow(context.Background(), `select count(*) from app.activation_challenges where invitation_id=$1 and invalidated_at is null and consumed_at is null`, invitationID).Scan(&active); err != nil || active != 1 {
			t.Fatal("concurrent start did not preserve one active challenge")
		}
	})

	t.Run("concurrent verify has one proof winner", func(t *testing.T) {
		pool, teardown, enrollment, _, _ := setupActivationIntegrationTest(t)
		defer teardown()
		handler := newActivationHandler(pool, []byte("test-pepper"), &mockOTPProvider{code: "654321"}, &mockAuthAdmin{}, slog.Default(), time.Now)
		startBody, _ := json.Marshal(map[string]string{"enrollment_number": enrollment})
		handler.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodPost, "/v1/activation/start", bytes.NewReader(startBody)))
		verifyBody, _ := json.Marshal(map[string]string{"enrollment_number": enrollment, "code": "654321"})
		ready, release := make(chan struct{}), make(chan struct{})
		results := make(chan int, 2)
		for range 2 {
			go func() {
				ready <- struct{}{}
				<-release
				rec := httptest.NewRecorder()
				handler.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/v1/activation/verify", bytes.NewReader(verifyBody)))
				results <- rec.Code
			}()
		}
		<-ready
		<-ready
		close(release)
		first, second := <-results, <-results
		if !((first == http.StatusOK && second == http.StatusUnauthorized) || (second == http.StatusOK && first == http.StatusUnauthorized)) {
			t.Fatalf("concurrent verify statuses = %d, %d", first, second)
		}
	})
}

func TestActivationRequiresSMSThenEmail(t *testing.T) {
	pool, teardown, enrollment, _, _ := setupActivationIntegrationTest(t)
	defer teardown()
	now := time.Now().UTC()
	handler := newActivationHandler(pool, []byte("dual-activation-pepper"), &mockOTPProvider{code: "654321"}, &mockAuthAdmin{}, slog.Default(), func() time.Time { return now })

	startBody, _ := json.Marshal(map[string]string{"enrollment_number": enrollment})
	start := httptest.NewRecorder()
	handler.ServeHTTP(start, httptest.NewRequest(http.MethodPost, "/v1/activation/start", bytes.NewReader(startBody)))
	if start.Code != http.StatusAccepted {
		t.Fatalf("expected SMS start acceptance, got %d", start.Code)
	}
	verifySMSBody, _ := json.Marshal(map[string]string{"enrollment_number": enrollment, "code": "654321"})
	verifySMS := httptest.NewRecorder()
	handler.ServeHTTP(verifySMS, httptest.NewRequest(http.MethodPost, "/v1/activation/verify-sms", bytes.NewReader(verifySMSBody)))
	if verifySMS.Code != http.StatusOK {
		t.Fatalf("expected SMS verification, got %d", verifySMS.Code)
	}
	var smsResult map[string]string
	_ = json.Unmarshal(verifySMS.Body.Bytes(), &smsResult)
	smsProof := smsResult["activation_proof"]
	if smsProof == "" {
		t.Fatal("expected SMS proof")
	}

	emailStartBody, _ := json.Marshal(map[string]string{"enrollment_number": enrollment, "sms_proof": smsProof})
	emailStart := httptest.NewRecorder()
	handler.ServeHTTP(emailStart, httptest.NewRequest(http.MethodPost, "/v1/activation/email/start", bytes.NewReader(emailStartBody)))
	if emailStart.Code != http.StatusAccepted {
		t.Fatalf("expected email start acceptance, got %d", emailStart.Code)
	}
	emailVerifyBody, _ := json.Marshal(map[string]string{"enrollment_number": enrollment, "code": "654321", "sms_proof": smsProof})
	emailVerify := httptest.NewRecorder()
	handler.ServeHTTP(emailVerify, httptest.NewRequest(http.MethodPost, "/v1/activation/email/verify", bytes.NewReader(emailVerifyBody)))
	if emailVerify.Code != http.StatusOK {
		t.Fatalf("expected email verification, got %d", emailVerify.Code)
	}
	var finalResult map[string]string
	_ = json.Unmarshal(emailVerify.Body.Bytes(), &finalResult)
	if finalResult["activation_proof"] == "" {
		t.Fatal("expected final activation proof")
	}
}

func TestActivationComplete(t *testing.T) {
	t.Run("invalid passwords", func(t *testing.T) {
		pool, teardown, _, _, _ := setupActivationIntegrationTest(t)
		defer teardown()
		handler := newActivationHandler(pool, []byte("test-pepper"), &mockOTPProvider{}, &mockAuthAdmin{}, slog.Default(), time.Now)

		passwords := []string{
			"short1!",                        // less than 15 chars
			"thisisverylongbutnodigits!",     // no digits
			"thisisverylong1234butnospecial", // no specials
			"1234567890123456",               // only digits
			"thisisavalidpassword1234!!",     // valid, but we test invalid here
		}
		for i, p := range passwords {
			if i == len(passwords)-1 {
				continue
			}
			reqBody := map[string]string{"activation_proof": "deadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeef", "password": p}
			b, _ := json.Marshal(reqBody)
			req := httptest.NewRequest(http.MethodPost, "/v1/activation/complete", bytes.NewReader(b))
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)
			if rec.Code != http.StatusUnprocessableEntity {
				t.Errorf("expected 422 for password %s, got %d", p, rec.Code)
			}
		}
	})

	t.Run("success consumes proof and calls auth admin", func(t *testing.T) {
		pool, teardown, _, profileID, invID := setupActivationIntegrationTest(t)
		defer teardown()

		createdAuthID := uuid.New()
		admin := &mockAuthAdmin{createdID: createdAuthID}
		handler := newActivationHandler(pool, []byte("test-pepper"), &mockOTPProvider{}, admin, slog.Default(), time.Now)

		adminPool, _ := pgxpool.New(context.Background(), os.Getenv("SYSAP_TEST_DATABASE_URL"))
		_, err := adminPool.Exec(context.Background(), "insert into auth.users (id, aud, role, email, raw_app_meta_data, raw_user_meta_data, created_at, updated_at) values ($1, 'authenticated', 'authenticated', $2, '{}'::jsonb, '{}'::jsonb, now(), now())", createdAuthID, "fake-"+createdAuthID.String()+"@example.test")
		adminPool.Close()
		if err != nil {
			t.Fatalf("failed to insert mock auth user: %v", err)
		}

		// Setup challenge and proof manually in DB to bypass start/verify for this test
		proofHex := "1a2b3c4d5e6f7a8b9c0d1e2f3a4b5c6d1a2b3c4d5e6f7a8b"
		mac := hmac.New(sha256.New, []byte("test-pepper"))
		mac.Write([]byte(proofHex))
		hashedProof := mac.Sum(nil)

		err = pool.WithTransaction(context.Background(), func(tx pgx.Tx) error {
			_, e := tx.Exec(context.Background(), "set local sysap.activation_flow = 'true'")
			if e != nil {
				return e
			}
			_, e = tx.Exec(context.Background(), `
				insert into app.activation_challenges(athlete_profile_id, invitation_id, otp_hmac, expires_at, activation_proof_hmac, proof_expires_at)
				values($1, $2, 'dummy', now() + interval '10 minute', $3, now() + interval '10 minute')`, profileID, invID, hashedProof)
			return e
		})
		if err != nil {
			t.Fatalf("failed to insert challenge: %v", err)
		}

		reqBody := map[string]string{"activation_proof": proofHex, "password": "ValidPassword1234!!"}
		b, _ := json.Marshal(reqBody)
		req := httptest.NewRequest(http.MethodPost, "/v1/activation/complete", bytes.NewReader(b))
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusNoContent {
			t.Errorf("expected 204 No Content, got %d", rec.Code)
		}

		// Verify proof cannot be reused
		req2 := httptest.NewRequest(http.MethodPost, "/v1/activation/complete", bytes.NewReader(b))
		rec2 := httptest.NewRecorder()
		handler.ServeHTTP(rec2, req2)
		if rec2.Code != http.StatusUnauthorized { // Because proof is not found -> invalid_proof -> fail()
			t.Errorf("expected 401 Unauthorized on reuse, got %d", rec2.Code)
		}
	})

	t.Run("compensates a local failure and records an unresolved compensation", func(t *testing.T) {
		pool, teardown, _, profileID, invID := setupActivationIntegrationTest(t)
		defer teardown()
		adminPool, err := pgxpool.New(context.Background(), os.Getenv("SYSAP_TEST_DATABASE_URL"))
		if err != nil {
			t.Fatal(err)
		}
		defer adminPool.Close()

		conflictingAuthID := uuid.New()
		if _, err := adminPool.Exec(context.Background(), `insert into auth.users (id, aud, role, email, raw_app_meta_data, raw_user_meta_data, created_at, updated_at) values ($1, 'authenticated', 'authenticated', $2, '{}'::jsonb, '{}'::jsonb, now(), now())`, conflictingAuthID, "fixture-"+conflictingAuthID.String()+"@example.test"); err != nil {
			t.Fatal(err)
		}
		if _, err := adminPool.Exec(context.Background(), `insert into app.profiles (id, auth_user_id, full_name) values ($1, $2, 'Conflict')`, profileID, conflictingAuthID); err != nil {
			t.Fatal(err)
		}

		proof := "0123456789abcdef0123456789abcdef0123456789abcdef"
		mac := hmac.New(sha256.New, []byte("test-pepper"))
		_, _ = mac.Write([]byte(proof))
		if _, err := adminPool.Exec(context.Background(), `insert into app.activation_challenges(athlete_profile_id, invitation_id, otp_hmac, expires_at, activation_proof_hmac, proof_expires_at) values($1, $2, 'fixture', now() + interval '10 minute', $3, now() + interval '10 minute')`, profileID, invID, mac.Sum(nil)); err != nil {
			t.Fatal(err)
		}

		admin := &mockAuthAdmin{createdID: uuid.New(), deleteErr: errors.New("fixture delete failure")}
		handler := newActivationHandler(pool, []byte("test-pepper"), &mockOTPProvider{}, admin, slog.Default(), time.Now)
		body, _ := json.Marshal(map[string]string{"activation_proof": proof, "password": "fixture-only-password1234!!"})
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/v1/activation/complete", bytes.NewReader(body)))
		if rec.Code != http.StatusConflict {
			t.Fatalf("status = %d, want %d", rec.Code, http.StatusConflict)
		}
		admin.mu.Lock()
		deleted := len(admin.deleted)
		admin.mu.Unlock()
		if deleted != 1 {
			t.Fatal("Auth compensation was not attempted")
		}
		var repairs int
		if err := adminPool.QueryRow(context.Background(), `select count(*) from app.identity_repair_tasks where auth_user_id=$1 and operation='delete_auth_user' and status='pending'`, admin.createdID).Scan(&repairs); err != nil || repairs != 1 {
			t.Fatal("unresolved compensation was not recorded")
		}
	})

	t.Run("concurrent completion has one winner", func(t *testing.T) {
		pool, teardown, _, profileID, invitationID := setupActivationIntegrationTest(t)
		defer teardown()
		adminPool, err := pgxpool.New(context.Background(), os.Getenv("SYSAP_TEST_DATABASE_URL"))
		if err != nil {
			t.Fatal(err)
		}
		defer adminPool.Close()
		authID := uuid.New()
		if _, err := adminPool.Exec(context.Background(), `insert into auth.users (id, aud, role, email, raw_app_meta_data, raw_user_meta_data, created_at, updated_at) values ($1, 'authenticated', 'authenticated', $2, '{}'::jsonb, '{}'::jsonb, now(), now())`, authID, "fixture-"+authID.String()+"@example.test"); err != nil {
			t.Fatal(err)
		}
		proof := "abcdef0123456789abcdef0123456789abcdef0123456789"
		mac := hmac.New(sha256.New, []byte("test-pepper"))
		_, _ = mac.Write([]byte(proof))
		if _, err := adminPool.Exec(context.Background(), `insert into app.activation_challenges(athlete_profile_id, invitation_id, otp_hmac, expires_at, activation_proof_hmac, proof_expires_at) values($1, $2, 'fixture', now() + interval '10 minute', $3, now() + interval '10 minute')`, profileID, invitationID, mac.Sum(nil)); err != nil {
			t.Fatal(err)
		}
		handler := newActivationHandler(pool, []byte("test-pepper"), &mockOTPProvider{}, &mockAuthAdmin{createdID: authID}, slog.Default(), time.Now)
		body, _ := json.Marshal(map[string]string{"activation_proof": proof, "password": "fixture-only-password1234!!"})
		ready, release, results := make(chan struct{}), make(chan struct{}), make(chan int, 2)
		for range 2 {
			go func() {
				ready <- struct{}{}
				<-release
				rec := httptest.NewRecorder()
				handler.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/v1/activation/complete", bytes.NewReader(body)))
				results <- rec.Code
			}()
		}
		<-ready
		<-ready
		close(release)
		first, second := <-results, <-results
		if !((first == http.StatusNoContent && second == http.StatusUnauthorized) || (second == http.StatusNoContent && first == http.StatusUnauthorized)) {
			t.Fatalf("concurrent completion statuses = %d, %d", first, second)
		}
		var memberships int
		if err := adminPool.QueryRow(context.Background(), `select count(*) from app.organization_memberships where profile_id=$1`, profileID).Scan(&memberships); err != nil || memberships != 1 {
			t.Fatal("completion duplicated membership")
		}
	})
}

func TestActivationChallengesRejectClientRoles(t *testing.T) {
	if os.Getenv("SYSAP_TEST_DATABASE_URL") == "" {
		t.Skip("SYSAP_TEST_DATABASE_URL is not set")
	}
	for _, role := range []string{"anon", "authenticated", "service_role"} {
		t.Run(role, func(t *testing.T) {
			pool, err := pgxpool.New(context.Background(), os.Getenv("SYSAP_TEST_DATABASE_URL"))
			if err != nil {
				t.Fatal(err)
			}
			defer pool.Close()
			if _, err := pool.Exec(context.Background(), "set role "+role); err != nil {
				t.Fatal(err)
			}
			var count int
			if err := pool.QueryRow(context.Background(), "select count(*) from app.activation_challenges").Scan(&count); err == nil {
				t.Fatal("client role can read activation challenges")
			}
		})
	}
}
