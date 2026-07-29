package identity

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/ccauepereira/SysAP/apps/api/internal/platform/auth"
	"github.com/ccauepereira/SysAP/apps/api/internal/platform/database"
	"github.com/ccauepereira/SysAP/apps/api/internal/platform/httpserver"
)

const (
	mfaMaxBody              = 8192
	recoveryMaxBody         = 8192
	recoveryProofLifetime   = 10 * time.Minute
	recoveryRateLimitWindow = 15 * time.Minute
)

var errMFARequired = errors.New("mfa is required")

// MFAProvider is the server-side boundary for Supabase Auth MFA. Secrets and
// provider challenges cross this interface only in request memory.
type MFAProvider interface {
	EnrollTOTP(context.Context, string) (MFAEnrollment, error)
	CreateChallenge(context.Context, string, uuid.UUID) (string, error)
	VerifyChallenge(context.Context, string, uuid.UUID, string, string, uuid.UUID) (ProviderSession, error)
	FactorActive(context.Context, string, uuid.UUID) (bool, error)
}

type MFAEnrollment struct {
	FactorID        uuid.UUID
	ProvisioningURI string
}

type unavailableMFAProvider struct{}

func (unavailableMFAProvider) EnrollTOTP(context.Context, string) (MFAEnrollment, error) {
	return MFAEnrollment{}, errAuthUnavailable
}
func (unavailableMFAProvider) CreateChallenge(context.Context, string, uuid.UUID) (string, error) {
	return "", errAuthUnavailable
}
func (unavailableMFAProvider) VerifyChallenge(context.Context, string, uuid.UUID, string, string, uuid.UUID) (ProviderSession, error) {
	return ProviderSession{}, errAuthUnavailable
}
func (unavailableMFAProvider) FactorActive(context.Context, string, uuid.UUID) (bool, error) {
	return false, errAuthUnavailable
}

type supabaseMFAProvider struct{ auth *supabaseAuthAdmin }

func NewSupabaseMFAProvider() MFAProvider {
	baseURL, roleKey := environmentAuthConfiguration()
	provider, err := newSupabaseAuthAdmin(baseURL, roleKey, environmentName(), nil)
	if err != nil {
		return unavailableMFAProvider{}
	}
	return &supabaseMFAProvider{auth: provider}
}

func environmentAuthConfiguration() (string, string) {
	return os.Getenv("SYSAP_SUPABASE_AUTH_URL"), os.Getenv("SYSAP_SUPABASE_SERVICE_ROLE_KEY")
}

func environmentName() string { return os.Getenv("SYSAP_ENV") }

func (p *supabaseMFAProvider) EnrollTOTP(ctx context.Context, bearer string) (MFAEnrollment, error) {
	var response struct {
		ID   uuid.UUID `json:"id"`
		URI  string    `json:"uri"`
		Type string    `json:"factor_type"`
	}
	if err := p.userJSON(ctx, http.MethodPost, "factors", bearer, []byte(`{"factor_type":"totp"}`), &response); err != nil || response.ID == uuid.Nil || response.URI == "" || response.Type != "" && response.Type != "totp" {
		return MFAEnrollment{}, errAuthUnavailable
	}
	return MFAEnrollment{FactorID: response.ID, ProvisioningURI: response.URI}, nil
}

func (p *supabaseMFAProvider) CreateChallenge(ctx context.Context, bearer string, factorID uuid.UUID) (string, error) {
	var response struct {
		ID string `json:"id"`
	}
	if factorID == uuid.Nil || p.userJSON(ctx, http.MethodPost, "factors/"+factorID.String()+"/challenge", bearer, nil, &response) != nil || response.ID == "" {
		return "", errAuthUnavailable
	}
	return response.ID, nil
}

func (p *supabaseMFAProvider) VerifyChallenge(ctx context.Context, bearer string, factorID uuid.UUID, challengeID, code string, subjectID uuid.UUID) (ProviderSession, error) {
	if factorID == uuid.Nil || challengeID == "" || code == "" || subjectID == uuid.Nil {
		return ProviderSession{}, errLoginDenied
	}
	payload, err := json.Marshal(struct {
		ChallengeID string `json:"challenge_id"`
		Code        string `json:"code"`
	}{challengeID, code})
	if err != nil {
		return ProviderSession{}, errLoginDenied
	}
	var response struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		ExpiresIn    int    `json:"expires_in"`
	}
	if p.userJSON(ctx, http.MethodPost, "factors/"+factorID.String()+"/verify", bearer, payload, &response) != nil {
		return ProviderSession{}, errLoginDenied
	}
	sessionID, aal, err := parseProviderSessionAtAAL(response.AccessToken, subjectID, "aal2")
	if err != nil || response.RefreshToken == "" || response.ExpiresIn <= 0 {
		return ProviderSession{}, errLoginDenied
	}
	return ProviderSession{SubjectID: subjectID, SessionID: sessionID, AAL: aal, AccessToken: response.AccessToken, RefreshToken: response.RefreshToken, ExpiresIn: response.ExpiresIn}, nil
}

