package main

import "testing"

func TestBootstrapRequiresLoopbackAuth(t *testing.T) {
	t.Setenv("SYSAP_ENV", "development")
	t.Setenv("SYSAP_DATABASE_URL", "postgres://fixture@127.0.0.1:55322/postgres")
	t.Setenv("SYSAP_SUPABASE_SERVICE_ROLE_KEY", "fixture")
	for _, test := range []struct {
		name, url string
		allowed   bool
	}{
		{"local", "http://127.0.0.1:55321/auth/v1", true},
		{"remote_https", "https://example.test/auth/v1", false},
		{"remote_http", "http://example.test/auth/v1", false},
		{"invalid_scheme", "ftp://127.0.0.1/auth/v1", false},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Setenv("SYSAP_SUPABASE_AUTH_URL", test.url)
			if (validateEnvironment() == nil) != test.allowed {
				t.Fatal("unexpected loopback validation result")
			}
		})
	}
}

func TestBootstrapRejectsDatabaseHostOverrides(t *testing.T) {
	t.Setenv("SYSAP_ENV", "development")
	t.Setenv("SYSAP_SUPABASE_AUTH_URL", "http://127.0.0.1:55321/auth/v1")
	t.Setenv("SYSAP_SUPABASE_SERVICE_ROLE_KEY", "fixture")
	for _, value := range []string{
		"postgres://fixture@remote.example.test/postgres",
		"postgres://fixture@127.0.0.1/postgres?host=remote.example.test",
		"postgres://fixture@127.0.0.1/postgres?hostaddr=192.0.2.1",
		"postgres:///postgres?service=remote",
	} {
		t.Setenv("SYSAP_DATABASE_URL", value)
		if validateEnvironment() == nil {
			t.Fatal("nonlocal or overridden database host was accepted")
		}
	}
}
