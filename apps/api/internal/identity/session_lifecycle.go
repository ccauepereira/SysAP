package identity

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/ccauepereira/SysAP/apps/api/internal/platform/auth"
	"github.com/ccauepereira/SysAP/apps/api/internal/platform/database"
	"github.com/ccauepereira/SysAP/apps/api/internal/platform/httpserver"
)

const refreshMaxBody = 4096

type sessionLifecycleHandler struct {
	database *database.Pool
	provider SessionIdentityProvider
}

func newSessionLifecycleHandler(databasePool *database.Pool, provider SessionIdentityProvider) *sessionLifecycleHandler {
	return &sessionLifecycleHandler{database: databasePool, provider: provider}
}

func NewSessionLifecycleHandler(databasePool *database.Pool) http.Handler {
	return newSessionLifecycleHandler(databasePool, NewSupabaseSessionIdentityProvider())
}

func (h *sessionLifecycleHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.URL.Path {
	case "/v1/auth/refresh":
		h.refresh(w, r)
	case "/v1/auth/logout":
		h.logout(w, r, false)
	case "/v1/auth/logout-all":
		h.logout(w, r, true)
	default:
		http.NotFound(w, r)
	}
}

type refreshSessionRequest struct {
	RefreshToken string `json:"refresh_token"`
}

type refreshSessionResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
}

func (h *sessionLifecycleHandler) refresh(w http.ResponseWriter, r *http.Request) {
	request, ok := decodeRefreshSessionRequest(w, r)
	if !ok || h.database == nil || h.provider == nil {
		writeSessionInvalidCredentials(w, r)
		return
	}

	// This is the sole point where the opaque refresh token is used. It is not
	// attached to context, written to a database, returned on error, or logged.
	providerSession, err := h.provider.Refresh(r.Context(), request.RefreshToken)
	if err != nil || !validProviderSession(providerSession) {
		writeSessionInvalidCredentials(w, r)
		return
	}

	identity := database.AuthenticatedContext{
		SubjectID: pgtype.UUID{Bytes: providerSession.SubjectID, Valid: true},
		SessionID: pgtype.UUID{Bytes: providerSession.SessionID, Valid: true},
	}
	err = h.database.WithAuthenticatedContext(r.Context(), identity, func(tx pgx.Tx) error {
		if _, err := tx.Exec(r.Context(), "select set_config('sysap.session_flow', 'true', true)"); err != nil {
			return err
		}
		state, err := loadSessionLifecycleState(r.Context(), tx, providerSession.SessionID, true)
		if err != nil {
			return err
		}
		if !state.eligible(providerSession.SubjectID) || state.AssuranceLevel != providerSession.AAL {
			return errLoginDenied
		}
		return writeSessionAudit(r.Context(), tx, state, "session_refreshed")
	})
	if err != nil {
		writeSessionInvalidCredentials(w, r)
		return
	}

	writeJSON(w, http.StatusOK, refreshSessionResponse{
		AccessToken: providerSession.AccessToken, RefreshToken: providerSession.RefreshToken,
		TokenType: "Bearer", ExpiresIn: providerSession.ExpiresIn,
	})
}

