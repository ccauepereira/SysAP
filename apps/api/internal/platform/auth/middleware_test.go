package auth

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/ccauepereira/SysAP/apps/api/internal/platform/httpserver"
)

func TestMiddlewareRejectsInvalidAuthorizationHeadersWithSafe401(t *testing.T) {
	tests := []struct {
		name   string
		values []string
	}{
		{name: "missing"},
		{name: "different scheme", values: []string{"Basic credential"}},
		{name: "lowercase scheme", values: []string{"bearer credential"}},
		{name: "empty token", values: []string{"Bearer "}},
		{name: "extra whitespace", values: []string{"Bearer  credential"}},
		{name: "multiple values", values: []string{"Bearer one", "Bearer two"}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			verifier := &verifierStub{}
			handler := httpserver.WithRequestContext(Middleware(verifier, &resolverStub{})(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
				t.Fatal("protected handler must not run")
			})))
			request := httptest.NewRequest(http.MethodGet, "/protected", nil)
			request.Header["Authorization"] = test.values
			response := httptest.NewRecorder()

			handler.ServeHTTP(response, request)

			assertAuthenticationRequired(t, response)
			if verifier.calls != 0 {
				t.Fatalf("verifier calls = %d, want 0", verifier.calls)
			}
		})
	}
}

func TestMiddlewareBuildsAuthenticatedContextOnlyAfterSessionResolution(t *testing.T) {
	subjectID := uuid.New()
	sessionID := uuid.New()
	verified := VerifiedToken{SubjectID: subjectID, SessionID: sessionID, AAL: AAL2}
	resolved := AuthenticatedContext{
		SubjectID: subjectID,
		ProfileID: uuid.New(),
		SessionID: sessionID,
		AAL:       AAL2,
	}
	verifier := &verifierStub{verified: verified}
	resolver := &resolverStub{authenticated: resolved}

	handler := httpserver.WithRequestContext(Middleware(verifier, resolver)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got, ok := AuthenticatedContextFromContext(r.Context())
		if !ok {
			t.Fatal("authenticated context is absent")
		}
		if got.SubjectID != subjectID || got.SessionID != sessionID || got.ProfileID != resolved.ProfileID || got.AAL != AAL2 {
			t.Fatalf("authenticated context = %+v", got)
		}
		if got.RequestID == "" || got.RequestID != resolver.requestID {
			t.Fatal("authenticated context does not carry the server request ID")
		}
		w.WriteHeader(http.StatusNoContent)
	})))
	request := httptest.NewRequest(http.MethodGet, "/protected", nil)
	request.Header.Set("Authorization", "Bearer fixture-token")
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	if response.Code != http.StatusNoContent || verifier.rawToken != "fixture-token" || resolver.token != verified {
		t.Fatalf("protected request was not resolved correctly: status=%d verifier=%+v resolver=%+v", response.Code, verifier, resolver)
	}
}

func TestMiddlewareDoesNotExposeVerifierOrSessionFailures(t *testing.T) {
	secretToken := "very-sensitive-fixture-token"
	tests := []struct {
		name     string
		verifier *verifierStub
		resolver *resolverStub
	}{
		{name: "verifier", verifier: &verifierStub{err: errors.New(secretToken)}, resolver: &resolverStub{}},
		{name: "session", verifier: &verifierStub{verified: VerifiedToken{SubjectID: uuid.New(), SessionID: uuid.New(), AAL: AAL1}}, resolver: &resolverStub{err: errors.New(secretToken)}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			handler := httpserver.WithRequestContext(Middleware(test.verifier, test.resolver)(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
				t.Fatal("protected handler must not run")
			})))
			request := httptest.NewRequest(http.MethodGet, "/protected", nil)
			request.Header.Set("Authorization", "Bearer "+secretToken)
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, request)
			assertAuthenticationRequired(t, response)
			if body := response.Body.String(); containsSensitiveValue(body, secretToken) {
				t.Fatal("authentication response exposed a sensitive value")
			}
		})
	}
}

func assertAuthenticationRequired(t *testing.T, response *httptest.ResponseRecorder) {
	t.Helper()
	if response.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusUnauthorized)
	}
	if response.Header().Get("WWW-Authenticate") != "Bearer" || response.Header().Get("X-Request-ID") == "" {
		t.Fatalf("authentication failure headers = %v", response.Header())
	}
	var body struct {
		Error struct {
			Code      string `json:"code"`
			Message   string `json:"message"`
			RequestID string `json:"request_id"`
		} `json:"error"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatalf("response is not JSON: %v", err)
	}
	if body.Error.Code != "authentication_required" || body.Error.Message != "authentication is required" || body.Error.RequestID == "" {
		t.Fatalf("authentication failure body = %+v", body)
	}
}

func containsSensitiveValue(value, sensitive string) bool { return strings.Contains(value, sensitive) }

type verifierStub struct {
	verified VerifiedToken
	err      error
	rawToken string
	calls    int
}

func (s *verifierStub) Verify(_ context.Context, rawToken string) (VerifiedToken, error) {
	s.calls++
	s.rawToken = rawToken
	return s.verified, s.err
}

type resolverStub struct {
	authenticated AuthenticatedContext
	err           error
	token         VerifiedToken
	requestID     string
}

func (s *resolverStub) Resolve(_ context.Context, token VerifiedToken, requestID string) (AuthenticatedContext, error) {
	s.token = token
	s.requestID = requestID
	authenticated := s.authenticated
	authenticated.RequestID = requestID
	return authenticated, s.err
}
