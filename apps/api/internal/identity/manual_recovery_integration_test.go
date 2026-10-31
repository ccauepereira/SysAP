package identity

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/ccauepereira/SysAP/apps/api/internal/platform/auth"
	"github.com/ccauepereira/SysAP/apps/api/internal/platform/database"
)

func TestManualPasswordRecoveryWorkflow(t *testing.T) {
	databaseURL := os.Getenv("SYSAP_TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("SYSAP_TEST_DATABASE_URL is not set")
	}
	ctx := context.Background()
	admin, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	defer admin.Close()
	pool, err := database.NewPool(ctx, databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()

	// 1. Fixture: an active athlete
	athleteFixture := createLoginFixture(t, admin, ctx, "athlete", "active", "active", false)
	defer func() {
		_, _ = admin.Exec(ctx, "delete from app.password_recovery_challenges where profile_id = $1", athleteFixture.profileID)
		_, _ = admin.Exec(ctx, "delete from app.athlete_profiles where id = $1", athleteFixture.profileID)
		athleteFixture.remove(t, admin, ctx)
	}()
	// Ensure athlete_profiles entry exists with status='activated'
	if _, err := admin.Exec(ctx, `
		insert into app.athlete_profiles (id, organization_id, enrollment_number, display_name, phone_e164, email, status)
		values ($1, $2, $3, 'Test Athlete Recovery', '+5585999990001', 'recovery-athlete@example.test', 'activated')
		on conflict (id) do update set status = 'activated'`,
		athleteFixture.profileID, athleteFixture.organizationID, athleteFixture.enrollment); err != nil {
		t.Fatal(err)
	}

	// 2. Fixture: an active owner with MFA
	ownerFixture := createLoginFixture(t, admin, ctx, "owner", "active", "active", false)
	defer func() {
		_, _ = admin.Exec(ctx, "delete from app.mfa_factors where profile_id = $1", ownerFixture.profileID)
		ownerFixture.remove(t, admin, ctx)
	}()
	// Ensure owner is in the same organization
	if _, err := admin.Exec(ctx, `
		update app.organization_memberships set organization_id = $1 where profile_id = $2`,
		athleteFixture.organizationID, ownerFixture.profileID); err != nil {
		t.Fatal(err)
	}
	var ownerSubjectID uuid.UUID
	if err := admin.QueryRow(ctx, "select auth_user_id from app.profiles where id = $1", ownerFixture.profileID).Scan(&ownerSubjectID); err != nil {
		t.Fatal(err)
	}
	ownerSessionID := uuid.New()
	if _, err := admin.Exec(ctx, "insert into app.auth_sessions (session_id, profile_id, organization_id, assurance_level) values ($1, $2, $3, 'aal2')",
		ownerSessionID, ownerFixture.profileID, athleteFixture.organizationID); err != nil {
		t.Fatal(err)
	}
	if _, err := admin.Exec(ctx, "insert into app.mfa_factors (profile_id, provider_factor_id, status, verified_at) values ($1, $2, 'verified', now())",
		ownerFixture.profileID, uuid.New()); err != nil {
		t.Fatal(err)
	}

	// Active session for the athlete to verify revocation after recovery
	athleteSessionID := uuid.New()
	if _, err := admin.Exec(ctx, "insert into app.auth_sessions (session_id, profile_id, organization_id, assurance_level) values ($1, $2, $3, 'aal1')",
		athleteSessionID, athleteFixture.profileID, athleteFixture.organizationID); err != nil {
		t.Fatal(err)
	}

	pepper := []byte("0123456789abcdef0123456789abcdef")
	recoveryProvider := &recoveryProviderStub{}
	handler := newPasswordRecoveryHandler(pool, recoveryProvider, pepper, time.Now)

	ownerAAL2 := auth.AuthenticatedContext{
		SubjectID: ownerSubjectID,
		ProfileID: ownerFixture.profileID,
		SessionID: ownerSessionID,
		AAL:       auth.AAL2,
	}
	ownerAAL1 := ownerAAL2
	ownerAAL1.AAL = auth.AAL1

	issueRequest := func(actor auth.AuthenticatedContext, orgID, profileID string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodPost, "/manual-recovery", nil)
		req.SetPathValue("organization_id", orgID)
		req.SetPathValue("athlete_profile_id", profileID)
		req.Header.Set("X-Organization-ID", orgID)
		req.Header.Set("Authorization", "Bearer token")
		w := httptest.NewRecorder()
		auth.Middleware(manualVerifier{}, manualResolver{actor})(handler).ServeHTTP(w, req)
		return w
	}

	// AAL1 owner rejected
	if got := issueRequest(ownerAAL1, athleteFixture.organizationID.String(), athleteFixture.profileID.String()).Code; got != http.StatusForbidden {
		t.Fatalf("AAL1 owner should be rejected, got %d", got)
	}

	// Other organization rejected
	otherOrgID := uuid.New().String()
	if got := issueRequest(ownerAAL2, otherOrgID, athleteFixture.profileID.String()).Code; got != http.StatusForbidden {
		t.Fatalf("mismatched organization should be rejected, got %d", got)
	}

	// Unknown athlete 404
	if got := issueRequest(ownerAAL2, athleteFixture.organizationID.String(), uuid.NewString()).Code; got != http.StatusNotFound {
		t.Fatalf("nonexistent athlete should return 404, got %d", got)
	}

	// Valid issuance
	res1 := issueRequest(ownerAAL2, athleteFixture.organizationID.String(), athleteFixture.profileID.String())
	if res1.Code != http.StatusCreated {
		t.Fatalf("expected 201 Created on manual recovery issue, got %d: %s", res1.Code, res1.Body.String())
	}
	var issued1 struct {
		Enrollment string    `json:"enrollment_number"`
		Code       string    `json:"code"`
		ExpiresAt  time.Time `json:"expires_at"`
	}
	if err := json.Unmarshal(res1.Body.Bytes(), &issued1); err != nil || len(issued1.Code) != 6 || issued1.Enrollment != athleteFixture.enrollment {
		t.Fatalf("invalid issued response: %+v", issued1)
	}

	// Replacement issuance invalidates the first code
	res2 := issueRequest(ownerAAL2, athleteFixture.organizationID.String(), athleteFixture.profileID.String())
	if res2.Code != http.StatusCreated {
		t.Fatalf("expected 201 on replacement, got %d", res2.Code)
	}
	var issued2 struct {
		Enrollment string    `json:"enrollment_number"`
		Code       string    `json:"code"`
		ExpiresAt  time.Time `json:"expires_at"`
	}
	if err := json.Unmarshal(res2.Body.Bytes(), &issued2); err != nil || len(issued2.Code) != 6 {
		t.Fatalf("invalid second issued response: %+v", issued2)
	}
	if issued1.Code == issued2.Code {
		t.Fatal("replacement code must be distinct")
	}

	verifyManual := func(enrollment, code string) *httptest.ResponseRecorder {
		body, _ := json.Marshal(map[string]string{"enrollment_number": enrollment, "code": code})
		req := httptest.NewRequest(http.MethodPost, "/v1/auth/password-recovery/manual/verify", bytes.NewReader(body))
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
		return w
	}

	// Old code is rejected
	if got := verifyManual(issued1.Enrollment, issued1.Code).Code; got != http.StatusUnauthorized {
		t.Fatalf("old replaced code should be rejected with 401, got %d", got)
	}

	// Wrong enrollment rejected
	if got := verifyManual("0000000000", issued2.Code).Code; got != http.StatusUnauthorized {
		t.Fatalf("wrong enrollment should be rejected with 401, got %d", got)
	}

	// Current code succeeds
	verRes := verifyManual(issued2.Enrollment, issued2.Code)
	if verRes.Code != http.StatusOK {
		t.Fatalf("expected 200 OK on manual verify, got %d: %s", verRes.Code, verRes.Body.String())
	}
	var proofData struct {
		Proof string `json:"proof"`
	}
	if err := json.Unmarshal(verRes.Body.Bytes(), &proofData); err != nil || proofData.Proof == "" {
		t.Fatal("missing proof in manual verification")
	}

	// Reusing the same verified code fails
	if got := verifyManual(issued2.Enrollment, issued2.Code).Code; got != http.StatusUnauthorized {
		t.Fatalf("reused code should be rejected with 401, got %d", got)
	}

	// Complete password recovery with proof
	newPassword := "NewSyntheticPass1234!!"
	completeReq := func(proof, password string) *httptest.ResponseRecorder {
		body, _ := json.Marshal(map[string]string{"proof": proof, "password": password})
		req := httptest.NewRequest(http.MethodPost, "/v1/auth/password-recovery/complete", bytes.NewReader(body))
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
		return w
	}

	// Concurrent completion test
	start := make(chan struct{})
	statuses := make(chan int, 2)
	var wait sync.WaitGroup
	for range 2 {
		wait.Add(1)
		go func() {
			defer wait.Done()
			<-start
			statuses <- completeReq(proofData.Proof, newPassword).Code
		}()
	}
	close(start)
	wait.Wait()
	close(statuses)
	successes := 0
	for status := range statuses {
		if status == http.StatusNoContent {
			successes++
		}
	}
	if successes != 1 || recoveryProvider.passwordCalls != 1 {
		t.Fatalf("concurrent completion winners=%d provider_calls=%d, expected exactly 1", successes, recoveryProvider.passwordCalls)
	}

	// Sessions for the athlete must now be revoked
	var athleteRevoked bool
	if err := admin.QueryRow(ctx, "select revoked_at is not null from app.auth_sessions where session_id = $1", athleteSessionID).Scan(&athleteRevoked); err != nil || !athleteRevoked {
		t.Fatalf("athlete session was not revoked after manual password recovery: %v", err)
	}

	// Proof replay rejected
	if got := completeReq(proofData.Proof, newPassword).Code; got != http.StatusUnauthorized {
		t.Fatalf("reused proof should be rejected, got %d", got)
	}
}