func (p *supabaseMFAProvider) FactorActive(ctx context.Context, bearer string, factorID uuid.UUID) (bool, error) {
	var response struct {
		Factors []struct {
			ID     uuid.UUID `json:"id"`
			Status string    `json:"status"`
		} `json:"factors"`
	}
	if factorID == uuid.Nil || p.userJSON(ctx, http.MethodGet, "factors", bearer, nil, &response) != nil {
		return false, errAuthUnavailable
	}
	for _, factor := range response.Factors {
		if factor.ID == factorID && factor.Status == "verified" {
			return true, nil
		}
	}
	return false, nil
}

func (p *supabaseMFAProvider) userJSON(ctx context.Context, method, path, bearer string, body []byte, target any) error {
	if p == nil || p.auth == nil || bearer == "" {
		return errAuthUnavailable
	}
	endpoint := p.auth.baseURL.JoinPath(path)
	var reader io.Reader
	if body != nil {
		reader = bytes.NewReader(body)
	}
	req, err := http.NewRequestWithContext(ctx, method, endpoint.String(), reader)
	if err != nil {
		return errAuthUnavailable
	}
	req.Header.Set("apikey", p.auth.roleKey)
	req.Header.Set("Authorization", "Bearer "+bearer)
	req.Header.Set("Content-Type", "application/json")
	response, err := p.auth.client.Do(req)
	if err != nil {
		return errAuthUnavailable
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return errAuthUnavailable
	}
	if target == nil {
		return discardBounded(response.Body)
	}
	data, err := readBounded(response.Body)
	if err != nil || json.Unmarshal(data, target) != nil {
		return errAuthUnavailable
	}
	return nil
}

type mfaHandler struct {
	database *database.Pool
	provider MFAProvider
}

func NewMFAHandler(pool *database.Pool) http.Handler {
	return newMFAHandler(pool, NewSupabaseMFAProvider())
}
func newMFAHandler(pool *database.Pool, provider MFAProvider) *mfaHandler {
	return &mfaHandler{database: pool, provider: provider}
}

func (h *mfaHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.URL.Path {
	case "/v1/auth/mfa":
		if r.Method == http.MethodGet {
			h.status(w, r)
			return
		}
	case "/v1/auth/mfa/enroll":
		if r.Method == http.MethodPost {
			h.enroll(w, r)
			return
		}
	case "/v1/auth/mfa/challenge":
		if r.Method == http.MethodPost {
			h.challenge(w, r)
			return
		}
	case "/v1/auth/mfa/verify":
		if r.Method == http.MethodPost {
			h.verify(w, r)
			return
		}
	}
	http.NotFound(w, r)
}

type mfaFactorRequest struct {
	FactorID uuid.UUID `json:"factor_id"`
}
type mfaVerifyRequest struct {
	FactorID    uuid.UUID `json:"factor_id"`
	ChallengeID string    `json:"challenge_id"`
	Code        string    `json:"code"`
}

func (h *mfaHandler) enroll(w http.ResponseWriter, r *http.Request) {
	authenticated, bearer, ok := mfaRequestIdentity(r)
	if !ok || h.database == nil || h.provider == nil || authenticated.AAL != auth.AAL1 {
		httpserver.WriteAuthenticationRequired(w, r.Context())
		return
	}
	if !h.staffEligible(r.Context(), authenticated) {
		h.writeMFARequired(w, r)
		return
	}
	enrollment, err := h.provider.EnrollTOTP(r.Context(), bearer)
	if err != nil {
		writeMFASafeUnavailable(w, r)
		return
	}
	err = h.withMFAFlow(r.Context(), authenticated, func(tx pgx.Tx) error {
		_, err := tx.Exec(r.Context(), `insert into app.mfa_factors (profile_id, provider_factor_id, status)
			values ($1, $2, 'pending')
			on conflict (profile_id) do update set provider_factor_id = excluded.provider_factor_id,
			status = 'pending', verified_at = null, disabled_at = null, updated_at = now()`, authenticated.ProfileID, enrollment.FactorID)
		return err
	})
	if err != nil {
		writeMFASafeUnavailable(w, r)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"factor_id": enrollment.FactorID, "provisioning_uri": enrollment.ProvisioningURI})
}

