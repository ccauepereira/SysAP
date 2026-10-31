package identity

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"testing"
	"time"

	"github.com/ccauepereira/SysAP/apps/api/internal/platform/auth"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestManualActivationWithRealIsolatedAuth(t *testing.T) {
	if os.Getenv("SYSAP_TEST_REAL_ISOLATED_AUTH") != "1" {
		t.Skip("explicit isolated Auth test not enabled")
	}
	authURL, err := url.Parse(os.Getenv("SYSAP_SUPABASE_AUTH_URL"))
	if err != nil || authURL.Scheme != "http" || authURL.Hostname() != "127.0.0.1" || authURL.Port() != "55421" || authURL.Path != "/auth/v1" {
		t.Fatal("Auth is not the designated isolated instance")
	}
	dbURL, err := url.Parse(os.Getenv("SYSAP_TEST_DATABASE_URL"))
	if err != nil || dbURL.Hostname() != "127.0.0.1" || dbURL.Port() != "55422" {
		t.Fatal("database is not the designated isolated instance")
	}
	anonKey := os.Getenv("SYSAP_TEST_ANON_KEY")
	if anonKey == "" {
		t.Fatal("isolated anonymous key missing")
	}
	signupBody, _ := json.Marshal(map[string]string{"email": "signup-" + uuid.NewString() + "@example.test", "password": "SyntheticPassword1234!!"})
	signupRequest, err := http.NewRequestWithContext(context.Background(), http.MethodPost, authURL.String()+"/signup", bytes.NewReader(signupBody))
	if err != nil {
		t.Fatal("signup check unavailable")
	}
	signupRequest.Header.Set("Content-Type", "application/json")
	signupRequest.Header.Set("apikey", anonKey)
	signupRequest.Header.Set("Authorization", "Bearer "+anonKey)
	signupResponse, err := http.DefaultClient.Do(signupRequest)
	if err != nil {
		t.Fatal("signup check unavailable")
	}
	defer signupResponse.Body.Close()
	var signupError struct {
		Code string `json:"error_code"`
	}
	if err := json.NewDecoder(signupResponse.Body).Decode(&signupError); err != nil || signupError.Code != "signup_disabled" {
		t.Fatalf("public signup was not confirmed disabled: HTTP %d", signupResponse.StatusCode)
	}
	pool, cleanup, enrollment, profileID, invitationID := setupActivationIntegrationTest(t)
	defer cleanup()
	ctx := context.Background()
	admin, err := pgxpool.New(ctx, os.Getenv("SYSAP_TEST_DATABASE_URL"))
	if err != nil {
		t.Fatal("isolated database unavailable")
	}
	defer admin.Close()
	var org, owner, subject uuid.UUID
	if err := admin.QueryRow(ctx, `select i.organization_id,i.invited_by_profile_id,p.auth_user_id from app.activation_invitations i join app.profiles p on p.id=i.invited_by_profile_id where i.id=$1`, invitationID).Scan(&org, &owner, &subject); err != nil {
		t.Fatal("fixture unavailable")
	}
	testEmail := "fixture-" + uuid.NewString() + "@example.test"
	if _, err := admin.Exec(ctx, `update app.athlete_profiles set email=$1 where id=$2`, testEmail, profileID); err != nil {
		t.Fatal("fixture contact unavailable")
	}
	session := uuid.New()
	if _, err := admin.Exec(ctx, `insert into app.auth_sessions(session_id,profile_id,organization_id,assurance_level) values($1,$2,$3,'aal2')`, session, owner, org); err != nil {
		t.Fatal("fixture session unavailable")
	}
	if _, err := admin.Exec(ctx, `insert into app.mfa_factors(profile_id,provider_factor_id,status,verified_at) values($1,$2,'verified',now())`, owner, uuid.New()); err != nil {
		t.Fatal("fixture MFA unavailable")
	}
	defer func() { _, _ = admin.Exec(ctx, `delete from app.mfa_factors where profile_id=$1`, owner) }()
	handler := newActivationHandler(pool, []byte("manual-isolated-test-pepper"), unavailableOTPProvider{}, NewSupabaseAuthAdmin(), slog.Default(), time.Now)
	actor := auth.AuthenticatedContext{SubjectID: subject, ProfileID: owner, SessionID: session, AAL: auth.AAL2}
	req := httptest.NewRequest(http.MethodPost, "/manual", nil)
	req.SetPathValue("organization_id", org.String())
	req.SetPathValue("invitation_id", invitationID.String())
	req.Header.Set("X-Organization-ID", org.String())
	req.Header.Set("Authorization", "Bearer fixture")
	issued := httptest.NewRecorder()
	auth.Middleware(manualVerifier{}, manualResolver{actor})(handler).ServeHTTP(issued, req)
	if issued.Code != http.StatusCreated {
		t.Fatalf("issue status = %d", issued.Code)
	}
	var code struct {
		Code string `json:"code"`
	}
	if err := json.Unmarshal(issued.Body.Bytes(), &code); err != nil || len(code.Code) != 6 {
		t.Fatal("missing code")
	}
	verifyBody, _ := json.Marshal(map[string]string{"enrollment_number": enrollment, "code": code.Code})
	verified := httptest.NewRecorder()
	handler.ServeHTTP(verified, httptest.NewRequest(http.MethodPost, "/v1/activation/manual/verify", bytes.NewReader(verifyBody)))
	if verified.Code != http.StatusOK {
		t.Fatalf("verify status = %d", verified.Code)
	}
	var proof struct {
		Value string `json:"activation_proof"`
	}
	if err := json.Unmarshal(verified.Body.Bytes(), &proof); err != nil || proof.Value == "" {
		t.Fatal("missing proof")
	}
	password := "SyntheticPassword1234!!"
	completeBody, _ := json.Marshal(map[string]string{"activation_proof": proof.Value, "password": password})
	completed := httptest.NewRecorder()
	handler.ServeHTTP(completed, httptest.NewRequest(http.MethodPost, "/v1/activation/complete", bytes.NewReader(completeBody)))
	if completed.Code != http.StatusNoContent {
		t.Fatalf("complete status = %d", completed.Code)
	}
	var athleteSubject uuid.UUID
	if err := admin.QueryRow(ctx, `select auth_user_id from app.profiles where id=$1`, profileID).Scan(&athleteSubject); err != nil {
		t.Fatal("activated subject missing")
	}
	if _, err := NewSupabasePasswordIdentityProvider().Authenticate(ctx, athleteSubject, password); err != nil {
		login, _ := json.Marshal(map[string]string{"email": testEmail, "password": password})
		request, requestErr := http.NewRequestWithContext(ctx, http.MethodPost, authURL.String()+"/token?grant_type=password", bytes.NewReader(login))
		if requestErr != nil {
			t.Fatal("real isolated Auth login request failed")
		}
		request.Header.Set("apikey", os.Getenv("SYSAP_SUPABASE_SERVICE_ROLE_KEY"))
		request.Header.Set("Authorization", "Bearer "+os.Getenv("SYSAP_SUPABASE_SERVICE_ROLE_KEY"))
		request.Header.Set("Content-Type", "application/json")
		response, requestErr := http.DefaultClient.Do(request)
		if requestErr != nil {
			t.Fatal("real isolated Auth login transport failed")
		}
		defer response.Body.Close()
		var details struct {
			ErrorCode string `json:"error_code"`
			Code      string `json:"code"`
		}
		_ = json.NewDecoder(response.Body).Decode(&details)
		t.Fatalf("real isolated Auth login failed: HTTP %d, provider code %s %s", response.StatusCode, details.ErrorCode, details.Code)
	}
}
