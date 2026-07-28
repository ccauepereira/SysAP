package identity

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestSupabaseAuthAdminRejectsUnsafeURLs(t *testing.T) {
	for _, rawURL := range []string{
		"",
		"https://user@example.test",
		"http://example.test",
		"http://127.0.0.1:9999?x=1",
		"ftp://127.0.0.1",
	} {
		if _, err := newSupabaseAuthAdmin(rawURL, "test-key", "production", nil); err == nil {
			t.Fatal("unsafe Auth URL was accepted")
		}
	}
}

func TestSupabaseAuthAdminHandlesOnlySafeResponses(t *testing.T) {
	userID := uuid.New()
	cases := []struct {
		name     string
		status   int
		body     string
		redirect bool
		wantOK   bool
	}{
		{name: "success", status: http.StatusCreated, body: `{"id":"` + userID.String() + `"}`, wantOK: true},
		{name: "negative status", status: http.StatusBadGateway, body: `{}`},
		{name: "invalid JSON", status: http.StatusCreated, body: `{`},
		{name: "oversized body", status: http.StatusCreated, body: strings.Repeat("x", authAdminMaxBody+1)},
		{name: "redirect", status: http.StatusFound, redirect: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if tc.redirect {
					http.Redirect(w, r, "/other", http.StatusFound)
					return
				}
				w.WriteHeader(tc.status)
				_, _ = w.Write([]byte(tc.body))
			}))
			defer server.Close()
			admin, err := newSupabaseAuthAdmin(server.URL, "test-key", "test", nil)
			if err != nil {
				t.Fatal(err)
			}
			got, err := admin.CreateUser(context.Background(), "+15555550101", "not-logged-test-password")
			if tc.wantOK && (err != nil || got != userID) {
				t.Fatal("valid response was not accepted")
			}
			if !tc.wantOK && err == nil {
				t.Fatal("unsafe response was accepted")
			}
		})
	}
}

func TestSupabaseAuthAdminTimeoutFailsSafely(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		time.Sleep(50 * time.Millisecond)
		w.WriteHeader(http.StatusCreated)
	}))
	defer server.Close()
	client := &http.Client{Timeout: time.Millisecond, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	admin, err := newSupabaseAuthAdmin(server.URL, "test-key", "test", client)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := admin.CreateUser(context.Background(), "+15555550101", "not-logged-test-password"); err == nil {
		t.Fatal("timeout was accepted")
	}
}
