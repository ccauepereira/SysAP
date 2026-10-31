package identity

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/ccauepereira/SysAP/apps/api/internal/platform/auth"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestAthleteSelfUsesAuthenticatedIdentityAndTenant(t *testing.T) {
	pool, cleanup, enrollment, profileID, invitationID := setupActivationIntegrationTest(t)
	defer cleanup()
	ctx := context.Background()
	admin, err := pgxpool.New(ctx, os.Getenv("SYSAP_TEST_DATABASE_URL"))
	if err != nil {
		t.Fatal(err)
	}
	defer admin.Close()
	var org, owner, ownerSubject uuid.UUID
	if err := admin.QueryRow(ctx, `select i.organization_id,i.invited_by_profile_id,p.auth_user_id from app.activation_invitations i join app.profiles p on p.id=i.invited_by_profile_id where i.id=$1`, invitationID).Scan(&org, &owner, &ownerSubject); err != nil {
		t.Fatal(err)
	}
	subject, session, group := uuid.New(), uuid.New(), uuid.New()
	queries := []struct {
		sql  string
		args []any
	}{
		{`insert into auth.users(id,aud,role,email,raw_app_meta_data,raw_user_meta_data,created_at,updated_at) values($1,'authenticated','authenticated',$2,'{}'::jsonb,'{}'::jsonb,now(),now())`, []any{subject, "fake-" + subject.String() + "@example.test"}},
		{`update app.profiles set auth_user_id=$1 where id=$2`, []any{subject, profileID}},
		{`update app.athlete_profiles set status='activated' where id=$1`, []any{profileID}},
		{`update app.organization_memberships set status='active',activated_at=now() where profile_id=$1`, []any{profileID}},
		{`insert into app.auth_sessions(session_id,profile_id,organization_id,assurance_level) values($1,$2,$3,'aal1')`, []any{session, profileID, org}},
		{`insert into app.training_groups(id,organization_id,name) values($1,$2,'Turma fictícia')`, []any{group, org}},
		{`insert into app.training_group_athletes(organization_id,training_group_id,athlete_profile_id) values($1,$2,$3)`, []any{org, group, profileID}},
	}
	for _, q := range queries {
		if _, err := admin.Exec(ctx, q.sql, q.args...); err != nil {
			t.Fatal(err)
		}
	}
	defer func() {
		_, _ = admin.Exec(ctx, `delete from app.training_group_athletes where training_group_id=$1`, group)
		_, _ = admin.Exec(ctx, `delete from app.training_groups where id=$1`, group)
	}()
	handler := NewMeHandler(pool, slog.Default())
	request := func(actor auth.AuthenticatedContext, organization uuid.UUID) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodGet, "/v1/me/athlete?profile_id="+owner.String(), nil)
		req.Header.Set("Authorization", "Bearer fixture")
		req.Header.Set("X-Organization-ID", organization.String())
		rec := httptest.NewRecorder()
		auth.Middleware(manualVerifier{}, manualResolver{actor})(handler).ServeHTTP(rec, req)
		return rec
	}
	actor := auth.AuthenticatedContext{SubjectID: subject, ProfileID: profileID, SessionID: session, AAL: auth.AAL1}
	response := request(actor, org)
	if response.Code != http.StatusOK {
		t.Fatalf("self status = %d", response.Code)
	}
	var body struct {
		Name       string   `json:"name"`
		Enrollment string   `json:"enrollment_number"`
		Groups     []string `json:"groups"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil || body.Name != "Test Athlete" || body.Enrollment != enrollment || len(body.Groups) != 1 || body.Groups[0] != "Turma fictícia" {
		t.Fatal("self response did not match the authenticated athlete")
	}
	if got := request(actor, uuid.New()).Code; got != http.StatusForbidden {
		t.Fatalf("other tenant status = %d", got)
	}
	ownerActor := auth.AuthenticatedContext{SubjectID: ownerSubject, ProfileID: owner, SessionID: uuid.New(), AAL: auth.AAL2}
	if got := request(ownerActor, org).Code; got != http.StatusForbidden {
		t.Fatalf("owner status = %d", got)
	}
}
