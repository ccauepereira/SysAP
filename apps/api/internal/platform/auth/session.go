package auth

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/ccauepereira/SysAP/apps/api/internal/platform/database"
)

var ErrUnauthenticated = errors.New("authentication could not be completed")

// AuthenticatedContext is an immutable value stored by the middleware. Its
// private context key and value semantics prevent unrelated handlers from
// manufacturing or mutating the authenticated request identity.
type AuthenticatedContext struct {
	SubjectID uuid.UUID
	ProfileID uuid.UUID
	SessionID uuid.UUID
	AAL       AssuranceLevel
	RequestID string
}

// SessionResolver checks the authoritative local session and profile state.
// It deliberately does not receive domain role or organization claims.
type SessionResolver interface {
	Resolve(context.Context, VerifiedToken, string) (AuthenticatedContext, error)
}

type postgresSessionResolver struct {
	database *database.Pool
}

func NewPostgresSessionResolver(pool *database.Pool) (SessionResolver, error) {
	if pool == nil {
		return nil, errors.New("PostgreSQL session resolver requires a pool")
	}
	return &postgresSessionResolver{database: pool}, nil
}

func (r *postgresSessionResolver) Resolve(ctx context.Context, token VerifiedToken, requestID string) (AuthenticatedContext, error) {
	identity := database.AuthenticatedContext{
		SubjectID: pgtype.UUID{Bytes: token.SubjectID, Valid: true},
		SessionID: pgtype.UUID{Bytes: token.SessionID, Valid: true},
	}

	var profileID pgtype.UUID
	err := r.database.WithAuthenticatedContext(ctx, identity, func(transaction pgx.Tx) error {
		err := transaction.QueryRow(ctx, `
			select auth_sessions.profile_id
			from app.auth_sessions
			join app.profiles on app.profiles.id = app.auth_sessions.profile_id
			where auth_sessions.session_id = $1
			  and auth_sessions.assurance_level = $2
			  and auth_sessions.revoked_at is null
			  and app.profiles.suspended_at is null
		`, token.SessionID, string(token.AAL)).Scan(&profileID)
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrUnauthenticated
		}
		return err
	})
	if err != nil {
		if errors.Is(err, ErrUnauthenticated) {
			return AuthenticatedContext{}, ErrUnauthenticated
		}
		return AuthenticatedContext{}, ErrUnauthenticated
	}
	if !profileID.Valid {
		return AuthenticatedContext{}, ErrUnauthenticated
	}

	return AuthenticatedContext{
		SubjectID: token.SubjectID,
		ProfileID: uuid.UUID(profileID.Bytes),
		SessionID: token.SessionID,
		AAL:       token.AAL,
		RequestID: requestID,
	}, nil
}

type authenticatedContextKey struct{}

func contextWithAuthenticatedContext(ctx context.Context, authenticated AuthenticatedContext) context.Context {
	return context.WithValue(ctx, authenticatedContextKey{}, authenticated)
}

// AuthenticatedContextFromContext is the only handler-facing read path. It
// returns a copy, so a handler cannot alter the context's stored principal.
func AuthenticatedContextFromContext(ctx context.Context) (AuthenticatedContext, bool) {
	authenticated, ok := ctx.Value(authenticatedContextKey{}).(AuthenticatedContext)
	return authenticated, ok
}