func (h *mfaHandler) challenge(w http.ResponseWriter, r *http.Request) {
	authenticated, bearer, ok := mfaRequestIdentity(r)
	request, decoded := decodeMFARequest[mfaFactorRequest](w, r)
	if !ok || !decoded || h.database == nil || h.provider == nil || request.FactorID == uuid.Nil {
		writeMFAInvalid(w, r)
		return
	}
	if !h.ownsUsableFactor(r.Context(), authenticated, request.FactorID) {
		writeMFAInvalid(w, r)
		return
	}
	challengeID, err := h.provider.CreateChallenge(r.Context(), bearer, request.FactorID)
	if err != nil {
		writeMFASafeUnavailable(w, r)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"challenge_id": challengeID})
}

func (h *mfaHandler) verify(w http.ResponseWriter, r *http.Request) {
	authenticated, bearer, ok := mfaRequestIdentity(r)
	request, decoded := decodeMFARequest[mfaVerifyRequest](w, r)
	if !ok || !decoded || h.database == nil || h.provider == nil || request.FactorID == uuid.Nil || request.ChallengeID == "" || !sixDigits(request.Code) {
		writeMFAInvalid(w, r)
		return
	}
	if !h.ownsPendingFactor(r.Context(), authenticated, request.FactorID) {
		writeMFAInvalid(w, r)
		return
	}
	providerSession, err := h.provider.VerifyChallenge(r.Context(), bearer, request.FactorID, request.ChallengeID, request.Code, authenticated.SubjectID)
	if err != nil || providerSession.SessionID != authenticated.SessionID || providerSession.AAL != "aal2" {
		writeMFAInvalid(w, r)
		return
	}
	err = h.withMFAFlow(r.Context(), authenticated, func(tx pgx.Tx) error {
		command, err := tx.Exec(r.Context(), `update app.mfa_factors set status = 'verified', verified_at = now(), updated_at = now()
			where profile_id = $1 and provider_factor_id = $2 and status = 'pending'`, authenticated.ProfileID, request.FactorID)
		if err != nil || command.RowsAffected() != 1 {
			return errLoginDenied
		}
		command, err = tx.Exec(r.Context(), `update app.auth_sessions set assurance_level = 'aal2'
			where session_id = $1 and profile_id = $2 and assurance_level = 'aal1' and revoked_at is null`, authenticated.SessionID, authenticated.ProfileID)
		if err != nil || command.RowsAffected() != 1 {
			return errLoginDenied
		}
		return nil
	})
	if err != nil {
		writeMFAInvalid(w, r)
		return
	}
	writeJSON(w, http.StatusOK, refreshSessionResponse{AccessToken: providerSession.AccessToken, RefreshToken: providerSession.RefreshToken, TokenType: "Bearer", ExpiresIn: providerSession.ExpiresIn})
}

func (h *mfaHandler) status(w http.ResponseWriter, r *http.Request) {
	authenticated, bearer, ok := mfaRequestIdentity(r)
	if !ok || h.database == nil || h.provider == nil {
		httpserver.WriteAuthenticationRequired(w, r.Context())
		return
	}
	var factorID uuid.UUID
	var status string
	err := h.database.WithAuthenticatedContext(r.Context(), databaseIdentity(authenticated), func(tx pgx.Tx) error {
		return tx.QueryRow(r.Context(), "select provider_factor_id, status from app.mfa_factors where profile_id = $1", authenticated.ProfileID).Scan(&factorID, &status)
	})
	if errors.Is(err, pgx.ErrNoRows) {
		writeJSON(w, http.StatusOK, map[string]bool{"mfa_enabled": false})
		return
	}
	if err != nil {
		writeMFASafeUnavailable(w, r)
		return
	}
	active, err := h.provider.FactorActive(r.Context(), bearer, factorID)
	if err != nil {
		writeMFASafeUnavailable(w, r)
		return
	}
	if !active && status != "disabled" {
		_ = h.withMFAFlow(r.Context(), authenticated, func(tx pgx.Tx) error {
			_, e := tx.Exec(r.Context(), "update app.mfa_factors set status = 'disabled', disabled_at = now(), updated_at = now() where profile_id = $1 and status <> 'disabled'", authenticated.ProfileID)
			return e
		})
	}
	writeJSON(w, http.StatusOK, map[string]bool{"mfa_enabled": active})
}

