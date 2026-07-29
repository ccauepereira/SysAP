package identity

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/ccauepereira/SysAP/apps/api/internal/platform/auth"
	"github.com/ccauepereira/SysAP/apps/api/internal/platform/database"
)

func TestMFAEnrollmentAndElevationIntegration(t *testing.T) {
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

	fixture := createLoginFixture(t, admin, ctx, "owner", "active", "active", false)
	defer func() {
		_, _ = admin.Exec(ctx, "delete from app.mfa_factors where profile_id = $1", fixture.profileID)
		fixture.remove(t, admin, ctx)
	}()
	var subjectID uuid.UUID
	if err := admin.QueryRow(ctx, "select auth_user_id from app.profiles where id = $1", fixture.profileID).Scan(&subjectID); err != nil {
		t.Fatal(err)
	}
	sessionID := uuid.New()
	if _, err := admin.Exec(ctx, "insert into app.auth_sessions (session_id, profile_id, organization_id, assurance_level) values ($1, $2, $3, 'aal1')", sessionID, fixture.profileID, fixture.organizationID); err != nil {
		t.Fatal(err)
	}

	provider := &mfaProviderStub{factorID: uuid.New(), sessionID: sessionID}
	handler := newMFAHandler(pool, provider)
	principal := auth.AuthenticatedContext{SubjectID: subjectID, ProfileID: fixture.profileID, SessionID: sessionID, AAL: auth.AAL1}
	secured := auth.Middleware(mfaVerifierStub{}, mfaResolverStub{principal: principal})(handler)

	enroll := httptest.NewRequest(http.MethodPost, "/v1/auth/mfa/enroll", nil)
	enroll.Header.Set("Authorization", "Bearer fixture-access-token")
	enrollResponse := httptest.NewRecorder()
	secured.ServeHTTP(enrollResponse, enroll)
	if enrollResponse.Code != http.StatusOK {
		t.Fatalf("enroll status=%d body=%s", enrollResponse.Code, enrollResponse.Body.String())
	}
	if provider.enrollCalls != 1 {
		t.Fatalf("enroll calls=%d", provider.enrollCalls)
	}

	verifyBody, _ := json.Marshal(mfaVerifyRequest{FactorID: provider.factorID, ChallengeID: "challenge", Code: "123456"})
	start := make(chan struct{})
	statuses := make(chan int, 2)
	var wait sync.WaitGroup
	for range 2 {
		wait.Add(1)
		go func() {
			defer wait.Done()
			<-start
			verify := httptest.NewRequest(http.MethodPost, "/v1/auth/mfa/verify", bytes.NewReader(verifyBody))
			verify.Header.Set("Authorization", "Bearer fixture-access-token")
			response := httptest.NewRecorder()
			secured.ServeHTTP(response, verify)
			statuses <- response.Code
		}()
	}
	close(start)
	wait.Wait()
	close(statuses)
	successes := 0
	for status := range statuses {
		if status == http.StatusOK {
			successes++
		}
	}
	if successes != 1 {
		t.Fatalf("concurrent MFA verify winners=%d, want one", successes)
	}
	var assurance string
	if err := admin.QueryRow(ctx, "select assurance_level from app.auth_sessions where session_id = $1", sessionID).Scan(&assurance); err != nil || assurance != "aal2" {
		t.Fatalf("assurance=%q err=%v", assurance, err)
	}

	if err := pool.WithAuthenticatedContext(ctx, database.AuthenticatedContext{SubjectID: pgtype.UUID{Bytes: subjectID, Valid: true}, SessionID: pgtype.UUID{Bytes: sessionID, Valid: true}}, func(tx pgx.Tx) error {
		if err := RequireAdministrativeAAL2(ctx, tx, principal, "owner"); err != errMFARequired {
			return err
		}
		return RequireAdministrativeAAL2(ctx, tx, auth.AuthenticatedContext{SubjectID: subjectID, ProfileID: fixture.profileID, SessionID: sessionID, AAL: auth.AAL2}, "owner")
	}); err != nil {
		t.Fatalf("AAL2 guard did not accept verified owner: %v", err)
	}
}

