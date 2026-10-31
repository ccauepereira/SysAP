package identity

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/ccauepereira/SysAP/apps/api/internal/platform/auth"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type manualVerifier struct{}

func (manualVerifier) Verify(context.Context, string) (auth.VerifiedToken, error) {
	return auth.VerifiedToken{}, nil
}

type manualResolver struct{ actor auth.AuthenticatedContext }

func (s manualResolver) Resolve(context.Context, auth.VerifiedToken, string) (auth.AuthenticatedContext, error) {
	return s.actor, nil
}

func TestManualActivationRequiresOwnerMFAAndReplacesCode(t *testing.T) {
	pool, cleanup, enrollment, _, invitationID := setupActivationIntegrationTest(t)
	defer cleanup()
	ctx := context.Background()
	admin, err := pgxpool.New(ctx, os.Getenv("SYSAP_TEST_DATABASE_URL"))
	if err != nil {
		t.Fatal(err)
	}
	defer admin.Close()
	var organizationID, ownerID, subjectID uuid.UUID
	if err := admin.QueryRow(ctx, `select i.organization_id,i.invited_by_profile_id,p.auth_user_id from app.activation_invitations i join app.profiles p on p.id=i.invited_by_profile_id where i.id=$1`, invitationID).Scan(&organizationID, &ownerID, &subjectID); err != nil {
		t.Fatal(err)
	}
	sessionID := uuid.New()
	if _, err := admin.Exec(ctx, `insert into app.auth_sessions(session_id,profile_id,organization_id,assurance_level) values($1,$2,$3,'aal2')`, sessionID, ownerID, organizationID); err != nil {
		t.Fatal(err)
	}
	if _, err := admin.Exec(ctx, `insert into app.mfa_factors(profile_id,provider_factor_id,status,verified_at) values($1,$2,'verified',now())`, ownerID, uuid.New()); err != nil {
		t.Fatal(err)
	}
	defer func() { _, _ = admin.Exec(ctx, `delete from app.mfa_factors where profile_id=$1`, ownerID) }()
	newAuthID := uuid.New()
	if _, err := admin.Exec(ctx, `insert into auth.users(id,aud,role,email,raw_app_meta_data,raw_user_meta_data,created_at,updated_at) values($1,'authenticated','authenticated',$2,'{}'::jsonb,'{}'::jsonb,now(),now())`, newAuthID, "fake-"+newAuthID.String()+"@example.test"); err != nil {
		t.Fatal(err)
	}
	handler := newActivationHandler(pool, []byte("manual-test-pepper-long"), unavailableOTPProvider{}, &mockAuthAdmin{createdID: newAuthID}, slog.Default(), time.Now)
	actor := auth.AuthenticatedContext{SubjectID: subjectID, ProfileID: ownerID, SessionID: sessionID, AAL: auth.AAL2}
	issue := func(actor auth.AuthenticatedContext, org uuid.UUID) *httptest.ResponseRecorder {
		r := httptest.NewRequest(http.MethodPost, "/manual", nil)
		r.SetPathValue("organization_id", org.String())
		r.SetPathValue("invitation_id", invitationID.String())
		r.Header.Set("X-Organization-ID", org.String())
		r.Header.Set("Authorization", "Bearer fixture")
		w := httptest.NewRecorder()
		auth.Middleware(manualVerifier{}, manualResolver{actor})(handler).ServeHTTP(w, r)
		return w
	}
	low := actor
	low.AAL = auth.AAL1
	lowRes := issue(low, organizationID)
	if lowRes.Code != http.StatusForbidden {
		t.Fatalf("AAL1 = %d", lowRes.Code)
	}
	var lowErr struct {
		Error struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	if err := json.Unmarshal(lowRes.Body.Bytes(), &lowErr); err != nil || lowErr.Error.Code != "mfa_required" {
		t.Fatalf("expected mfa_required on AAL1, got %+v", lowErr)
	}
	otherRes := issue(actor, uuid.New())
	if otherRes.Code != http.StatusForbidden {
		t.Fatalf("other organization = %d", otherRes.Code)
	}
	var code struct {
		Enrollment string    `json:"enrollment_number"`
		Code       string    `json:"code"`
		ExpiresAt  time.Time `json:"expires_at"`
	}
	beforeIssue := time.Now().UTC()
	first := issue(actor, organizationID)
	if first.Code != http.StatusCreated {
		t.Fatalf("issue = %d", first.Code)
	}
	if err := json.Unmarshal(first.Body.Bytes(), &code); err != nil || code.Enrollment != enrollment || len(code.Code) != 6 {
		t.Fatal("invalid issued response")
	}
	ttl := code.ExpiresAt.Sub(beforeIssue)
	if ttl < 9*time.Minute+50*time.Second || ttl > 10*time.Minute+10*time.Second {
		t.Fatalf("expected ~10m expiration, got %v", ttl)
	}
	previous := code.Code
	second := issue(actor, organizationID)
	if second.Code != http.StatusCreated {
		t.Fatalf("reissue = %d", second.Code)
	}
	if err := json.Unmarshal(second.Body.Bytes(), &code); err != nil {
		t.Fatal(err)
	}
	if previous == code.Code {
		t.Fatal("replacement code must differ from the previous code")
	}
	verify := func(value string) *httptest.ResponseRecorder {
		body, _ := json.Marshal(map[string]string{"enrollment_number": enrollment, "code": value})
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/v1/activation/manual/verify", bytes.NewReader(body)))
		return w
	}
	if got := verify(previous).Code; got != http.StatusUnauthorized {
		t.Fatalf("previous code status = %d", got)
	}
	verified := verify(code.Code)
	if verified.Code != http.StatusOK {
		t.Fatalf("current code status = %d", verified.Code)
	}
	if got := verify(code.Code).Code; got != http.StatusUnauthorized {
		t.Fatalf("reused code status = %d", got)
	}
	var proof struct {
		Value string `json:"activation_proof"`
	}
	if err := json.Unmarshal(verified.Body.Bytes(), &proof); err != nil || proof.Value == "" {
		t.Fatal("missing proof")
	}
	payload, _ := json.Marshal(map[string]string{"activation_proof": proof.Value, "password": "ValidPassword1234!!"})
	var wg sync.WaitGroup
	statuses := make(chan int, 2)
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			w := httptest.NewRecorder()
			handler.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/v1/activation/complete", bytes.NewReader(payload)))
			statuses <- w.Code
		}()
	}
	wg.Wait()
	close(statuses)
	var succeeded int
	for status := range statuses {
		if status == http.StatusNoContent {
			succeeded++
		} else if status != http.StatusUnauthorized {
			t.Fatalf("unexpected completion status = %d", status)
		}
	}
	if succeeded != 1 {
		t.Fatalf("successful completions = %d", succeeded)
	}
}
