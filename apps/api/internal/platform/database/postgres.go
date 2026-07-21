package database

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrUnavailable = errors.New("database is unavailable")

type Unavailable struct{}

func (Unavailable) Ping(context.Context) error {
	return ErrUnavailable
}

type Pool struct {
	pool *pgxpool.Pool
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

func (p *Pool) Close() {
	p.pool.Close()
}
