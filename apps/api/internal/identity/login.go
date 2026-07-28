package identity

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/ccauepereira/SysAP/apps/api/internal/platform/database"
	"github.com/ccauepereira/SysAP/apps/api/internal/platform/httpserver"
)

const (
	loginBlockPeriod = 15 * time.Minute
	loginMaxFailures = 5
	loginMaxBody     = 4096
)

var errLoginDenied = errors.New("login denied")

// PasswordIdentityProvider is the only boundary that sees a password. Its
// caller never logs or persists either the password or returned tokens.
type PasswordIdentityProvider interface {
	Authenticate(context.Context, uuid.UUID, string) (ProviderSession, error)
}

// SessionIdentityProvider is deliberately narrower than the password
// provider: the only client credential it receives is a refresh token held in
// the request body for the duration of the call.
type SessionIdentityProvider interface {
	Refresh(context.Context, string) (ProviderSession, error)
}

type ProviderSession struct {
	SubjectID    uuid.UUID
	SessionID    uuid.UUID
	AAL          string
	AccessToken  string
	RefreshToken string
	ExpiresIn    int
}

type unavailablePasswordIdentityProvider struct{}

func (unavailablePasswordIdentityProvider) Authenticate(context.Context, uuid.UUID, string) (ProviderSession, error) {
	return ProviderSession{}, errLoginDenied
}

type unavailableSessionIdentityProvider struct{}

func (unavailableSessionIdentityProvider) Refresh(context.Context, string) (ProviderSession, error) {
	return ProviderSession{}, errLoginDenied
}

type loginHandler struct {
	database *database.Pool
	pepper   []byte
	provider PasswordIdentityProvider
	now      func() time.Time
}

func newLoginHandler(databasePool *database.Pool, pepper []byte, provider PasswordIdentityProvider, now func() time.Time) http.Handler {
	return &loginHandler{database: databasePool, pepper: pepper, provider: provider, now: now}
}

func NewLoginHandler(databasePool *database.Pool, pepper string) http.Handler {
	provider := NewSupabasePasswordIdentityProvider()
	return newLoginHandler(databasePool, []byte(pepper), provider, time.Now)
}

func (h *loginHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	request, ok := decodeLoginRequest(w, r)
	if !ok || h.database == nil || len(h.pepper) < 32 || h.provider == nil {
		h.writeInvalidCredentials(w, r)
		return
	}

	// The enrollment is part of the HMAC input only. It never reaches a log,
	// audit metadata, or stored database column in clear text.
	fingerprint := loginFingerprint(h.pepper, normalizedRemoteAddress(r)+"\x00"+request.EnrollmentNumber)

	var response loginSuccessResponse
	denied := false
	err := h.database.WithTransaction(r.Context(), func(tx pgx.Tx) error {
		if _, err := tx.Exec(r.Context(), "select set_config('sysap.login_flow', 'true', true)"); err != nil {
			return err
		}

		blocked, err := reserveLoginAttempt(r.Context(), tx, fingerprint, h.now().UTC())
		if err != nil {
			return err
		}
		identity, found, err := loadLoginIdentity(r.Context(), tx, request.EnrollmentNumber)
		if err != nil {
			return err
		}
		if blocked {
			denied = true
			return writeLoginAudit(r.Context(), tx, identity, found, "failure", "rate_limited", fingerprint)
		}
		if !found || !identity.eligible() {
			if err := writeLoginAudit(r.Context(), tx, identity, found, "failure", "invalid_credentials", fingerprint); err != nil {
				return err
			}
			denied = true
			return nil
		}

		session, err := h.provider.Authenticate(r.Context(), identity.AuthUserID, request.Password)
		if err != nil || session.SubjectID != identity.AuthUserID || session.SessionID == uuid.Nil || session.AAL != "aal1" || session.AccessToken == "" || session.RefreshToken == "" || session.ExpiresIn <= 0 {
			if auditErr := writeLoginAudit(r.Context(), tx, identity, true, "failure", "invalid_credentials", fingerprint); auditErr != nil {
				return auditErr
			}
			denied = true
			return nil
		}

		if _, err := tx.Exec(r.Context(), `
			update app.security_rate_limits
			set window_started_at = $2, failure_count = 0, blocked_until = null, updated_at = $2
			where key_fingerprint = $1`, fingerprint, h.now().UTC()); err != nil {
			return err
		}
		if _, err := tx.Exec(r.Context(), `
			insert into app.auth_sessions (session_id, profile_id, organization_id, assurance_level)
			values ($1, $2, $3, 'aal1')`, session.SessionID, identity.ProfileID, identity.OrganizationID); err != nil {
			return err
		}
		if err := writeLoginAudit(r.Context(), tx, identity, true, "success", "authenticated", fingerprint); err != nil {
			return err
		}

		response = loginSuccessResponse{
			SessionID:      session.SessionID,
			ProfileID:      identity.ProfileID,
			OrganizationID: identity.OrganizationID,
			Role:           identity.Role,
			AAL:            "aal1",
			AccessToken:    session.AccessToken,
			RefreshToken:   session.RefreshToken,
			ExpiresIn:      session.ExpiresIn,
		}
		return nil
	})
	if err != nil || denied {
		h.writeInvalidCredentials(w, r)
		return
	}
	writeJSON(w, http.StatusOK, response)
}

