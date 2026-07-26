package database

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrUnavailable = errors.New("database is unavailable")

var (
	ErrMissingAuthenticatedSubjectID = errors.New("authenticated database context requires a subject ID")
	ErrMissingAuthenticatedSessionID = errors.New("authenticated database context requires a session ID")
	ErrNilTransactionAction          = errors.New("authenticated database context requires a transaction action")
)

type Unavailable struct{}

func (Unavailable) Ping(context.Context) error {
	return ErrUnavailable
}

type Pool struct {
	pool *pgxpool.Pool
}

// AuthenticatedContext contains UUIDs already validated by the future token
// verifier. OrganizationID is optional because global self-service operations,
// such as the planned GET /v1/me, do not select a tenant.
type AuthenticatedContext struct {
	SubjectID      pgtype.UUID
	SessionID      pgtype.UUID
	OrganizationID pgtype.UUID
}

// NewPool parses the connection configuration without contacting PostgreSQL.
// Connectivity is checked only when Ping is called by the readiness handler.
func NewPool(ctx context.Context, databaseURL string) (*Pool, error) {
	configuration, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		return nil, err
	}

	configuration.AfterConnect = func(ctx context.Context, connection *pgx.Conn) error {
		_, err := connection.Exec(ctx, "set role sysap_api")
		return err
	}

	pool, err := pgxpool.NewWithConfig(ctx, configuration)
	if err != nil {
		return nil, err
	}

	return &Pool{pool: pool}, nil
}

func (p *Pool) Ping(ctx context.Context) error {
	return p.pool.Ping(ctx)
}

// WithAuthenticatedContext executes action in a single transaction after
// setting local identity GUCs with fixed, parameterized SQL. GUC values cannot
// outlive the transaction or be interpolated into a query.
func (p *Pool) WithAuthenticatedContext(
	ctx context.Context,
	identity AuthenticatedContext,
	action func(pgx.Tx) error,
) error {
	if !identity.SubjectID.Valid {
		return ErrMissingAuthenticatedSubjectID
	}
	if !identity.SessionID.Valid {
		return ErrMissingAuthenticatedSessionID
	}
	if action == nil {
		return ErrNilTransactionAction
	}

	transaction, err := p.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = transaction.Rollback(ctx) }()

	organizationID := ""
	if identity.OrganizationID.Valid {
		organizationID = identity.OrganizationID.String()
	}

	for _, setting := range []struct {
		query string
		value string
	}{
		{query: "select set_config('app.current_auth_subject_id', $1, true)", value: identity.SubjectID.String()},
		{query: "select set_config('app.current_auth_session_id', $1, true)", value: identity.SessionID.String()},
		{query: "select set_config('app.current_organization_id', $1, true)", value: organizationID},
	} {
		if _, err := transaction.Exec(ctx, setting.query, setting.value); err != nil {
			return err
		}
	}

	if err := action(transaction); err != nil {
		return err
	}

	return transaction.Commit(ctx)
}

func (p *Pool) Close() {
	p.pool.Close()
}
