package auth

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/ccauepereira/SysAP/apps/api/internal/platform/database"
)

func TestPostgresSessionResolverPhase2C(t *testing.T) {
	databaseURL := os.Getenv("SYSAP_TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("integration database is not configured")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	adminPool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		t.Fatal("could not prepare integration administration pool")
	}
	defer adminPool.Close()
	apiPool, err := database.NewPool(ctx, databaseURL)
	if err != nil {
		t.Fatal("could not prepare integration API pool")
	}
	defer apiPool.Close()
	resolver, err := NewPostgresSessionResolver(apiPool)
	if err != nil {
		t.Fatal(err)
	}

	t.Run("active session and profile resolve only the authenticated subject", func(t *testing.T) {
		fixture := createSessionResolverFixture(t, adminPool, ctx)
		defer fixture.remove(t, adminPool, ctx)

		got, err := resolver.Resolve(ctx, fixture.token(), "integration-request-id")
		if err != nil {
			t.Fatalf("Resolve() error = %v", err)
		}
		if got.SubjectID != fixture.subjectID || got.ProfileID != fixture.profileID || got.SessionID != fixture.sessionID || got.AAL != AAL1 || got.RequestID != "integration-request-id" {
			t.Fatalf("Resolve() = %+v, want active fixture identity", got)
		}
	})

	t.Run("revoked session is denied", func(t *testing.T) {
		fixture := createSessionResolverFixture(t, adminPool, ctx)
		defer fixture.remove(t, adminPool, ctx)
		if _, err := adminPool.Exec(ctx, "update app.auth_sessions set revoked_at = now(), revocation_reason = 'fixture revocation' where session_id = $1", fixture.sessionID); err != nil {
			t.Fatal("could not revoke session fixture")
		}
		assertResolverDenied(t, resolver, ctx, fixture.token())
	})

	t.Run("session of a different subject is denied by RLS", func(t *testing.T) {
		fixture := createSessionResolverFixture(t, adminPool, ctx)
		defer fixture.remove(t, adminPool, ctx)
		foreignToken := fixture.token()
		foreignToken.SubjectID = uuid.New()
		assertResolverDenied(t, resolver, ctx, foreignToken)
	})

	t.Run("suspended profile is denied", func(t *testing.T) {
		fixture := createSessionResolverFixture(t, adminPool, ctx)
		defer fixture.remove(t, adminPool, ctx)
		if _, err := adminPool.Exec(ctx, "update app.profiles set suspended_at = now() where id = $1", fixture.profileID); err != nil {
			t.Fatal("could not suspend profile fixture")
		}
		assertResolverDenied(t, resolver, ctx, fixture.token())
	})
}

func assertResolverDenied(t *testing.T, resolver SessionResolver, ctx context.Context, token VerifiedToken) {
	t.Helper()
	if _, err := resolver.Resolve(ctx, token, "integration-request-id"); !errors.Is(err, ErrUnauthenticated) {
		t.Fatalf("Resolve() error = %v, want ErrUnauthenticated", err)
	}
}

type sessionResolverFixture struct {
	profileID uuid.UUID
	subjectID uuid.UUID
	sessionID uuid.UUID
}

func (f sessionResolverFixture) token() VerifiedToken {
	return VerifiedToken{SubjectID: f.subjectID, SessionID: f.sessionID, AAL: AAL1}
}

func createSessionResolverFixture(t *testing.T, pool *pgxpool.Pool, ctx context.Context) sessionResolverFixture {
	t.Helper()
	fixture := sessionResolverFixture{profileID: uuid.New(), subjectID: uuid.New(), sessionID: uuid.New()}
	transaction, err := pool.Begin(ctx)
	if err != nil {
		t.Fatal("could not begin session resolver fixture")
	}
	defer func() { _ = transaction.Rollback(ctx) }()
	if _, err := transaction.Exec(ctx, "set local session_replication_role = replica"); err != nil {
		t.Fatal("could not prepare fictional identity fixture")
	}
	if _, err := transaction.Exec(ctx, "insert into app.profiles (id, auth_user_id, full_name) values ($1, $2, 'JWT fixture profile')", fixture.profileID, fixture.subjectID); err != nil {
		t.Fatal("could not create fictional profile")
	}
	if _, err := transaction.Exec(ctx, "insert into app.auth_sessions (session_id, profile_id, assurance_level) values ($1, $2, 'aal1')", fixture.sessionID, fixture.profileID); err != nil {
		t.Fatal("could not create fictional session")
	}
	if err := transaction.Commit(ctx); err != nil {
		t.Fatal("could not commit fictional identity fixture")
	}
	return fixture
}

func (f sessionResolverFixture) remove(t *testing.T, pool *pgxpool.Pool, ctx context.Context) {
	t.Helper()
	for _, statement := range []struct {
		query string
		id    uuid.UUID
	}{
		{query: "delete from app.auth_sessions where session_id = $1", id: f.sessionID},
		{query: "delete from app.profiles where id = $1", id: f.profileID},
	} {
		if _, err := pool.Exec(ctx, statement.query, statement.id); err != nil && !errors.Is(err, pgx.ErrNoRows) {
			t.Fatal("could not remove fictional identity fixture")
		}
	}
}