type loginRequest struct {
	EnrollmentNumber string `json:"enrollment_number"`
	Password         string `json:"password"`
}

func decodeLoginRequest(w http.ResponseWriter, r *http.Request) (loginRequest, bool) {
	defer r.Body.Close()
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, loginMaxBody))
	decoder.DisallowUnknownFields()
	var request loginRequest
	if err := decoder.Decode(&request); err != nil || decoder.Decode(&struct{}{}) != io.EOF || !activationEnrollment.MatchString(request.EnrollmentNumber) || request.Password == "" || len(request.Password) > 1024 {
		return loginRequest{}, false
	}
	return request, true
}

type loginIdentity struct {
	ProfileID      uuid.UUID
	AuthUserID     uuid.UUID
	OrganizationID uuid.UUID
	Role           string
	ProfileActive  bool
	OrganizationOK bool
	MembershipOK   bool
}

func (i loginIdentity) eligible() bool {
	return i.ProfileID != uuid.Nil && i.AuthUserID != uuid.Nil && i.ProfileActive && i.OrganizationOK && i.MembershipOK && (i.Role == "athlete" || i.Role == "trainer" || i.Role == "owner")
}

func loadLoginIdentity(ctx context.Context, tx pgx.Tx, enrollment string) (loginIdentity, bool, error) {
	var identity loginIdentity
	var suspendedAt *time.Time
	var organizationStatus, membershipStatus string
	err := tx.QueryRow(ctx, `
		select enrollment.profile_id, profile.auth_user_id, enrollment.organization_id,
		       membership.role, profile.suspended_at, organization.status, membership.status
		from app.login_enrollments enrollment
		join app.profiles profile on profile.id = enrollment.profile_id
		join app.organizations organization on organization.id = enrollment.organization_id
		join app.organization_memberships membership
		  on membership.profile_id = enrollment.profile_id
		 and membership.organization_id = enrollment.organization_id
		where enrollment.enrollment_number = $1
		limit 1`, enrollment).Scan(&identity.ProfileID, &identity.AuthUserID, &identity.OrganizationID, &identity.Role, &suspendedAt, &organizationStatus, &membershipStatus)
	if errors.Is(err, pgx.ErrNoRows) {
		return loginIdentity{}, false, nil
	}
	if err != nil {
		return loginIdentity{}, false, err
	}
	identity.ProfileActive = suspendedAt == nil
	identity.OrganizationOK = organizationStatus == "active"
	identity.MembershipOK = membershipStatus == "active"
	return identity, true, nil
}

