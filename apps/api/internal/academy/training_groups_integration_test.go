package academy

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/ccauepereira/SysAP/apps/api/internal/platform/auth"
	"github.com/ccauepereira/SysAP/apps/api/internal/platform/authorization"
	"github.com/ccauepereira/SysAP/apps/api/internal/platform/database"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestTrainingGroupRequiresActiveOwnerAndMFA(t *testing.T) {
	dbURL := os.Getenv("SYSAP_TEST_DATABASE_URL")
	if dbURL == "" {
		t.Skip("isolated PostgreSQL not configured")
	}
	ctx := context.Background()
	admin, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		t.Fatal("cannot open isolated test database")
	}
	t.Cleanup(admin.Close)
	pool, err := database.NewPool(ctx, dbURL)
	if err != nil {
		t.Fatal("cannot open application test pool")
	}
	t.Cleanup(pool.Close)
	orgID, profileID, subjectID := uuid.New(), uuid.New(), uuid.New()
	otherOrg, athleteProfile, athleteSubject := uuid.New(), uuid.New(), uuid.New()
	t.Cleanup(func() {
		_, _ = admin.Exec(ctx, "delete from app.training_groups where organization_id = any($1::uuid[])", []uuid.UUID{orgID, otherOrg})
		_, _ = admin.Exec(ctx, "delete from app.mfa_factors where profile_id = $1", profileID)
		_, _ = admin.Exec(ctx, "delete from app.organization_memberships where organization_id = any($1::uuid[])", []uuid.UUID{orgID, otherOrg})
		_, _ = admin.Exec(ctx, "delete from app.profiles where id = any($1::uuid[])", []uuid.UUID{profileID, athleteProfile})
		_, _ = admin.Exec(ctx, "delete from app.organizations where id = any($1::uuid[])", []uuid.UUID{orgID, otherOrg})
		_, _ = admin.Exec(ctx, "delete from auth.users where id = any($1::uuid[])", []uuid.UUID{subjectID, athleteSubject})
	})
	queries := []struct {
		sql  string
		args []any
	}{
		{"insert into auth.users (id, aud, role, email, raw_app_meta_data, raw_user_meta_data, created_at, updated_at) values ($1, 'authenticated', 'authenticated', $2, '{}'::jsonb, '{}'::jsonb, now(), now())", []any{subjectID, subjectID.String() + "@example.test"}},
		{"insert into auth.users (id, aud, role, email, raw_app_meta_data, raw_user_meta_data, created_at, updated_at) values ($1, 'authenticated', 'authenticated', $2, '{}'::jsonb, '{}'::jsonb, now(), now())", []any{athleteSubject, athleteSubject.String() + "@example.test"}},
		{"insert into app.organizations (id, name, timezone, status) values ($1, 'Fictional Org', 'America/Fortaleza', 'active'), ($2, 'Other Fictional Org', 'America/Fortaleza', 'active')", []any{orgID, otherOrg}},
		{"insert into app.profiles (id, auth_user_id, full_name) values ($1, $2, 'Fictional Owner'), ($3, $4, 'Fictional Athlete')", []any{profileID, subjectID, athleteProfile, athleteSubject}},
		{"insert into app.organization_memberships (organization_id, profile_id, role, status) values ($1, $2, 'owner', 'active'), ($1, $3, 'athlete', 'active')", []any{orgID, profileID, athleteProfile}},
		{"insert into app.mfa_factors (profile_id, provider_factor_id, status, verified_at) values ($1, $2, 'verified', now())", []any{profileID, uuid.New()}},
	}
	for _, q := range queries {
		if _, err := admin.Exec(ctx, q.sql, q.args...); err != nil {
			t.Fatal("failed to set up isolated fixtures")
		}
	}
	service := groupService{pool: pool}
	owner := auth.AuthenticatedContext{SubjectID: subjectID, ProfileID: profileID, SessionID: uuid.New(), AAL: auth.AAL2}
	group, err := service.create(ctx, owner, orgID.String(), "Turma fictícia")
	if err != nil || group.ID == uuid.Nil || group.Name != "Turma fictícia" {
		t.Fatal("verified owner could not create group")
	}
	if _, err := service.create(ctx, owner, orgID.String(), "Turma fictícia"); !errors.Is(err, errGroupConflict) {
		t.Fatal("duplicate group was not rejected")
	}
	owner.AAL = auth.AAL1
	if _, err := service.create(ctx, owner, orgID.String(), "Sem MFA"); !errors.Is(err, errAccessDenied) {
		t.Fatal("owner without MFA was not rejected")
	}
	owner.AAL = auth.AAL2
	if _, err := service.create(ctx, owner, otherOrg.String(), "Outra organização"); !errors.Is(err, authorization.ErrUnauthorized) {
		t.Fatal("cross-organization access was not rejected")
	}
	athlete := auth.AuthenticatedContext{SubjectID: athleteSubject, ProfileID: athleteProfile, SessionID: uuid.New(), AAL: auth.AAL2}
	if _, err := service.create(ctx, athlete, orgID.String(), "Sem permissão"); !errors.Is(err, errAccessDenied) {
		t.Fatal("athlete was not rejected")
	}
	handler := auth.Middleware(testVerifier{}, testResolver{actor: owner})(NewCreateTrainingGroupHandler(pool))
	request := httptest.NewRequest(http.MethodPost, "/v1/organizations/"+orgID.String()+"/training-groups", strings.NewReader(`{"name":"Turma HTTP"}`))
	request.SetPathValue("organization_id", orgID.String())
	request.Header.Set("X-Organization-ID", orgID.String())
	request.Header.Set("Authorization", "Bearer fixture")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusCreated {
		t.Fatalf("HTTP create status = %d", response.Code)
	}
	var count int
	if err := admin.QueryRow(ctx, "select count(*) from app.training_groups where organization_id = $1", orgID).Scan(&count); err != nil || count != 2 {
		t.Fatal("unexpected persisted groups")
	}
}

type testVerifier struct{}

func (testVerifier) Verify(context.Context, string) (auth.VerifiedToken, error) {
	return auth.VerifiedToken{}, nil
}

type testResolver struct{ actor auth.AuthenticatedContext }

func (r testResolver) Resolve(context.Context, auth.VerifiedToken, string) (auth.AuthenticatedContext, error) {
	return r.actor, nil
}

func TestTrainingGroupHandlerRejectsUnauthenticated(t *testing.T) {
	request := httptest.NewRequest(http.MethodPost, "/v1/organizations/invalid/training-groups", strings.NewReader(`{"name":"Fictícia"}`))
	response := httptest.NewRecorder()
	NewCreateTrainingGroupHandler(nil).ServeHTTP(response, request)
	if response.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", response.Code)
	}
}
