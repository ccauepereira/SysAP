package identity_test

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/ccauepereira/SysAP/apps/api/internal/identity"
	"github.com/ccauepereira/SysAP/apps/api/internal/platform/auth"
	"github.com/google/uuid"
)

func TestManualInvitationDoesNotSendEmail(t *testing.T) {
	pool, cleanup, org, owner, trainer, _, subject, trainerSubject, _, group := setupInvitationIntegrationTest(t)
	defer cleanup()
	delivery := &recordingEnrollmentDelivery{}
	handler := identity.NewInvitationHandlerWithDelivery(pool, slog.New(slog.NewTextHandler(io.Discard, nil)), time.Now, &testEnrollmentGenerator{}, delivery)
	actor := auth.AuthenticatedContext{SubjectID: subject, ProfileID: owner, SessionID: uuid.New(), AAL: auth.AAL2}
	mux := http.NewServeMux()
	mux.Handle("POST /v1/organizations/{organization_id}/athlete-invitations", auth.Middleware(&stubTokenVerifier{}, &stubSessionResolver{context: actor})(handler))
	mux.Handle("GET /v1/organizations/{organization_id}/athlete-invitations", auth.Middleware(&stubTokenVerifier{}, &stubSessionResolver{context: actor})(handler))
	data := map[string]string{"full_name": "Fictional Athlete", "phone_e164": "(85) 99999-0111", "email": "athlete@example.test", "birth_date": "2010-01-01", "locality": "Fictional", "team_id": group.String(), "football_position": "midfielder", "enrollment_delivery_channel": "manual"}
	body, err := json.Marshal(data)
	if err != nil {
		t.Fatal(err)
	}
	key := uuid.NewString()
	post := func() *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodPost, "/v1/organizations/"+org.String()+"/athlete-invitations", bytes.NewReader(body))
		req.Header.Set("Authorization", "Bearer fixture")
		req.Header.Set("X-Organization-ID", org.String())
		req.Header.Set("Idempotency-Key", key)
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)
		return rec
	}
	first := post()
	if first.Code != http.StatusCreated {
		t.Fatalf("create = %d", first.Code)
	}
	var result struct {
		Status           string    `json:"delivery_status"`
		Channel          string    `json:"delivery_channel"`
		ID               uuid.UUID `json:"invitation_id"`
		EnrollmentNumber string    `json:"enrollment_number"`
	}
	if err := json.Unmarshal(first.Body.Bytes(), &result); err != nil || result.Status != "not_applicable" || result.Channel != "manual" || len(result.EnrollmentNumber) != 10 {
		t.Fatalf("incorrect manual invitation status or enrollment_number: %+v", result)
	}
	second := post()
	if second.Code != http.StatusCreated {
		t.Fatalf("repeat = %d", second.Code)
	}
	var repeated struct {
		ID               uuid.UUID `json:"invitation_id"`
		EnrollmentNumber string    `json:"enrollment_number"`
	}
	if err := json.Unmarshal(second.Body.Bytes(), &repeated); err != nil || result.ID != repeated.ID || result.EnrollmentNumber != repeated.EnrollmentNumber {
		t.Fatal("repeat changed invitation or enrollment_number")
	}
	if delivery.calls != 0 {
		t.Fatal("manual invitation reached external delivery")
	}
	listRequest := httptest.NewRequest(http.MethodGet, "/v1/organizations/"+org.String()+"/athlete-invitations", nil)
	listRequest.Header.Set("Authorization", "Bearer fixture")
	listRequest.Header.Set("X-Organization-ID", org.String())
	list := httptest.NewRecorder()
	mux.ServeHTTP(list, listRequest)
	if list.Code != http.StatusOK {
		t.Fatalf("list status = %d", list.Code)
	}
	var listed struct {
		Items []struct {
			ID    uuid.UUID `json:"invitation_id"`
			Name  string    `json:"name"`
			Group string    `json:"group"`
		} `json:"items"`
	}
	if err := json.Unmarshal(list.Body.Bytes(), &listed); err != nil {
		t.Fatal(err)
	}
	found := false
	for _, item := range listed.Items {
		if item.ID == result.ID && item.Name == "Fictional Athlete" && item.Group != "" {
			found = true
		}
	}
	if !found {
		t.Fatal("manual athlete and group missing from owner listing")
	}
	trainerMux := http.NewServeMux()
	trainerMux.Handle("GET /v1/organizations/{organization_id}/athlete-invitations", auth.Middleware(&stubTokenVerifier{}, &stubSessionResolver{context: auth.AuthenticatedContext{SubjectID: trainerSubject, ProfileID: trainer, SessionID: uuid.New(), AAL: auth.AAL2}})(handler))
	trainerReq := httptest.NewRequest(http.MethodGet, "/v1/organizations/"+org.String()+"/athlete-invitations", nil)
	trainerReq.Header.Set("Authorization", "Bearer fixture")
	trainerReq.Header.Set("X-Organization-ID", org.String())
	trainerRec := httptest.NewRecorder()
	trainerMux.ServeHTTP(trainerRec, trainerReq)
	if trainerRec.Code != http.StatusForbidden {
		t.Fatalf("trainer list status = %d", trainerRec.Code)
	}
	other := uuid.NewString()
	otherReq := httptest.NewRequest(http.MethodGet, "/v1/organizations/"+other+"/athlete-invitations", nil)
	otherReq.Header.Set("Authorization", "Bearer fixture")
	otherReq.Header.Set("X-Organization-ID", other)
	otherRec := httptest.NewRecorder()
	mux.ServeHTTP(otherRec, otherReq)
	if otherRec.Code != http.StatusForbidden {
		t.Fatalf("other tenant list status = %d", otherRec.Code)
	}
}
