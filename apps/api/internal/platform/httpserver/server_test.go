package httpserver

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

type checkerStub struct {
	err      error
	deadline time.Time
}

func (c *checkerStub) Ping(ctx context.Context) error {
	c.deadline, _ = ctx.Deadline()
	return c.err
}

func TestHealthIsIndependentFromDatabase(t *testing.T) {
	database := &checkerStub{err: errors.New("database unavailable")}
	handler := newHandler(database, discardLogger(), time.Second, fixedRequestID)
	response := performRequest(handler, http.MethodGet, "/healthz")

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusOK)
	}
	assertJSON(t, response, map[string]any{
		"status":  "ok",
		"service": "sysap-api",
	})
	assertCommonHeaders(t, response)
}

func TestReadinessReportsAvailableDatabase(t *testing.T) {
	database := &checkerStub{}
	handler := newHandler(database, discardLogger(), 250*time.Millisecond, fixedRequestID)
	startedAt := time.Now()
	response := performRequest(handler, http.MethodGet, "/readyz")

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusOK)
	}
	assertJSON(t, response, map[string]any{
		"status":  "ready",
		"service": "sysap-api",
		"dependencies": map[string]any{
			"database": "ready",
		},
	})
	if database.deadline.Before(startedAt.Add(200 * time.Millisecond)) {
		t.Fatalf("database ping deadline = %v, want configured timeout", database.deadline)
	}
}

func TestReadinessReturnsSafeErrorWhenDatabaseIsUnavailable(t *testing.T) {
	databaseURL := "postgresql://user:secret@example.invalid"
	database := &checkerStub{err: errors.New(databaseURL)}
	var logOutput bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&logOutput, nil))
	handler := newHandler(database, logger, time.Second, fixedRequestID)
	response := performRequest(handler, http.MethodGet, "/readyz")

	if response.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusServiceUnavailable)
	}
	assertJSON(t, response, map[string]any{
		"error": map[string]any{
			"code":       "service_not_ready",
			"message":    "service is not ready",
			"request_id": "server-request-id",
		},
	})
	wantBody := "{\"error\":{\"code\":\"service_not_ready\",\"message\":\"service is not ready\",\"request_id\":\"server-request-id\"}}\n"
	if got := response.Body.String(); got != wantBody {
		t.Fatalf("body = %q, want exact contract %q", got, wantBody)
	}
	if bytes.Contains(response.Body.Bytes(), []byte(databaseURL)) {
		t.Fatal("response exposed the database error")
	}
	if bytes.Contains(logOutput.Bytes(), []byte(databaseURL)) {
		t.Fatal("log exposed the database error")
	}
	assertCommonHeaders(t, response)
}

func TestReadinessRecoversWithoutRecreatingHandler(t *testing.T) {
	database := &checkerStub{err: errors.New("database is starting")}
	handler := newHandler(database, discardLogger(), time.Second, fixedRequestID)

	firstResponse := performRequest(handler, http.MethodGet, "/readyz")
	if firstResponse.Code != http.StatusServiceUnavailable {
		t.Fatalf("initial status = %d, want %d", firstResponse.Code, http.StatusServiceUnavailable)
	}

	database.err = nil
	secondResponse := performRequest(handler, http.MethodGet, "/readyz")
	if secondResponse.Code != http.StatusOK {
		t.Fatalf("recovered status = %d, want %d", secondResponse.Code, http.StatusOK)
	}
}

func TestNewServerConfiguresHTTPTimeouts(t *testing.T) {
	server := NewServer(":8080", http.NewServeMux())

	if server.ReadHeaderTimeout <= 0 || server.ReadTimeout <= 0 || server.WriteTimeout <= 0 || server.IdleTimeout <= 0 {
		t.Fatalf("server has unsafe timeouts: %+v", server)
	}
	if server.MaxHeaderBytes != maxHeaderBytes {
		t.Fatalf("MaxHeaderBytes = %d, want %d", server.MaxHeaderBytes, maxHeaderBytes)
	}
}

func performRequest(handler http.Handler, method, target string) *httptest.ResponseRecorder {
	request := httptest.NewRequest(method, target, nil)
	request.Header.Set(requestIDHeader, "client-request-id")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	return response
}

func assertCommonHeaders(t *testing.T, response *httptest.ResponseRecorder) {
	t.Helper()
	if got := response.Header().Get("Content-Type"); got != "application/json; charset=utf-8" {
		t.Errorf("Content-Type = %q", got)
	}
	if got := response.Header().Get("Cache-Control"); got != "no-store" {
		t.Errorf("Cache-Control = %q", got)
	}
	if got := response.Header().Get(requestIDHeader); got != "server-request-id" {
		t.Errorf("X-Request-ID = %q, want server-generated ID", got)
	}
}

func assertJSON(t *testing.T, response *httptest.ResponseRecorder, want map[string]any) {
	t.Helper()
	var got map[string]any
	if err := json.Unmarshal(response.Body.Bytes(), &got); err != nil {
		t.Fatalf("response is not valid JSON: %v", err)
	}
	if !mapsEqual(got, want) {
		t.Fatalf("body = %v, want %v", got, want)
	}
}

func mapsEqual(left, right map[string]any) bool {
	leftJSON, _ := json.Marshal(left)
	rightJSON, _ := json.Marshal(right)
	return bytes.Equal(leftJSON, rightJSON)
}

func fixedRequestID() string {
	return "server-request-id"
}

func discardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(bytes.NewBuffer(nil), nil))
}
