package httpserver

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRequestMiddlewareUsesServerGeneratedID(t *testing.T) {
	handler := withRequestContext(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := requestIDFromContext(r.Context()); got != "generated-id" {
			t.Errorf("request ID in context = %q", got)
		}
		w.WriteHeader(http.StatusNoContent)
	}), func() string { return "generated-id" })

	request := httptest.NewRequest(http.MethodGet, "/", nil)
	request.Header.Set(requestIDHeader, "untrusted-client-id")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)

	if got := response.Header().Get(requestIDHeader); got != "generated-id" {
		t.Fatalf("X-Request-ID = %q, want generated-id", got)
	}
}

func TestRequestLoggerWritesStructuredCompletionEntry(t *testing.T) {
	var output bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&output, nil))
	handler := withRequestContext(
		logRequests(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte("{}"))
		}), logger),
		func() string { return "logged-request-id" },
	)

	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/example?ignored=true", nil))

	var entry map[string]any
	if err := json.Unmarshal(output.Bytes(), &entry); err != nil {
		t.Fatalf("log is not valid JSON: %v", err)
	}
	if entry["request_id"] != "logged-request-id" || entry["method"] != "POST" || entry["path"] != "/example" {
		t.Fatalf("unexpected request log: %v", entry)
	}
	if entry["status"] != float64(http.StatusCreated) || entry["response_bytes"] != float64(2) {
		t.Fatalf("unexpected response metadata: %v", entry)
	}
	if _, exists := entry["duration_ms"]; !exists {
		t.Fatalf("request log is missing duration_ms: %v", entry)
	}
}