func (h *mfaHandler) withMFAFlow(ctx context.Context, authenticated auth.AuthenticatedContext, action func(pgx.Tx) error) error {
	return h.database.WithAuthenticatedContext(ctx, databaseIdentity(authenticated), func(tx pgx.Tx) error {
		if _, err := tx.Exec(ctx, "select set_config('sysap.mfa_flow', 'true', true)"); err != nil {
			return err
		}
		return action(tx)
	})
}
func (h *mfaHandler) ownsUsableFactor(ctx context.Context, authenticated auth.AuthenticatedContext, factorID uuid.UUID) bool {
	return h.factorHasStatus(ctx, authenticated, factorID, "pending", "verified")
}
func (h *mfaHandler) ownsPendingFactor(ctx context.Context, authenticated auth.AuthenticatedContext, factorID uuid.UUID) bool {
	return h.factorHasStatus(ctx, authenticated, factorID, "pending")
}
func (h *mfaHandler) factorHasStatus(ctx context.Context, authenticated auth.AuthenticatedContext, factorID uuid.UUID, statuses ...string) bool {
	var count int
	err := h.database.WithAuthenticatedContext(ctx, databaseIdentity(authenticated), func(tx pgx.Tx) error {
		return tx.QueryRow(ctx, "select count(*) from app.mfa_factors where profile_id = $1 and provider_factor_id = $2 and status = any($3::text[])", authenticated.ProfileID, factorID, statuses).Scan(&count)
	})
	return err == nil && count == 1
}
func (h *mfaHandler) staffEligible(ctx context.Context, authenticated auth.AuthenticatedContext) bool {
	var count int
	err := h.database.WithAuthenticatedContext(ctx, databaseIdentity(authenticated), func(tx pgx.Tx) error {
		return tx.QueryRow(ctx, `select count(*) from app.organization_memberships where profile_id = $1 and status = 'active' and role in ('owner', 'trainer')`, authenticated.ProfileID).Scan(&count)
	})
	return err == nil && count > 0
}

// RequireAdministrativeAAL2 is the reusable local authorization guard for
// administrative handlers. The membership role comes from the database, never
// from the bearer token or request body.
func RequireAdministrativeAAL2(ctx context.Context, tx pgx.Tx, authenticated auth.AuthenticatedContext, role string) error {
	if role == "athlete" {
		return nil
	}
	if role != "owner" && role != "trainer" {
		return errMFARequired
	}
	if authenticated.AAL != auth.AAL2 {
		return errMFARequired
	}
	var enabled bool
	if err := tx.QueryRow(ctx, "select exists (select 1 from app.mfa_factors where profile_id = $1 and status = 'verified')", authenticated.ProfileID).Scan(&enabled); err != nil || !enabled {
		return errMFARequired
	}
	return nil
}

func mfaRequestIdentity(r *http.Request) (auth.AuthenticatedContext, string, bool) {
	authenticated, ok := auth.AuthenticatedContextFromContext(r.Context())
	if !ok {
		return auth.AuthenticatedContext{}, "", false
	}
	values := r.Header.Values("Authorization")
	if len(values) != 1 || !strings.HasPrefix(values[0], "Bearer ") || strings.TrimSpace(strings.TrimPrefix(values[0], "Bearer ")) == "" {
		return auth.AuthenticatedContext{}, "", false
	}
	return authenticated, strings.TrimSpace(strings.TrimPrefix(values[0], "Bearer ")), true
}
func databaseIdentity(value auth.AuthenticatedContext) database.AuthenticatedContext {
	return database.AuthenticatedContext{SubjectID: pgtype.UUID{Bytes: value.SubjectID, Valid: true}, SessionID: pgtype.UUID{Bytes: value.SessionID, Valid: true}}
}
func (h *mfaHandler) writeMFARequired(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusForbidden, errorResponse{Error: errorDetail{Code: "mfa_required", Message: "multi-factor authentication is required", RequestID: httpserver.RequestIDFromContext(r.Context())}})
}
func writeMFAInvalid(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusUnauthorized, errorResponse{Error: errorDetail{Code: "invalid_credentials", Message: "invalid credentials", RequestID: httpserver.RequestIDFromContext(r.Context())}})
}
func writeMFASafeUnavailable(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusServiceUnavailable, errorResponse{Error: errorDetail{Code: "service_unavailable", Message: "service is unavailable", RequestID: httpserver.RequestIDFromContext(r.Context())}})
}