func TestManualRecoveryCrossPurposesAndBruteForce(t *testing.T) {
	databaseURL := os.Getenv("SYSAP_TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("SYSAP_TEST_DATABASE_URL is not set")
	}
	ctx := context.Background()
	admin, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	defer admin.Close()
	pool, err := database.NewPool(ctx, databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()

	athleteFixture := createLoginFixture(t, admin, ctx, "athlete", "active", "active", false)
	defer func() {
		_, _ = admin.Exec(ctx, "delete from app.password_recovery_challenges where profile_id = $1", athleteFixture.profileID)
		_, _ = admin.Exec(ctx, "delete from app.athlete_profiles where id = $1", athleteFixture.profileID)
		athleteFixture.remove(t, admin, ctx)
	}()
	if _, err := admin.Exec(ctx, `
		insert into app.athlete_profiles (id, organization_id, enrollment_number, display_name, phone_e164, email, status)
		values ($1, $2, $3, 'Test Athlete Recovery Cross', '+5585999990002', 'cross-athlete@example.test', 'activated')
		on conflict (id) do update set status = 'activated'`,
		athleteFixture.profileID, athleteFixture.organizationID, athleteFixture.enrollment); err != nil {
		t.Fatal(err)
	}

	ownerFixture := createLoginFixture(t, admin, ctx, "owner", "active", "active", false)
	defer func() {
		_, _ = admin.Exec(ctx, "delete from app.mfa_factors where profile_id = $1", ownerFixture.profileID)
		ownerFixture.remove(t, admin, ctx)
	}()
	if _, err := admin.Exec(ctx, `update app.organization_memberships set organization_id = $1 where profile_id = $2`,
		athleteFixture.organizationID, ownerFixture.profileID); err != nil {
		t.Fatal(err)
	}
	var ownerSubjectID uuid.UUID
	if err := admin.QueryRow(ctx, "select auth_user_id from app.profiles where id = $1", ownerFixture.profileID).Scan(&ownerSubjectID); err != nil {
		t.Fatal(err)
	}
	ownerSessionID := uuid.New()
	if _, err := admin.Exec(ctx, "insert into app.auth_sessions (session_id, profile_id, organization_id, assurance_level) values ($1, $2, $3, 'aal2')",
		ownerSessionID, ownerFixture.profileID, athleteFixture.organizationID); err != nil {
		t.Fatal(err)
	}
	if _, err := admin.Exec(ctx, "insert into app.mfa_factors (profile_id, provider_factor_id, status, verified_at) values ($1, $2, 'verified', now())",
		ownerFixture.profileID, uuid.New()); err != nil {
		t.Fatal(err)
	}

	pepper := []byte("0123456789abcdef0123456789abcdef")
	recoveryProvider := &recoveryProviderStub{}
	recoveryHandler := newPasswordRecoveryHandler(pool, recoveryProvider, pepper, time.Now)
	activationHandler := newActivationHandler(pool, pepper, unavailableOTPProvider{}, &mockAuthAdmin{}, nil, time.Now)

	ownerAAL2 := auth.AuthenticatedContext{
		SubjectID: ownerSubjectID,
		ProfileID: ownerFixture.profileID,
		SessionID: ownerSessionID,
		AAL:       auth.AAL2,
	}

	// 1. Issue recovery code
	req := httptest.NewRequest(http.MethodPost, "/manual-recovery", nil)
	req.SetPathValue("organization_id", athleteFixture.organizationID.String())
	req.SetPathValue("athlete_profile_id", athleteFixture.profileID.String())
	req.Header.Set("X-Organization-ID", athleteFixture.organizationID.String())
	req.Header.Set("Authorization", "Bearer token")
	w := httptest.NewRecorder()
	auth.Middleware(manualVerifier{}, manualResolver{ownerAAL2})(recoveryHandler).ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("issue status = %d", w.Code)
	}
	var recCode struct {
		Enrollment string `json:"enrollment_number"`
		Code       string `json:"code"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &recCode)

	// 2. Recovery code cannot be used to verify activation
	actVerifyBody, _ := json.Marshal(map[string]string{"enrollment_number": recCode.Enrollment, "code": recCode.Code})
	actW := httptest.NewRecorder()
	activationHandler.ServeHTTP(actW, httptest.NewRequest(http.MethodPost, "/v1/activation/manual/verify", bytes.NewReader(actVerifyBody)))
	if actW.Code != http.StatusUnauthorized {
		t.Fatalf("activation must reject recovery code, got status %d", actW.Code)
	}

	// 3. Brute force protection on manual recovery: 5 wrong attempts lock the challenge
	for i := 0; i < 5; i++ {
		wrongBody, _ := json.Marshal(map[string]string{"enrollment_number": recCode.Enrollment, "code": "000000"})
		wWrong := httptest.NewRecorder()
		recoveryHandler.ServeHTTP(wWrong, httptest.NewRequest(http.MethodPost, "/v1/auth/password-recovery/manual/verify", bytes.NewReader(wrongBody)))
		if wWrong.Code != http.StatusUnauthorized {
			t.Fatalf("wrong attempt %d should be 401, got %d", i, wWrong.Code)
		}
	}
	// Even the correct code is now rejected because attempt_count >= 5
	correctBody, _ := json.Marshal(map[string]string{"enrollment_number": recCode.Enrollment, "code": recCode.Code})
	wLocked := httptest.NewRecorder()
	recoveryHandler.ServeHTTP(wLocked, httptest.NewRequest(http.MethodPost, "/v1/auth/password-recovery/manual/verify", bytes.NewReader(correctBody)))
	if wLocked.Code != http.StatusUnauthorized {
		t.Fatalf("challenge should be exhausted after 5 attempts, got %d", wLocked.Code)
	}
}
