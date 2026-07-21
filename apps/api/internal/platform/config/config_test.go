package config

import (
	"testing"
	"time"
)

func TestLoadUsesDefaultsAndAllowsMissingDatabaseURL(t *testing.T) {
	clearEnvironment(t)

	got, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if got.Environment != defaultEnvironment {
		t.Errorf("Environment = %q, want %q", got.Environment, defaultEnvironment)
	}
	if got.HTTPAddress != defaultHTTPAddress {
		t.Errorf("HTTPAddress = %q, want %q", got.HTTPAddress, defaultHTTPAddress)
	}
	if got.DatabaseURL != "" {
		t.Errorf("DatabaseURL = %q, want empty", got.DatabaseURL)
	}
	if got.ShutdownTimeout != defaultShutdownTimeout {
		t.Errorf("ShutdownTimeout = %v, want %v", got.ShutdownTimeout, defaultShutdownTimeout)
	}
	if got.DatabasePingTimeout != defaultDatabasePingTimeout {
		t.Errorf("DatabasePingTimeout = %v, want %v", got.DatabasePingTimeout, defaultDatabasePingTimeout)
	}
}

func TestLoadReadsEnvironment(t *testing.T) {
	clearEnvironment(t)
	t.Setenv("SYSAP_ENV", "test")
	t.Setenv("SYSAP_HTTP_ADDR", "127.0.0.1:9090")
	t.Setenv("SYSAP_DATABASE_URL", "postgresql://local.example/sysap")
	t.Setenv("SYSAP_SHUTDOWN_TIMEOUT", "15s")
	t.Setenv("SYSAP_DATABASE_PING_TIMEOUT", "750ms")

	got, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if got.Environment != "test" || got.HTTPAddress != "127.0.0.1:9090" {
		t.Fatalf("Load() did not preserve environment values: %+v", got)
	}
	if got.DatabaseURL != "postgresql://local.example/sysap" {
		t.Fatal("Load() did not preserve SYSAP_DATABASE_URL")
	}
	if got.ShutdownTimeout != 15*time.Second || got.DatabasePingTimeout != 750*time.Millisecond {
		t.Fatalf("Load() did not parse durations: %+v", got)
	}
}

func TestLoadDoesNotRejectDatabaseURL(t *testing.T) {
	clearEnvironment(t)
	t.Setenv("SYSAP_DATABASE_URL", "://invalid")

	got, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v; database availability must not block startup", err)
	}
	if got.DatabaseURL != "://invalid" {
		t.Fatalf("DatabaseURL = %q, want original value for pool preparation", got.DatabaseURL)
	}
}

func TestLoadRejectsInvalidConfiguration(t *testing.T) {
	tests := []struct {
		name  string
		key   string
		value string
	}{
		{name: "invalid shutdown duration", key: "SYSAP_SHUTDOWN_TIMEOUT", value: "later"},
		{name: "zero shutdown duration", key: "SYSAP_SHUTDOWN_TIMEOUT", value: "0s"},
		{name: "negative database timeout", key: "SYSAP_DATABASE_PING_TIMEOUT", value: "-1s"},
		{name: "address without port", key: "SYSAP_HTTP_ADDR", value: "localhost"},
		{name: "address with invalid port", key: "SYSAP_HTTP_ADDR", value: ":70000"},
		{name: "address with whitespace", key: "SYSAP_HTTP_ADDR", value: " :8080"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			clearEnvironment(t)
			t.Setenv(test.key, test.value)

			if _, err := Load(); err == nil {
				t.Fatal("Load() error = nil, want configuration error")
			}
		})
	}
}

func clearEnvironment(t *testing.T) {
	t.Helper()
	for _, key := range []string{
		"SYSAP_ENV",
		"SYSAP_HTTP_ADDR",
		"SYSAP_DATABASE_URL",
		"SYSAP_SHUTDOWN_TIMEOUT",
		"SYSAP_DATABASE_PING_TIMEOUT",
	} {
		t.Setenv(key, "")
	}
}
