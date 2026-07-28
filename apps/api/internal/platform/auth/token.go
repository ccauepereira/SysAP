// Package auth validates provider-issued tokens and resolves their local,
// revocable SysAP identity. It deliberately has no knowledge of HTTP headers
// or domain roles.
package auth

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/lestrrat-go/jwx/v3/jwa"
	"github.com/lestrrat-go/jwx/v3/jwk"
	"github.com/lestrrat-go/jwx/v3/jwt"

	"github.com/ccauepereira/SysAP/apps/api/internal/platform/config"
)

const maxTokenLength = 16 * 1024
const maxJWKSBodyLength = 64 * 1024

var (
	// ErrInvalidToken deliberately carries no provider, claim, signature, or
	// key details. HTTP middleware maps it to the fixed 401 envelope.
	ErrInvalidToken = errors.New("token is invalid")
	// ErrTokenVerificationUnavailable is distinct for metrics and retry policy,
	// but has the same external response as ErrInvalidToken.
	ErrTokenVerificationUnavailable = errors.New("token verification is unavailable")
)

// AssuranceLevel is validated rather than trusted as a role. It is retained
// for a later policy that can require aal2 for sensitive actions.
type AssuranceLevel string

const (
	AAL1 AssuranceLevel = "aal1"
	AAL2 AssuranceLevel = "aal2"
)

// VerifiedToken contains only claims needed to resolve local identity. No
// domain role, organization, raw claim map, or bearer token is retained.
type VerifiedToken struct {
	SubjectID uuid.UUID
	SessionID uuid.UUID
	AAL       AssuranceLevel
}

// TokenVerifier does not know headers or handlers. This keeps token parsing
// independently testable and prevents a handler from bypassing verification.
type TokenVerifier interface {
	Verify(context.Context, string) (VerifiedToken, error)
}

type jwtVerifier struct {
	issuer   string
	audience string
	jwksURL  string
	client   *http.Client
	now      func() time.Time
}

// NewTokenVerifier validates complete server-side configuration without doing
// I/O. A temporarily unavailable JWKS therefore never prevents API startup.
func NewTokenVerifier(configuration config.AuthConfig) (TokenVerifier, error) {
	if !configuration.Configured() || configuration.Issuer == "" || configuration.Audience == "" || configuration.JWKSURL == "" || configuration.JWKSQueryTimeout <= 0 {
		return nil, errors.New("JWT verifier configuration is incomplete")
	}

	verifier, err := newTokenVerifier(configuration, &http.Client{
		Timeout: configuration.JWKSQueryTimeout,
		CheckRedirect: func(_ *http.Request, _ []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}, time.Now)
	if err != nil {
		return nil, err
	}
	return verifier, nil
}

func newTokenVerifier(configuration config.AuthConfig, client *http.Client, now func() time.Time) (*jwtVerifier, error) {
	if client == nil || now == nil {
		return nil, errors.New("JWT verifier dependencies are required")
	}
	return &jwtVerifier{
		issuer:   configuration.Issuer,
		audience: configuration.Audience,
		jwksURL:  configuration.JWKSURL,
		client:   client,
		now:      now,
	}, nil
}

func (v *jwtVerifier) Verify(ctx context.Context, rawToken string) (VerifiedToken, error) {
	header, err := parseProtectedHeader(rawToken)
	if err != nil {
		return VerifiedToken{}, ErrInvalidToken
	}

	key, err := v.fetchKey(ctx, header.KID)
	if err != nil {
		return VerifiedToken{}, err
	}

	token, err := jwt.Parse([]byte(rawToken), jwt.WithKey(jwa.ES256(), key), jwt.WithValidate(false))
	if err != nil {
		return VerifiedToken{}, ErrInvalidToken
	}

	return claimsFromToken(token, v.issuer, v.audience, v.now())
}

type protectedHeader struct {
	Algorithm string          `json:"alg"`
	KID       string          `json:"kid"`
	JWK       json.RawMessage `json:"jwk"`
	JWKURL    string          `json:"jku"`
	X509URL   string          `json:"x5u"`
	Critical  []string        `json:"crit"`
}

func parseProtectedHeader(rawToken string) (protectedHeader, error) {
	if len(rawToken) == 0 || len(rawToken) > maxTokenLength || strings.ContainsAny(rawToken, " \t\r\n") {
		return protectedHeader{}, ErrInvalidToken
	}

	parts := strings.Split(rawToken, ".")
	if len(parts) != 3 || parts[0] == "" || parts[1] == "" || parts[2] == "" {
		return protectedHeader{}, ErrInvalidToken
	}

	encodedHeader := parts[0]
	decodedHeader, err := base64.RawURLEncoding.DecodeString(encodedHeader)
	if err != nil {
		return protectedHeader{}, ErrInvalidToken
	}

	var header protectedHeader
	if err := json.Unmarshal(decodedHeader, &header); err != nil || header.Algorithm != "ES256" || strings.TrimSpace(header.KID) == "" {
		return protectedHeader{}, ErrInvalidToken
	}
	// All key material and URLs must come from the configured JWKS endpoint.
	if len(header.JWK) != 0 || header.JWKURL != "" || header.X509URL != "" || len(header.Critical) != 0 {
		return protectedHeader{}, ErrInvalidToken
	}
	return header, nil
}

func (v *jwtVerifier) fetchKey(ctx context.Context, kid string) (jwk.Key, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, v.jwksURL, nil)
	if err != nil {
		return nil, ErrTokenVerificationUnavailable
	}
	request.Header.Set("Accept", "application/jwk-set+json, application/json")

	response, err := v.client.Do(request)
	if err != nil {
		return nil, ErrTokenVerificationUnavailable
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return nil, ErrTokenVerificationUnavailable
	}

	body, err := io.ReadAll(io.LimitReader(response.Body, maxJWKSBodyLength+1))
	if err != nil || len(body) == 0 || len(body) > maxJWKSBodyLength {
		return nil, ErrTokenVerificationUnavailable
	}
	set, err := jwk.Parse(body, jwk.WithMaxKeys(16), jwk.WithRejectDuplicateKID(true))
	if err != nil || set.Len() == 0 {
		return nil, ErrTokenVerificationUnavailable
	}
	key, found := set.LookupKeyID(kid)
	if !found {
		return nil, ErrInvalidToken
	}
	if algorithm, hasAlgorithm := key.Algorithm(); hasAlgorithm && algorithm != jwa.ES256() {
		return nil, ErrInvalidToken
	}
	return key, nil
}

