package auth

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/lestrrat-go/jwx/v3/jwa"
	"github.com/lestrrat-go/jwx/v3/jwk"
	"github.com/lestrrat-go/jwx/v3/jwt"

	"github.com/ccauepereira/SysAP/apps/api/internal/platform/config"
)

var verificationTime = time.Date(2026, time.July, 28, 12, 0, 0, 0, time.UTC)

func TestTokenVerifierAcceptsOnlyValidatedES256Claims(t *testing.T) {
	key := newSigningKey(t, "key-1")
	var requests atomic.Int32
	server := newJWKSServer(t, func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		writeJWKS(t, w, key)
	})
	verifier := newTestVerifier(t, server.URL, verificationTime)

	if requests.Load() != 0 {
		t.Fatal("constructing a verifier must not fetch JWKS")
	}
	token := signedToken(t, key, tokenClaims{})
	got, err := verifier.Verify(context.Background(), token)
	if err != nil {
		t.Fatalf("Verify() error = %v", err)
	}
	if got.SubjectID == uuid.Nil || got.SessionID == uuid.Nil || got.AAL != AAL1 {
		t.Fatalf("Verify() = %+v, want valid token with default aal1", got)
	}
	if requests.Load() != 1 {
		t.Fatalf("JWKS requests = %d, want 1", requests.Load())
	}
}

func TestTokenVerifierRejectsInvalidTokenFormsAndClaims(t *testing.T) {
	key := newSigningKey(t, "key-1")
	server := newJWKSServer(t, func(w http.ResponseWriter, _ *http.Request) { writeJWKS(t, w, key) })
	verifier := newTestVerifier(t, server.URL, verificationTime)

	valid := signedToken(t, key, tokenClaims{})
	wrongKey := newSigningKey(t, "key-1")
	withoutKID := signedTokenWithoutKID(t, key, tokenClaims{})
	unknownKID := signedToken(t, newSigningKey(t, "other-key"), tokenClaims{})

	tests := []struct {
		name  string
		token string
	}{
		{name: "invalid signature", token: corruptSignature(t, valid)},
		{name: "alg none", token: compactToken(`{"alg":"none","kid":"key-1"}`)},
		{name: "HS256", token: compactToken(`{"alg":"HS256","kid":"key-1"}`)},
		{name: "RS256", token: compactToken(`{"alg":"RS256","kid":"key-1"}`)},
		{name: "missing kid", token: withoutKID},
		{name: "unknown kid", token: unknownKID},
		{name: "wrong signing key", token: signedToken(t, wrongKey, tokenClaims{})},
		{name: "wrong issuer", token: signedToken(t, key, tokenClaims{issuer: "wrong-issuer"})},
		{name: "wrong audience", token: signedToken(t, key, tokenClaims{audience: []string{"wrong-audience"}})},
		{name: "expired", token: signedToken(t, key, tokenClaims{expiration: verificationTime})},
		{name: "not before future", token: signedToken(t, key, tokenClaims{notBefore: verificationTime.Add(time.Second)})},
		{name: "missing subject", token: signedToken(t, key, tokenClaims{omitSubject: true})},
		{name: "invalid subject UUID", token: signedToken(t, key, tokenClaims{subject: "not-a-uuid"})},
		{name: "missing session", token: signedToken(t, key, tokenClaims{omitSessionID: true})},
		{name: "invalid session UUID", token: signedToken(t, key, tokenClaims{sessionID: "not-a-uuid"})},
		{name: "invalid AAL", token: signedToken(t, key, tokenClaims{aal: stringPointer("aal3")})},
		{name: "conflicting role", token: signedToken(t, key, tokenClaims{role: stringPointer("service_role")})},
		{name: "untrusted JWK header", token: compactToken(`{"alg":"ES256","kid":"key-1","jku":"https://attacker.invalid/jwks"}`)},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := verifier.Verify(context.Background(), test.token)
			if !errorsIsInvalidToken(err) {
				t.Fatalf("Verify() error = %v, want safe invalid-token failure", err)
			}
			if err != nil && strings.Contains(err.Error(), test.token) {
				t.Fatal("verification error exposed a token")
			}
		})
	}
}

func TestTokenVerifierFailsSafelyForJWKSResponses(t *testing.T) {
	key := newSigningKey(t, "key-1")
	token := signedToken(t, key, tokenClaims{})

	tests := []struct {
		name    string
		handler http.HandlerFunc
		timeout time.Duration
	}{
		{name: "invalid body", handler: func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte("{")) }},
		{name: "empty set", handler: func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte(`{"keys":[]}`)) }},
		{name: "server error", handler: func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusInternalServerError) }},
		{name: "redirect", handler: func(w http.ResponseWriter, r *http.Request) { http.Redirect(w, r, "/other", http.StatusFound) }},
		{name: "timeout", timeout: 10 * time.Millisecond, handler: func(w http.ResponseWriter, r *http.Request) { <-r.Context().Done() }},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			server := httptest.NewServer(test.handler)
			defer server.Close()
			timeout := test.timeout
			if timeout == 0 {
				timeout = time.Second
			}
			verifier, err := NewTokenVerifier(config.AuthConfig{
				Issuer:            "test-issuer",
				Audience:          "authenticated",
				JWKSURL:           server.URL,
				JWKSQueryTimeout:  timeout,
				JWKSCacheTTL:      5 * time.Minute,
				JWKSMaxBodyLength: 64 * 1024,
				JWKSMaxKeys:       16,
			})
			if err != nil {
				t.Fatalf("NewTokenVerifier() error = %v", err)
			}
			_, err = verifier.Verify(context.Background(), token)
			if err != ErrTokenVerificationUnavailable {
				t.Fatalf("Verify() error = %v, want safe JWKS failure", err)
			}
		})
	}
}

