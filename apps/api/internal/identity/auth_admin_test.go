package identity

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
)

type roundTripperFunc func(*http.Request) (*http.Response, error)

func (f roundTripperFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func TestSupabaseAuthAdmin_CreateUserWithEmail(t *testing.T) {
	expectedAPIKey := "test-service-role-key"
	expectedID := uuid.New()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/auth/v1/admin/users" {
			t.Errorf("Unexpected path: %s", r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
			return
		}
		if r.Header.Get("apikey") != expectedAPIKey {
			t.Errorf("Missing or incorrect apikey header: %s", r.Header.Get("apikey"))
		}
		if r.Header.Get("Authorization") != "Bearer "+expectedAPIKey {
			t.Errorf("Missing or incorrect Authorization header: %s", r.Header.Get("Authorization"))
		}
		if r.Method != http.MethodPost {
			t.Errorf("Unexpected method: %s", r.Method)
		}

		action := r.Header.Get("X-Test-Action")

		if action == "timeout" {
			time.Sleep(50 * time.Millisecond)
			return
		}

		if action == "invalid-json" {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("{invalid-json"))
			return
		}

		if action == "403" {
			w.WriteHeader(http.StatusForbidden)
			return
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]any{"id": expectedID})
	}))
	defer server.Close()

	createAuthAdmin := func(action string, timeout time.Duration) SupabaseAuthAdmin {
		client := server.Client()
		client.Timeout = timeout

		// Create a completely new transport for this client
		client.Transport = roundTripperFunc(func(req *http.Request) (*http.Response, error) {
			if action != "" {
				req.Header.Set("X-Test-Action", action)
			}
			return http.DefaultTransport.RoundTrip(req)
		})

		authAdmin, err := newSupabaseAuthAdmin(server.URL+"/auth/v1", expectedAPIKey, "development", client)
		if err != nil {
			t.Fatalf("Failed to create auth admin: %v", err)
		}
		return authAdmin
	}

	ctx := context.Background()

	t.Run("Valid creation", func(t *testing.T) {
		authAdmin := createAuthAdmin("", 1*time.Second)
		id, err := authAdmin.CreateUserWithEmail(ctx, "test@example.com", "+5511999999999", "password123")
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		if id != expectedID {
			t.Errorf("Expected ID %s, got %s", expectedID, id)
		}
	})

	t.Run("403 response", func(t *testing.T) {
		authAdmin := createAuthAdmin("403", 1*time.Second)
		_, err := authAdmin.CreateUserWithEmail(ctx, "test@example.com", "+5511999999999", "password123")
		if err == nil {
			t.Fatal("Expected error, got nil")
		}
		if err != errAuthUnavailable {
			t.Errorf("Expected errAuthUnavailable, got %v", err)
		}
		if strings.Contains(err.Error(), expectedAPIKey) {
			t.Errorf("Error message leaks secret key: %v", err)
		}
	})

	t.Run("Timeout", func(t *testing.T) {
		authAdmin := createAuthAdmin("timeout", 10*time.Millisecond)
		_, err := authAdmin.CreateUserWithEmail(ctx, "test@example.com", "+5511999999999", "password123")
		if err == nil {
			t.Fatal("Expected error, got nil")
		}
		if err != errAuthUnavailable {
			t.Errorf("Expected errAuthUnavailable, got %v", err)
		}
		if strings.Contains(err.Error(), expectedAPIKey) {
			t.Errorf("Error message leaks secret key: %v", err)
		}
	})

	t.Run("Invalid JSON", func(t *testing.T) {
		authAdmin := createAuthAdmin("invalid-json", 1*time.Second)
		_, err := authAdmin.CreateUserWithEmail(ctx, "test@example.com", "+5511999999999", "password123")
		if err == nil {
			t.Fatal("Expected error, got nil")
		}
		if err != errAuthUnavailable {
			t.Errorf("Expected errAuthUnavailable, got: %v", err)
		}
	})
}

func TestProviders_MissingConfigFallback(t *testing.T) {
	t.Setenv("SYSAP_SUPABASE_AUTH_URL", "")
	t.Setenv("SYSAP_SUPABASE_SERVICE_ROLE_KEY", "")

	pwdProvider := NewSupabasePasswordIdentityProvider()
	if _, ok := pwdProvider.(unavailablePasswordIdentityProvider); !ok {
		t.Errorf("Expected unavailablePasswordIdentityProvider when config is missing, got %T", pwdProvider)
	}

	sessionProvider := NewSupabaseSessionIdentityProvider()
	if _, ok := sessionProvider.(unavailableSessionIdentityProvider); !ok {
		t.Errorf("Expected unavailableSessionIdentityProvider when config is missing, got %T", sessionProvider)
	}
}

func TestProviders_ValidConfig(t *testing.T) {
	t.Setenv("SYSAP_ENV", "development")
	t.Setenv("SYSAP_SUPABASE_AUTH_URL", "http://127.0.0.1:54321/auth/v1")
	t.Setenv("SYSAP_SUPABASE_SERVICE_ROLE_KEY", "dummy-key")

	pwdProvider := NewSupabasePasswordIdentityProvider()
	if _, ok := pwdProvider.(*supabasePasswordIdentityProvider); !ok {
		t.Errorf("Expected *supabasePasswordIdentityProvider when config is present, got %T", pwdProvider)
	}

	sessionProvider := NewSupabaseSessionIdentityProvider()
	if _, ok := sessionProvider.(*supabaseSessionIdentityProvider); !ok {
		t.Errorf("Expected *supabaseSessionIdentityProvider when config is present, got %T", sessionProvider)
	}
}
