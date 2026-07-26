package database

import (
	"context"
	"errors"
	"net/url"
	"testing"

	"github.com/jackc/pgx/v5"
)

func TestWithAuthenticatedContextRejectsIncompleteContextWithoutDatabase(t *testing.T) {
	pool := &Pool{}

	if err := pool.WithAuthenticatedContext(context.Background(), AuthenticatedContext{}, func(pgx.Tx) error {
		return nil
	}); !errors.Is(err, ErrMissingAuthenticatedSubjectID) {
		t.Fatalf("WithAuthenticatedContext() error = %v, want missing subject ID", err)
	}
}

func TestNewPoolRejectsInvalidURLWithoutConnecting(t *testing.T) {
	if _, err := NewPool(context.Background(), "://invalid"); err == nil {
		t.Fatal("NewPool() error = nil, want invalid URL error")
	}
}

func TestNewPoolDoesNotRequireDatabaseAtStartup(t *testing.T) {
	fixtureURL := &url.URL{
		Scheme: "postgresql",
		User:   url.UserPassword("fixture-user", "fixture-password"),
		Host:   "127.0.0.1:1",
		Path:   "/fixture-database",
	}
	query := fixtureURL.Query()
	query.Set("sslmode", "disable")
	fixtureURL.RawQuery = query.Encode()

	pool, err := NewPool(
		context.Background(),
		fixtureURL.String(),
	)
	if err != nil {
		t.Fatalf("NewPool() error = %v", err)
	}
	pool.Close()
}