func TestPasswordRecoveryIntegration(t *testing.T) {
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
	fixture := createLoginFixture(t, admin, ctx, "athlete", "active", "active", false)
	defer func() {
		_, _ = admin.Exec(ctx, "delete from app.password_recovery_challenges where profile_id = $1", fixture.profileID)
		fixture.remove(t, admin, ctx)
	}()
	var subjectID uuid.UUID
	if err := admin.QueryRow(ctx, "select auth_user_id from app.profiles where id = $1", fixture.profileID).Scan(&subjectID); err != nil {
		t.Fatal(err)
	}
	sessionID := uuid.New()
	if _, err := admin.Exec(ctx, "insert into app.auth_sessions (session_id, profile_id, organization_id, assurance_level) values ($1, $2, $3, 'aal1')", sessionID, fixture.profileID, fixture.organizationID); err != nil {
		t.Fatal(err)
	}

	provider := &recoveryProviderStub{}
	handler := newPasswordRecoveryHandler(pool, provider, []byte("0123456789abcdef0123456789abcdef"), time.Now)
	known := recoveryRequest(handler, "/v1/auth/password-recovery/start", recoveryStartRequest{EnrollmentNumber: fixture.enrollment})
	unknown := recoveryRequest(handler, "/v1/auth/password-recovery/start", recoveryStartRequest{EnrollmentNumber: "9999999999"})
	if known.Code != http.StatusAccepted || unknown.Code != http.StatusAccepted || known.Body.String() != unknown.Body.String() {
		t.Fatalf("start leaked account existence: known=%d/%q unknown=%d/%q", known.Code, known.Body.String(), unknown.Code, unknown.Body.String())
	}
	var openChallenges int
	if err := admin.QueryRow(ctx, "select count(*) from app.password_recovery_challenges where profile_id = $1", fixture.profileID).Scan(&openChallenges); err != nil || openChallenges != 1 {
		t.Fatalf("recovery start did not create a usable challenge: count=%d calls=%d err=%v", openChallenges, provider.startCalls, err)
	}
	proofResponse := recoveryRequest(handler, "/v1/auth/password-recovery/verify", recoveryVerifyRequest{EnrollmentNumber: fixture.enrollment, Code: "123456"})
	if proofResponse.Code != http.StatusOK {
		t.Fatalf("verify status=%d body=%s", proofResponse.Code, proofResponse.Body.String())
	}
	var proof struct {
		Proof string `json:"proof"`
	}
	if json.Unmarshal(proofResponse.Body.Bytes(), &proof) != nil || proof.Proof == "" {
		t.Fatal("verification did not return an opaque proof")
	}
	start := make(chan struct{})
	statuses := make(chan int, 2)
	var wait sync.WaitGroup
	for range 2 {
		wait.Add(1)
		go func() {
			defer wait.Done()
			<-start
			statuses <- recoveryRequest(handler, "/v1/auth/password-recovery/complete", recoveryCompleteRequest{Proof: proof.Proof, Password: "fictional1234!!"}).Code
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
	if successes != 1 || provider.passwordCalls != 1 {
		t.Fatalf("concurrent completion winners=%d provider=%d", successes, provider.passwordCalls)
	}
	var revoked bool
	if err := admin.QueryRow(ctx, "select revoked_at is not null from app.auth_sessions where session_id = $1", sessionID).Scan(&revoked); err != nil || !revoked {
		t.Fatalf("local sessions were not revoked: %v", err)
	}
	second := recoveryRequest(handler, "/v1/auth/password-recovery/complete", recoveryCompleteRequest{Proof: proof.Proof, Password: "fictional1234!!"})
	if second.Code != http.StatusUnauthorized || provider.passwordCalls != 1 {
		t.Fatalf("proof replay was accepted: status=%d calls=%d", second.Code, provider.passwordCalls)
	}
}

type mfaProviderStub struct {
	factorID, sessionID uuid.UUID
	enrollCalls         int
}

func (s *mfaProviderStub) EnrollTOTP(context.Context, string) (MFAEnrollment, error) {
	s.enrollCalls++
	return MFAEnrollment{FactorID: s.factorID, ProvisioningURI: "otpauth://totp/fixture"}, nil
}
func (s *mfaProviderStub) CreateChallenge(context.Context, string, uuid.UUID) (string, error) {
	return "challenge", nil
}
func (s *mfaProviderStub) VerifyChallenge(_ context.Context, _ string, _ uuid.UUID, _, _ string, subject uuid.UUID) (ProviderSession, error) {
	payload, _ := json.Marshal(map[string]string{"sub": subject.String(), "session_id": s.sessionID.String(), "aal": "aal2"})
	token := "x." + base64.RawURLEncoding.EncodeToString(payload) + ".x"
	return ProviderSession{SubjectID: subject, SessionID: s.sessionID, AAL: "aal2", AccessToken: token, RefreshToken: "fixture-refresh", ExpiresIn: 3600}, nil
}
func (s *mfaProviderStub) FactorActive(context.Context, string, uuid.UUID) (bool, error) {
	return true, nil
}

type mfaVerifierStub struct{}

func (mfaVerifierStub) Verify(context.Context, string) (auth.VerifiedToken, error) {
	return auth.VerifiedToken{}, nil
}

type mfaResolverStub struct{ principal auth.AuthenticatedContext }

func (s mfaResolverStub) Resolve(context.Context, auth.VerifiedToken, string) (auth.AuthenticatedContext, error) {
	return s.principal, nil
}

type recoveryProviderStub struct{ startCalls, verifyCalls, passwordCalls int }

func (s *recoveryProviderStub) Start(context.Context, uuid.UUID) error { s.startCalls++; return nil }
func (s *recoveryProviderStub) Verify(context.Context, uuid.UUID, string) error {
	s.verifyCalls++
	return nil
}
func (s *recoveryProviderStub) SetPassword(context.Context, uuid.UUID, string) error {
	s.passwordCalls++
	return nil
}
func recoveryRequest(handler http.Handler, path string, payload any) *httptest.ResponseRecorder {
	body, _ := json.Marshal(payload)
	request := httptest.NewRequest(http.MethodPost, path, bytes.NewReader(body))
	request.RemoteAddr = "127.0.0.1:3333"
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	return response
}