func decodeMFARequest[T any](w http.ResponseWriter, r *http.Request) (T, bool) {
	var request T
	defer r.Body.Close()
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, mfaMaxBody))
	decoder.DisallowUnknownFields()
	if decoder.Decode(&request) != nil || decoder.Decode(&struct{}{}) != io.EOF {
		return request, false
	}
	return request, true
}
func sixDigits(value string) bool {
	if len(value) != 6 {
		return false
	}
	for _, char := range value {
		if char < '0' || char > '9' {
			return false
		}
	}
	return true
}

// RecoveryProvider keeps delivery, OTP verification, and password mutation in
// Supabase Auth. It never returns a contact, OTP, token, or provider error.
type RecoveryProvider interface {
	Start(context.Context, uuid.UUID) error
	Verify(context.Context, uuid.UUID, string) error
	SetPassword(context.Context, uuid.UUID, string) error
}
type unavailableRecoveryProvider struct{}

func (unavailableRecoveryProvider) Start(context.Context, uuid.UUID) error { return errAuthUnavailable }
func (unavailableRecoveryProvider) Verify(context.Context, uuid.UUID, string) error {
	return errAuthUnavailable
}
func (unavailableRecoveryProvider) SetPassword(context.Context, uuid.UUID, string) error {
	return errAuthUnavailable
}

type supabaseRecoveryProvider struct{ auth *supabaseAuthAdmin }

func NewSupabaseRecoveryProvider() RecoveryProvider {
	baseURL, roleKey := environmentAuthConfiguration()
	provider, err := newSupabaseAuthAdmin(baseURL, roleKey, environmentName(), nil)
	if err != nil {
		return unavailableRecoveryProvider{}
	}
	return &supabaseRecoveryProvider{auth: provider}
}
func (p *supabaseRecoveryProvider) Start(ctx context.Context, subject uuid.UUID) error {
	account, err := p.account(ctx, subject)
	if err != nil {
		return errAuthUnavailable
	}
	if account.Email != "" {
		payload, _ := json.Marshal(struct {
			Email string `json:"email"`
		}{account.Email})
		return p.auth.doJSON(ctx, http.MethodPost, "recover", bytes.NewReader(payload), nil)
	}
	if account.Phone == "" {
		return errAuthUnavailable
	}
	payload, _ := json.Marshal(struct {
		Phone            string `json:"phone"`
		ShouldCreateUser bool   `json:"should_create_user"`
	}{account.Phone, false})
	return p.auth.doJSON(ctx, http.MethodPost, "otp", bytes.NewReader(payload), nil)
}
func (p *supabaseRecoveryProvider) Verify(ctx context.Context, subject uuid.UUID, code string) error {
	if !sixDigits(code) {
		return errAuthUnavailable
	}
	account, err := p.account(ctx, subject)
	if err != nil {
		return errAuthUnavailable
	}
	if account.Email != "" {
		payload, _ := json.Marshal(struct {
			Email string `json:"email"`
			Token string `json:"token"`
			Type  string `json:"type"`
		}{account.Email, code, "recovery"})
		return p.auth.doJSON(ctx, http.MethodPost, "verify", bytes.NewReader(payload), nil)
	}
	if account.Phone == "" {
		return errAuthUnavailable
	}
	payload, _ := json.Marshal(struct {
		Phone string `json:"phone"`
		Token string `json:"token"`
		Type  string `json:"type"`
	}{account.Phone, code, "sms"})
	return p.auth.doJSON(ctx, http.MethodPost, "verify", bytes.NewReader(payload), nil)
}

func (p *supabaseRecoveryProvider) account(ctx context.Context, subject uuid.UUID) (struct {
	ID    uuid.UUID `json:"id"`
	Email string    `json:"email"`
	Phone string    `json:"phone"`
}, error) {
	var account struct {
		ID    uuid.UUID `json:"id"`
		Email string    `json:"email"`
		Phone string    `json:"phone"`
	}
	if p == nil || p.auth == nil || p.auth.getUser(ctx, subject, &account) != nil || account.ID != subject {
		return account, errAuthUnavailable
	}
	return account, nil
}
func (p *supabaseRecoveryProvider) SetPassword(ctx context.Context, subject uuid.UUID, password string) error {
	if p == nil || p.auth == nil || subject == uuid.Nil || password == "" {
		return errAuthUnavailable
	}
	payload, err := json.Marshal(struct {
		Password string `json:"password"`
	}{password})
	if err != nil {
		return errAuthUnavailable
	}
	return p.auth.doJSON(ctx, http.MethodPut, authAdminEndpoint+"/"+subject.String(), bytes.NewReader(payload), nil)
}