func claimsFromToken(token jwt.Token, issuer, audience string, now time.Time) (VerifiedToken, error) {
	actualIssuer, issuerPresent := token.Issuer()
	if !issuerPresent || actualIssuer != issuer {
		return VerifiedToken{}, ErrInvalidToken
	}

	audiences, audiencePresent := token.Audience()
	if !audiencePresent || !contains(audiences, audience) {
		return VerifiedToken{}, ErrInvalidToken
	}

	expiresAt, expiryPresent := token.Expiration()
	if !expiryPresent || !now.Before(expiresAt) {
		return VerifiedToken{}, ErrInvalidToken
	}
	if notBefore, present := token.NotBefore(); present && now.Before(notBefore) {
		return VerifiedToken{}, ErrInvalidToken
	}

	subject, subjectPresent := token.Subject()
	subjectID, err := parseNonNilUUID(subject, subjectPresent)
	if err != nil {
		return VerifiedToken{}, ErrInvalidToken
	}

	var rawSessionID string
	if err := token.Get("session_id", &rawSessionID); err != nil {
		return VerifiedToken{}, ErrInvalidToken
	}
	sessionID, err := parseNonNilUUID(rawSessionID, true)
	if err != nil {
		return VerifiedToken{}, ErrInvalidToken
	}

	aal := AAL1
	if token.Has("aal") {
		var rawAAL string
		if err := token.Get("aal", &rawAAL); err != nil {
			return VerifiedToken{}, ErrInvalidToken
		}
		aal = AssuranceLevel(rawAAL)
	}
	if aal != AAL1 && aal != AAL2 {
		return VerifiedToken{}, ErrInvalidToken
	}

	if token.Has("role") {
		var role string
		if err := token.Get("role", &role); err != nil || role != "authenticated" {
			return VerifiedToken{}, ErrInvalidToken
		}
	}

	return VerifiedToken{SubjectID: subjectID, SessionID: sessionID, AAL: aal}, nil
}

func parseNonNilUUID(value string, present bool) (uuid.UUID, error) {
	if !present || strings.TrimSpace(value) == "" {
		return uuid.Nil, ErrInvalidToken
	}
	parsed, err := uuid.Parse(value)
	if err != nil || parsed == uuid.Nil {
		return uuid.Nil, ErrInvalidToken
	}
	return parsed, nil
}

func contains(values []string, wanted string) bool {
	for _, value := range values {
		if value == wanted {
			return true
		}
	}
	return false
}