func (h *sessionLifecycleHandler) logout(w http.ResponseWriter, r *http.Request, all bool) {
	if h.database == nil {
		httpserver.WriteAuthenticationRequired(w, r.Context())
		return
	}
	authenticated, ok := auth.AuthenticatedContextFromContext(r.Context())
	if !ok {
		httpserver.WriteAuthenticationRequired(w, r.Context())
		return
	}
	identity := database.AuthenticatedContext{
		SubjectID: pgtype.UUID{Bytes: authenticated.SubjectID, Valid: true},
		SessionID: pgtype.UUID{Bytes: authenticated.SessionID, Valid: true},
	}
	err := h.database.WithAuthenticatedContext(r.Context(), identity, func(tx pgx.Tx) error {
		if _, err := tx.Exec(r.Context(), "select set_config('sysap.session_flow', 'true', true)"); err != nil {
			return err
		}
		state, err := loadSessionLifecycleState(r.Context(), tx, authenticated.SessionID, true)
		if err != nil {
			return err
		}
		if !state.eligible(authenticated.SubjectID) || state.ProfileID != authenticated.ProfileID {
			return errLoginDenied
		}

		var count int64
		if all {
			command, err := tx.Exec(r.Context(), `
				update app.auth_sessions
				set revoked_at = now(), revocation_reason = 'logout_all'
				where profile_id = $1 and revoked_at is null`, state.ProfileID)
			if err != nil {
				return err
			}
			count = command.RowsAffected()
		} else {
			command, err := tx.Exec(r.Context(), `
				update app.auth_sessions
				set revoked_at = now(), revocation_reason = 'logout'
				where session_id = $1 and profile_id = $2 and revoked_at is null`, authenticated.SessionID, state.ProfileID)
			if err != nil {
				return err
			}
			count = command.RowsAffected()
		}
		// A retry that reaches this point is still terminal and does not create a
		// duplicate audit event. Normally middleware rejects a revoked bearer.
		if count == 0 {
			return nil
		}
		eventType := "session_revoked"
		scope := "current_session"
		var syncSessionID any = authenticated.SessionID
		if all {
			eventType, scope, syncSessionID = "sessions_revoked", "all_sessions", nil
		}
		if err := writeSessionAudit(r.Context(), tx, state, eventType); err != nil {
			return err
		}
		_, err = tx.Exec(r.Context(), `
			insert into app.auth_session_revocation_sync (profile_id, session_id, scope, status)
			values ($1, $2, $3, 'pending')`, state.ProfileID, syncSessionID, scope)
		return err
	})
	if err != nil {
		httpserver.WriteAuthenticationRequired(w, r.Context())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

type sessionLifecycleState struct {
	ProfileID      uuid.UUID
	OrganizationID uuid.UUID
	AuthUserID     uuid.UUID
	AssuranceLevel string
	ProfileActive  bool
	OrganizationOK bool
	MembershipOK   bool
}

func (s sessionLifecycleState) eligible(subjectID uuid.UUID) bool {
	return s.ProfileID != uuid.Nil && s.OrganizationID != uuid.Nil && s.AuthUserID == subjectID &&
		s.ProfileActive && s.OrganizationOK && s.MembershipOK
}

func loadSessionLifecycleState(ctx context.Context, tx pgx.Tx, sessionID uuid.UUID, lock bool) (sessionLifecycleState, error) {
	var state sessionLifecycleState
	var suspendedAt *time.Time
	var organizationStatus, membershipStatus string
	query := `
		select session.profile_id, session.organization_id, profile.auth_user_id,
		       session.assurance_level, profile.suspended_at, organization.status, membership.status
		from app.auth_sessions session
		join app.profiles profile on profile.id = session.profile_id
		join app.organizations organization on organization.id = session.organization_id
		join app.organization_memberships membership
		  on membership.profile_id = session.profile_id
		 and membership.organization_id = session.organization_id
		where session.session_id = $1 and session.revoked_at is null`
	if lock {
		query += " for update of session"
	}
	err := tx.QueryRow(ctx, query, sessionID).Scan(
		&state.ProfileID, &state.OrganizationID, &state.AuthUserID, &state.AssuranceLevel,
		&suspendedAt, &organizationStatus, &membershipStatus,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return sessionLifecycleState{}, errLoginDenied
	}
	if err != nil {
		return sessionLifecycleState{}, err
	}
	state.ProfileActive = suspendedAt == nil
	state.OrganizationOK = organizationStatus == "active"
	state.MembershipOK = membershipStatus == "active"
	return state, nil
}

func writeSessionAudit(ctx context.Context, tx pgx.Tx, state sessionLifecycleState, eventType string) error {
	_, err := tx.Exec(ctx, `
		insert into app.security_audit_events (
			organization_id, actor_profile_id, target_profile_id, event_type, result,
			reason_code, resource_type, resource_id, request_id, network_fingerprint, metadata
		) values ($1, $2, $2, $3, 'success', 'authenticated', 'auth_session', null, $4, '', '{}'::jsonb)`,
		state.OrganizationID, state.ProfileID, eventType, httpserver.RequestIDFromContext(ctx))
	return err
}

func decodeRefreshSessionRequest(w http.ResponseWriter, r *http.Request) (refreshSessionRequest, bool) {
	defer r.Body.Close()
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, refreshMaxBody))
	decoder.DisallowUnknownFields()
	var request refreshSessionRequest
	if err := decoder.Decode(&request); err != nil || decoder.Decode(&struct{}{}) != io.EOF || request.RefreshToken == "" || len(request.RefreshToken) > 3072 {
		return refreshSessionRequest{}, false
	}
	return request, true
}

func validProviderSession(session ProviderSession) bool {
	return session.SubjectID != uuid.Nil && session.SessionID != uuid.Nil && session.AAL == "aal1" &&
		session.AccessToken != "" && session.RefreshToken != "" && session.ExpiresIn > 0
}

func writeSessionInvalidCredentials(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusUnauthorized, errorResponse{Error: errorDetail{
		Code: "invalid_credentials", Message: "invalid credentials", RequestID: httpserver.RequestIDFromContext(r.Context()),
	}})
}