type recoveryHandler struct {
	database *database.Pool
	provider RecoveryProvider
	pepper   []byte
	now      func() time.Time
}

func NewPasswordRecoveryHandler(pool *database.Pool, pepper string) http.Handler {
	return newPasswordRecoveryHandler(pool, NewSupabaseRecoveryProvider(), []byte(pepper), time.Now)
}
func newPasswordRecoveryHandler(pool *database.Pool, provider RecoveryProvider, pepper []byte, now func() time.Time) *recoveryHandler {
	return &recoveryHandler{database: pool, provider: provider, pepper: pepper, now: now}
}
func (h *recoveryHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.URL.Path {
	case "/v1/auth/password-recovery/start":
		h.start(w, r)
	case "/v1/auth/password-recovery/verify":
		h.verify(w, r)
	case "/v1/auth/password-recovery/complete":
		h.complete(w, r)
	default:
		http.NotFound(w, r)
	}
}

type recoveryStartRequest struct {
	EnrollmentNumber string `json:"enrollment_number"`
}
type recoveryVerifyRequest struct {
	EnrollmentNumber string `json:"enrollment_number"`
	Code             string `json:"code"`
}
type recoveryCompleteRequest struct {
	Proof    string `json:"proof"`
	Password string `json:"password"`
}

func (h *recoveryHandler) start(w http.ResponseWriter, r *http.Request) {
	request, ok := decodeRecoveryRequest[recoveryStartRequest](w, r)
	if !ok || !validEnrollment(request.EnrollmentNumber) || h.database == nil || len(h.pepper) < 16 {
		writeRecoveryAccepted(w)
		return
	}
	identity, found, err := h.loadRecoveryIdentity(r.Context(), request.EnrollmentNumber)
	if err == nil && found && identity.eligible() && !h.recoveryLimited(r.Context(), "start", request.EnrollmentNumber, r) {
		if h.provider != nil && h.provider.Start(r.Context(), identity.AuthUserID) == nil {
			_ = h.withRecoveryFlow(r.Context(), func(tx pgx.Tx) error {
				_, e := tx.Exec(r.Context(), `insert into app.password_recovery_challenges (profile_id, organization_id, expires_at) values ($1, $2, $3) on conflict (profile_id) where used_at is null do update set expires_at = excluded.expires_at, proof_digest = null, verified_at = null, completing_at = null`, identity.ProfileID, identity.OrganizationID, h.now().UTC().Add(recoveryProofLifetime))
				return e
			})
		}
	}
	writeRecoveryAccepted(w)
}
func (h *recoveryHandler) verify(w http.ResponseWriter, r *http.Request) {
	request, ok := decodeRecoveryRequest[recoveryVerifyRequest](w, r)
	if !ok || !validEnrollment(request.EnrollmentNumber) || !sixDigits(request.Code) || h.database == nil || len(h.pepper) < 16 {
		writeRecoveryInvalid(w, r)
		return
	}
	identity, found, err := h.loadRecoveryIdentity(r.Context(), request.EnrollmentNumber)
	if err != nil || !found || !identity.eligible() || h.recoveryLimited(r.Context(), "verify", request.EnrollmentNumber, r) || h.provider == nil || h.provider.Verify(r.Context(), identity.AuthUserID, request.Code) != nil {
		writeRecoveryInvalid(w, r)
		return
	}
	proof, digest, err := randomRecoveryProof(h.pepper)
	if err != nil {
		writeRecoveryInvalid(w, r)
		return
	}
	err = h.withRecoveryFlow(r.Context(), func(tx pgx.Tx) error {
		command, e := tx.Exec(r.Context(), `update app.password_recovery_challenges set proof_digest = $1, verified_at = now() where profile_id = $2 and proof_digest is null and used_at is null and expires_at > now()`, digest, identity.ProfileID)
		if e != nil || command.RowsAffected() != 1 {
			return errLoginDenied
		}
		return nil
	})
	if err != nil {
		writeRecoveryInvalid(w, r)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"proof": proof})
}
func (h *recoveryHandler) complete(w http.ResponseWriter, r *http.Request) {
	request, ok := decodeRecoveryRequest[recoveryCompleteRequest](w, r)
	if !ok || request.Proof == "" || len(request.Proof) > 256 || !isValidPassword(request.Password) || h.database == nil || len(h.pepper) < 16 {
		writeRecoveryInvalid(w, r)
		return
	}
	digest := recoveryDigest(h.pepper, request.Proof)
	var challengeID uuid.UUID
	var profileID uuid.UUID
	var role string
	var mfaEnabled bool
	err := h.withRecoveryFlow(r.Context(), func(tx pgx.Tx) error {
		return tx.QueryRow(r.Context(), `select challenge.id, challenge.profile_id, membership.role, exists (select 1 from app.mfa_factors factor where factor.profile_id = challenge.profile_id and factor.status = 'verified') from app.password_recovery_challenges challenge join app.organization_memberships membership on membership.profile_id = challenge.profile_id and membership.organization_id = challenge.organization_id and membership.status = 'active' where challenge.proof_digest = $1 and challenge.used_at is null and challenge.expires_at > now() limit 1 for update of challenge`, digest).Scan(&challengeID, &profileID, &role, &mfaEnabled)
	})
	if err != nil {
		writeRecoveryInvalid(w, r)
		return
	}
	if (role == "owner" || role == "trainer") && mfaEnabled {
		(&mfaHandler{}).writeMFARequired(w, r)
		return
	}
	claimed := false
	err = h.withRecoveryFlow(r.Context(), func(tx pgx.Tx) error {
		command, e := tx.Exec(r.Context(), `update app.password_recovery_challenges set completing_at = now() where id = $1 and proof_digest = $2 and verified_at is not null and completing_at is null and used_at is null and expires_at > now()`, challengeID, digest)
		if e != nil || command.RowsAffected() != 1 {
			return errLoginDenied
		}
		claimed = true
		return nil
	})
	if err != nil || !claimed {
		writeRecoveryInvalid(w, r)
		return
	}
	if h.provider == nil || h.provider.SetPassword(r.Context(), profileID, request.Password) != nil {
		_ = h.withRecoveryFlow(r.Context(), func(tx pgx.Tx) error {
			_, e := tx.Exec(r.Context(), "update app.password_recovery_challenges set completing_at = null where id = $1 and used_at is null", challengeID)
			return e
		})
		writeRecoveryUnavailable(w, r)
		return
	}
	err = h.withRecoveryFlow(r.Context(), func(tx pgx.Tx) error {
		var recovered uuid.UUID
		return tx.QueryRow(r.Context(), "select app.complete_password_recovery($1, $2)", challengeID, digest).Scan(&recovered)
	})
	if err != nil {
		writeRecoveryUnavailable(w, r)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

type recoveryIdentity struct {
	ProfileID, OrganizationID, AuthUserID       uuid.UUID
	Role                                        string
	ProfileActive, OrganizationOK, MembershipOK bool
}

func (i recoveryIdentity) eligible() bool {
	return i.ProfileID != uuid.Nil && i.OrganizationID != uuid.Nil && i.AuthUserID != uuid.Nil && i.ProfileActive && i.OrganizationOK && i.MembershipOK
}
func (h *recoveryHandler) loadRecoveryIdentity(ctx context.Context, enrollment string) (recoveryIdentity, bool, error) {
	var identity recoveryIdentity
	var suspendedAt *time.Time
	var org, membership string
	err := h.withRecoveryFlow(ctx, func(tx pgx.Tx) error {
		return tx.QueryRow(ctx, `select enrollment.profile_id,enrollment.organization_id,profile.auth_user_id,membership.role,profile.suspended_at,organization.status,membership.status from app.login_enrollments enrollment join app.profiles profile on profile.id=enrollment.profile_id join app.organizations organization on organization.id=enrollment.organization_id join app.organization_memberships membership on membership.profile_id=enrollment.profile_id and membership.organization_id=enrollment.organization_id where enrollment.enrollment_number=$1`, enrollment).Scan(&identity.ProfileID, &identity.OrganizationID, &identity.AuthUserID, &identity.Role, &suspendedAt, &org, &membership)
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return recoveryIdentity{}, false, nil
	}
	if err != nil {
		return recoveryIdentity{}, false, err
	}
	identity.ProfileActive = suspendedAt == nil
	identity.OrganizationOK = org == "active"
	identity.MembershipOK = membership == "active"
	return identity, true, nil
}
func (h *recoveryHandler) withRecoveryFlow(ctx context.Context, action func(pgx.Tx) error) error {
	return h.database.WithTransaction(ctx, func(tx pgx.Tx) error {
		if _, err := tx.Exec(ctx, "select set_config('sysap.recovery_flow', 'true', true)"); err != nil {
			return err
		}
		return action(tx)
	})
}
func (h *recoveryHandler) recoveryLimited(ctx context.Context, kind, enrollment string, r *http.Request) bool {
	fingerprint := recoveryDigest(h.pepper, kind+"|"+normalizedRemoteAddress(r)+"|"+enrollment)
	blocked := false
	_ = h.withRecoveryFlow(ctx, func(tx pgx.Tx) error {
		var until *time.Time
		err := tx.QueryRow(ctx, `insert into app.password_recovery_rate_limits (key_fingerprint,window_started_at,attempt_count,blocked_until,updated_at) values ($1,$2,1,null,$2) on conflict (key_fingerprint) do update set window_started_at=case when app.password_recovery_rate_limits.blocked_until is not null and app.password_recovery_rate_limits.blocked_until>excluded.updated_at then app.password_recovery_rate_limits.window_started_at when app.password_recovery_rate_limits.window_started_at + interval '15 minutes' <= excluded.updated_at then excluded.updated_at else app.password_recovery_rate_limits.window_started_at end,attempt_count=case when app.password_recovery_rate_limits.blocked_until is not null and app.password_recovery_rate_limits.blocked_until>excluded.updated_at then app.password_recovery_rate_limits.attempt_count when app.password_recovery_rate_limits.window_started_at + interval '15 minutes' <= excluded.updated_at then 1 when app.password_recovery_rate_limits.attempt_count>=5 then 5 else app.password_recovery_rate_limits.attempt_count+1 end,blocked_until=case when app.password_recovery_rate_limits.blocked_until is not null and app.password_recovery_rate_limits.blocked_until>excluded.updated_at then app.password_recovery_rate_limits.blocked_until when app.password_recovery_rate_limits.window_started_at + interval '15 minutes' <= excluded.updated_at then null when app.password_recovery_rate_limits.attempt_count>=5 then excluded.updated_at + interval '15 minutes' else null end,updated_at=excluded.updated_at returning blocked_until`, fingerprint, h.now().UTC()).Scan(&until)
		blocked = err != nil || until != nil && until.After(h.now().UTC())
		return err
	})
	return blocked
}
func decodeRecoveryRequest[T any](w http.ResponseWriter, r *http.Request) (T, bool) {
	var request T
	defer r.Body.Close()
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, recoveryMaxBody))
	decoder.DisallowUnknownFields()
	if decoder.Decode(&request) != nil || decoder.Decode(&struct{}{}) != io.EOF {
		return request, false
	}
	return request, true
}
func randomRecoveryProof(pepper []byte) (string, []byte, error) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", nil, err
	}
	proof := base64.RawURLEncoding.EncodeToString(raw)
	return proof, recoveryDigest(pepper, proof), nil
}
func recoveryDigest(pepper []byte, value string) []byte {
	mac := hmac.New(sha256.New, pepper)
	_, _ = mac.Write([]byte(value))
	return mac.Sum(nil)
}
func validEnrollment(value string) bool {
	if len(value) != 10 {
		return false
	}
	for _, character := range value {
		if character < '0' || character > '9' {
			return false
		}
	}
	return true
}
func writeRecoveryAccepted(w http.ResponseWriter) {
	writeJSON(w, http.StatusAccepted, map[string]string{"status": "accepted"})
}
func writeRecoveryInvalid(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusUnauthorized, errorResponse{Error: errorDetail{Code: "invalid_credentials", Message: "invalid credentials", RequestID: httpserver.RequestIDFromContext(r.Context())}})
}
func writeRecoveryUnavailable(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusServiceUnavailable, errorResponse{Error: errorDetail{Code: "service_unavailable", Message: "service is unavailable", RequestID: httpserver.RequestIDFromContext(r.Context())}})
}

func parseProviderSessionAtAAL(accessToken string, expectedSubject uuid.UUID, expectedAAL string) (uuid.UUID, string, error) {
	parts := strings.Split(accessToken, ".")
	if len(parts) != 3 || len(accessToken) > 16*1024 {
		return uuid.Nil, "", errLoginDenied
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil || len(payload) == 0 || len(payload) > 8192 {
		return uuid.Nil, "", errLoginDenied
	}
	var claims struct {
		Subject   string `json:"sub"`
		SessionID string `json:"session_id"`
		AAL       string `json:"aal"`
	}
	if json.Unmarshal(payload, &claims) != nil || claims.Subject != expectedSubject.String() || claims.AAL != expectedAAL {
		return uuid.Nil, "", errLoginDenied
	}
	id, err := uuid.Parse(claims.SessionID)
	if err != nil || id == uuid.Nil {
		return uuid.Nil, "", errLoginDenied
	}
	return id, claims.AAL, nil
}