func reserveLoginAttempt(ctx context.Context, tx pgx.Tx, fingerprint []byte, now time.Time) (bool, error) {
	var blockedUntil *time.Time
	err := tx.QueryRow(ctx, `
		insert into app.security_rate_limits (key_fingerprint, window_started_at, failure_count, blocked_until, updated_at)
		values ($1, $2, 1, null, $2)
		on conflict (key_fingerprint) do update
		set window_started_at = case
		        when app.security_rate_limits.blocked_until is not null and app.security_rate_limits.blocked_until > excluded.updated_at then app.security_rate_limits.window_started_at
		        when app.security_rate_limits.window_started_at + interval '15 minutes' <= excluded.updated_at then excluded.updated_at
		        else app.security_rate_limits.window_started_at
		    end,
		    failure_count = case
		        when app.security_rate_limits.blocked_until is not null and app.security_rate_limits.blocked_until > excluded.updated_at then app.security_rate_limits.failure_count
		        when app.security_rate_limits.window_started_at + interval '15 minutes' <= excluded.updated_at then 1
		        when app.security_rate_limits.failure_count >= 5 then 5
		        else app.security_rate_limits.failure_count + 1
		    end,
		    blocked_until = case
		        when app.security_rate_limits.blocked_until is not null and app.security_rate_limits.blocked_until > excluded.updated_at then app.security_rate_limits.blocked_until
		        when app.security_rate_limits.window_started_at + interval '15 minutes' <= excluded.updated_at then null
		        when app.security_rate_limits.failure_count >= 5 then excluded.updated_at + interval '15 minutes'
		        else null
		    end,
		    updated_at = excluded.updated_at
		returning blocked_until`, fingerprint, now).Scan(&blockedUntil)
	if err != nil {
		return false, err
	}
	return blockedUntil != nil && blockedUntil.After(now), nil
}

func writeLoginAudit(ctx context.Context, tx pgx.Tx, identity loginIdentity, found bool, result, reason string, networkFingerprint []byte) error {
	var organizationID any
	var profileID any
	if found {
		organizationID = identity.OrganizationID
		profileID = identity.ProfileID
	}
	eventType := "login_failed"
	if result == "success" {
		eventType = "login_success"
	}
	_, err := tx.Exec(ctx, `
		insert into app.security_audit_events (
			organization_id, actor_profile_id, target_profile_id, event_type, result,
			reason_code, resource_type, resource_id, request_id, network_fingerprint, metadata
		) values ($1, null, $2, $3, $4, $5, 'auth_session', null, $6, encode($7, 'hex'), '{}'::jsonb)`,
		organizationID, profileID, eventType, result, reason, httpserver.RequestIDFromContext(ctx), networkFingerprint)
	return err
}

func loginFingerprint(pepper []byte, value string) []byte {
	mac := hmac.New(sha256.New, pepper)
	_, _ = mac.Write([]byte(value))
	return mac.Sum(nil)
}

func normalizedRemoteAddress(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		host = r.RemoteAddr
	}
	ip := net.ParseIP(strings.TrimSpace(host))
	if ip == nil {
		return "unparseable"
	}
	if ipv4 := ip.To4(); ipv4 != nil {
		return "v4:" + ipv4.String()
	}
	return "v6:" + ip.String()
}

type loginSuccessResponse struct {
	SessionID      uuid.UUID `json:"session_id"`
	ProfileID      uuid.UUID `json:"profile_id"`
	OrganizationID uuid.UUID `json:"organization_id"`
	Role           string    `json:"role"`
	AAL            string    `json:"aal"`
	AccessToken    string    `json:"access_token"`
	RefreshToken   string    `json:"refresh_token"`
	ExpiresIn      int       `json:"expires_in"`
}

func (h *loginHandler) writeInvalidCredentials(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusUnauthorized, errorResponse{Error: errorDetail{
		Code: "invalid_credentials", Message: "invalid credentials", RequestID: httpserver.RequestIDFromContext(r.Context()),
	}})
}

// parseProviderSession claims only the session identifier and AAL needed to
// register the already-provider-authenticated session. It never treats JWT
// roles as authorization data; protected routes verify the token separately.
func parseProviderSession(accessToken string, expectedSubject uuid.UUID) (uuid.UUID, string, error) {
	parts := strings.Split(accessToken, ".")
	if len(parts) != 3 || len(accessToken) > 16*1024 {
		return uuid.Nil, "", errLoginDenied
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil || len(payload) == 0 || len(payload) > 8*1024 {
		return uuid.Nil, "", errLoginDenied
	}
	var claims struct {
		Subject   string `json:"sub"`
		SessionID string `json:"session_id"`
		AAL       string `json:"aal"`
	}
	if json.NewDecoder(bytes.NewReader(payload)).Decode(&claims) != nil || claims.Subject != expectedSubject.String() || claims.AAL != "aal1" {
		return uuid.Nil, "", errLoginDenied
	}
	sessionID, err := uuid.Parse(claims.SessionID)
	if err != nil || sessionID == uuid.Nil {
		return uuid.Nil, "", errLoginDenied
	}
	return sessionID, claims.AAL, nil
}
