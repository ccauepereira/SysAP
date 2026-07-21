package database

import (
	"context"
	"testing"
)

func TestNewPoolRejectsInvalidURLWithoutConnecting(t *testing.T) {
	if _, err := NewPool(context.Background(), "://invalid"); err == nil {
		t.Fatal("NewPool() error = nil, want invalid URL error")
	}
}

func TestNewPoolDoesNotRequireDatabaseAtStartup(t *testing.T) {
	pool, err := NewPool(
		context.Background(),
		"postgresql://postgres:postgres@127.0.0.1:1/postgres?sslmode=disable",
	)
	if err != nil {
		t.Fatalf("NewPool() error = %v", err)
	}
	pool.Close()
}