func TestNewTokenVerifierRejectsIncompleteConfiguration(t *testing.T) {
	if _, err := NewTokenVerifier(config.AuthConfig{}); err == nil {
		t.Fatal("NewTokenVerifier() error = nil, want invalid configuration")
	}
}

type tokenClaims struct {
	issuer        string
	audience      []string
	expiration    time.Time
	notBefore     time.Time
	subject       string
	sessionID     string
	omitSubject   bool
	omitSessionID bool
	aal           *string
	role          *string
}

func signedToken(t *testing.T, key jwk.Key, claims tokenClaims) string {
	t.Helper()
	token := jwt.New()
	issuer := claims.issuer
	if issuer == "" {
		issuer = "test-issuer"
	}
	audience := claims.audience
	if audience == nil {
		audience = []string{"authenticated"}
	}
	expiration := claims.expiration
	if expiration.IsZero() {
		expiration = verificationTime.Add(time.Hour)
	}
	subject := claims.subject
	if subject == "" && !claims.omitSubject {
		subject = uuid.NewString()
	}
	sessionID := claims.sessionID
	if sessionID == "" && !claims.omitSessionID {
		sessionID = uuid.NewString()
	}

	for name, value := range map[string]any{
		jwt.IssuerKey:     issuer,
		jwt.AudienceKey:   audience,
		jwt.ExpirationKey: expiration,
		jwt.SubjectKey:    subject,
		"session_id":      sessionID,
	} {
		if err := token.Set(name, value); err != nil {
			t.Fatalf("token.Set(%q) error = %v", name, err)
		}
	}
	if !claims.notBefore.IsZero() {
		if err := token.Set(jwt.NotBeforeKey, claims.notBefore); err != nil {
			t.Fatal(err)
		}
	}
	if claims.aal != nil {
		if err := token.Set("aal", *claims.aal); err != nil {
			t.Fatal(err)
		}
	}
	if claims.role != nil {
		if err := token.Set("role", *claims.role); err != nil {
			t.Fatal(err)
		}
	}
	signed, err := jwt.Sign(token, jwt.WithKey(jwa.ES256(), key))
	if err != nil {
		t.Fatalf("jwt.Sign() error = %v", err)
	}
	return string(signed)
}

func signedTokenWithoutKID(t *testing.T, key jwk.Key, claims tokenClaims) string {
	t.Helper()
	// A cloned JWK without kid keeps the same private material for signing.
	withoutKID, err := key.Clone()
	if err != nil {
		t.Fatal(err)
	}
	if err := withoutKID.Remove("kid"); err != nil {
		t.Fatal(err)
	}
	return signedToken(t, withoutKID, claims)
}

func newSigningKey(t *testing.T, kid string) jwk.Key {
	t.Helper()
	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	key, err := jwk.Import(privateKey)
	if err != nil {
		t.Fatal(err)
	}
	if err := key.Set("kid", kid); err != nil {
		t.Fatal(err)
	}
	if err := key.Set("alg", "ES256"); err != nil {
		t.Fatal(err)
	}
	return key
}

func newJWKSServer(t *testing.T, handler http.HandlerFunc) *httptest.Server {
	t.Helper()
	return httptest.NewServer(handler)
}

func writeJWKS(t *testing.T, writer http.ResponseWriter, privateKey jwk.Key) {
	t.Helper()
	publicKey, err := privateKey.PublicKey()
	if err != nil {
		t.Fatal(err)
	}
	set := jwk.NewSet()
	if err := set.AddKey(publicKey); err != nil {
		t.Fatal(err)
	}
	writer.Header().Set("Content-Type", "application/jwk-set+json")
	if err := json.NewEncoder(writer).Encode(set); err != nil {
		t.Fatal(err)
	}
}

func newTestVerifier(t *testing.T, jwksURL string, now time.Time) *jwtVerifier {
	t.Helper()
	verifier, err := newTokenVerifier(config.AuthConfig{
		Issuer:            "test-issuer",
		Audience:          "authenticated",
		JWKSURL:           jwksURL,
		JWKSQueryTimeout:  time.Second,
		JWKSCacheTTL:      5 * time.Minute,
		JWKSMaxBodyLength: 64 * 1024,
		JWKSMaxKeys:       16,
	}, &http.Client{Timeout: time.Second, CheckRedirect: func(_ *http.Request, _ []*http.Request) error {
		return http.ErrUseLastResponse
	}}, func() time.Time { return now })
	if err != nil {
		t.Fatal(err)
	}
	return verifier
}

func compactToken(header string) string {
	return base64.RawURLEncoding.EncodeToString([]byte(header)) + ".e30.signature"
}

func corruptSignature(t *testing.T, token string) string {
	t.Helper()
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		t.Fatal("test token is not compact JWT")
	}
	signature, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil || len(signature) == 0 {
		t.Fatal("test token has no decodable signature")
	}
	signature[0] ^= 1
	parts[2] = base64.RawURLEncoding.EncodeToString(signature)
	return strings.Join(parts, ".")
}

func stringPointer(value string) *string { return &value }

func errorsIsInvalidToken(err error) bool { return err == ErrInvalidToken }
