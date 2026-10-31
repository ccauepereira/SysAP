package identity

import (
	"crypto/hmac"
	"crypto/rand"
	"errors"
	"fmt"
	"math/big"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/ccauepereira/SysAP/apps/api/internal/platform/auth"
	"github.com/ccauepereira/SysAP/apps/api/internal/platform/authorization"
	"github.com/ccauepereira/SysAP/apps/api/internal/platform/httpserver"
	"github.com/ccauepereira/SysAP/apps/api/internal/platform/tenant"
)

// Manual (in-person) password recovery. An owner verifies the athlete's
// identity outside the system and issues a short-lived code; the athlete then
// exchanges enrollment number + code for the same opaque proof used by the
// provider flow, and completes through /v1/auth/password-recovery/complete.
const (
	manualRecoveryCodeLifetime = 15 * time.Minute
	manualRecoveryMaxAttempts  = 5
	manualRecoveryPurpose      = "sysap.manual_password_recovery.v1"
)

var (
	errManualRecoveryInFlight  = errors.New("manual recovery completion in progress")
	errManualRecoveryNotTarget = errors.New("manual recovery target is not eligible")
)

type manualRecoveryIssuedResponse struct {
	EnrollmentNumber string    `json:"enrollment_number"`
	Code             string    `json:"code"`
	ExpiresAt        time.Time `json:"expires_at"`
}

// manualRecoveryDigest binds the code to the recovery purpose and the target
// profile, so it cannot be replayed as an activation code or for another
// athlete even if both use the same pepper.
func manualRecoveryDigest(pepper []byte, profileID uuid.UUID, code string) []byte {
	return recoveryDigest(pepper, manualRecoveryPurpose+"|"+profileID.String()+"|"+code)
}

func newManualRecoveryCode() (string, error) {
	value, err := rand.Int(rand.Reader, big.NewInt(1_000_000))
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%06d", value.Int64()), nil
}

func (h *recoveryHandler) issueManualRecovery(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	ctx := r.Context()
	actor, ok := auth.AuthenticatedContextFromContext(ctx)
	if !ok {
		httpserver.WriteAuthenticationRequired(w, ctx)
		return
	}
	if h.database == nil || len(h.pepper) < 16 {
		writeRecoveryUnavailable(w, r)
		return
	}
	organizationID, err := tenant.ParseOrganizationHeader(r.Header)
	if err != nil || organizationID != r.PathValue("organization_id") {
		writeRecoveryValidationFailed(w, r)
		return
	}
	targetProfileID, err := uuid.Parse(r.PathValue("athlete_profile_id"))
	if err != nil {
		writeRecoveryValidationFailed(w, r)
		return
	}

	now := h.now().UTC()
	response := manualRecoveryIssuedResponse{ExpiresAt: now.Add(manualRecoveryCodeLifetime)}
	err = authorization.WithMembershipContext(ctx, h.database, databaseIdentity(actor), organizationID, actor.ProfileID.String(), func(tx pgx.Tx, membership authorization.Membership) error {
		// Role and MFA come from the database, never from the client.
		if membership.Role != authorization.RoleOwner {
			return errInvitationForbidden
		}
		if err := validateActiveInvitationOrganization(ctx, tx, organizationID); err != nil {
			return err
		}
		if err := RequireAdministrativeAAL2(ctx, tx, actor, string(membership.Role)); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, "select set_config('sysap.recovery_flow', 'true', true)"); err != nil {
			return err
		}
		err := tx.QueryRow(ctx, `
			select enrollment.enrollment_number
			from app.login_enrollments enrollment
			join app.profiles profile on profile.id = enrollment.profile_id
			join app.athlete_profiles athlete on athlete.id = enrollment.profile_id and athlete.organization_id = enrollment.organization_id
			join app.organization_memberships membership
			  on membership.profile_id = enrollment.profile_id and membership.organization_id = enrollment.organization_id
			where enrollment.profile_id = $1 and enrollment.organization_id = $2
			  and membership.role = 'athlete' and membership.status = 'active'
			  and athlete.status = 'activated'
			  and profile.auth_user_id is not null and profile.suspended_at is null`,
			targetProfileID, organizationID).Scan(&response.EnrollmentNumber)
		if errors.Is(err, pgx.ErrNoRows) {
			return errManualRecoveryNotTarget
		}
		if err != nil {
			return err
		}

		// Serialize with an in-flight completion: replacing the challenge while
		// the password is being changed would leave Auth and the database apart.
		var inFlight bool
		var previousDigest []byte
		err = tx.QueryRow(ctx, `
			select completing_at is not null and expires_at > $2, code_digest
			from app.password_recovery_challenges
			where profile_id = $1 and used_at is null
			for update`, targetProfileID, now).Scan(&inFlight, &previousDigest)
		if err != nil && !errors.Is(err, pgx.ErrNoRows) {
			return err
		}
		if inFlight {
			return errManualRecoveryInFlight
		}

		var digest []byte
		for attempt := 0; attempt < 8 && digest == nil; attempt++ {
			code, err := newManualRecoveryCode()
			if err != nil {
				return err
			}
			candidate := manualRecoveryDigest(h.pepper, targetProfileID, code)
			if previousDigest != nil && hmac.Equal(candidate, previousDigest) {
				continue
			}
			response.Code, digest = code, candidate
		}
		if digest == nil {
			return errors.New("unable to generate distinct recovery code")
		}

		// One open challenge per profile (unique partial index). Replacing it
		// resets proof, attempts and verification, invalidating the old code.
		var challengeID uuid.UUID
		err = tx.QueryRow(ctx, `
			insert into app.password_recovery_challenges
			    (profile_id, organization_id, expires_at, channel, code_digest, attempt_count, issued_by_profile_id, issued_at)
			values ($1, $2, $3, 'manual', $4, 0, $5, $6)
			on conflict (profile_id) where used_at is null do update
			set expires_at = excluded.expires_at, channel = 'manual', code_digest = excluded.code_digest,
			    attempt_count = 0, issued_by_profile_id = excluded.issued_by_profile_id, issued_at = excluded.issued_at,
			    proof_digest = null, verified_at = null, completing_at = null
			where app.password_recovery_challenges.organization_id = excluded.organization_id
			returning id`,
			targetProfileID, organizationID, response.ExpiresAt, digest, actor.ProfileID, now).Scan(&challengeID)
		if err != nil {
			return err
		}
		_, err = tx.Exec(ctx, `
			insert into app.security_audit_events (
				organization_id, actor_profile_id, target_profile_id, event_type, result,
				reason_code, resource_type, resource_id, request_id, network_fingerprint, metadata
			) values ($1, $2, $3, 'manual_password_recovery_issued', 'success',
				'identity_verified_in_person', 'password_recovery_challenge', $4, $5, '', '{}'::jsonb)`,
			organizationID, actor.ProfileID, targetProfileID, challengeID.String(), httpserver.RequestIDFromContext(ctx))
		return err
	})
	if err != nil {
		switch {
		case errors.Is(err, authorization.ErrUnauthorized), errors.Is(err, errInvitationForbidden), errors.Is(err, errMFARequired):
			writeJSON(w, http.StatusForbidden, errorResponse{Error: errorDetail{Code: "access_denied", Message: "access denied", RequestID: httpserver.RequestIDFromContext(ctx)}})
		case errors.Is(err, errManualRecoveryNotTarget):
			writeJSON(w, http.StatusNotFound, errorResponse{Error: errorDetail{Code: "not_found", Message: "eligible athlete not found", RequestID: httpserver.RequestIDFromContext(ctx)}})
		case errors.Is(err, errManualRecoveryInFlight):
			writeJSON(w, http.StatusConflict, errorResponse{Error: errorDetail{Code: "resource_conflict", Message: "recovery completion in progress", RequestID: httpserver.RequestIDFromContext(ctx)}})
		default:
			writeRecoveryUnavailable(w, r)
		}
		return
	}
	writeJSON(w, http.StatusCreated, response)
}

// verifyManualRecovery exchanges enrollment number + manual code for a
// one-time recovery proof. Unknown enrollment, wrong code, expired, reused or
// exhausted challenges all produce the same response to avoid enumeration.
func (h *recoveryHandler) verifyManualRecovery(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	request, ok := decodeRecoveryRequest[recoveryVerifyRequest](w, r)
	if !ok || !validEnrollment(request.EnrollmentNumber) || !sixDigits(request.Code) || h.database == nil || len(h.pepper) < 16 {
		writeRecoveryInvalid(w, r)
		return
	}
	proof, proofDigest, err := randomRecoveryProof(h.pepper)
	if err != nil {
		writeRecoveryUnavailable(w, r)
		return
	}
	ctx := r.Context()
	now := h.now().UTC()
	verified := false
	err = h.withRecoveryFlow(ctx, func(tx pgx.Tx) error {
		var challengeID, profileID uuid.UUID
		var codeDigest []byte
		err := tx.QueryRow(ctx, `
			select challenge.id, challenge.profile_id, challenge.code_digest
			from app.password_recovery_challenges challenge
			join app.login_enrollments enrollment
			  on enrollment.profile_id = challenge.profile_id and enrollment.organization_id = challenge.organization_id
			join app.profiles profile on profile.id = challenge.profile_id
			join app.organizations organization on organization.id = challenge.organization_id
			join app.organization_memberships membership
			  on membership.profile_id = challenge.profile_id and membership.organization_id = challenge.organization_id
			where enrollment.enrollment_number = $1
			  and challenge.channel = 'manual' and challenge.used_at is null
			  and challenge.proof_digest is null and challenge.completing_at is null
			  and challenge.expires_at > $2 and challenge.attempt_count < $3
			  and profile.auth_user_id is not null and profile.suspended_at is null
			  and organization.status = 'active' and membership.status = 'active' and membership.role = 'athlete'
			for update of challenge`, request.EnrollmentNumber, now, manualRecoveryMaxAttempts).Scan(&challengeID, &profileID, &codeDigest)
		if errors.Is(err, pgx.ErrNoRows) {
			return nil
		}
		if err != nil {
			return err
		}
		if !hmac.Equal(codeDigest, manualRecoveryDigest(h.pepper, profileID, request.Code)) {
			_, err := tx.Exec(ctx, "update app.password_recovery_challenges set attempt_count = attempt_count + 1 where id = $1", challengeID)
			return err
		}
		command, err := tx.Exec(ctx, `
			update app.password_recovery_challenges
			set proof_digest = $1, verified_at = $2
			where id = $3 and proof_digest is null and used_at is null`, proofDigest, now, challengeID)
		if err != nil {
			return err
		}
		verified = command.RowsAffected() == 1
		return nil
	})
	if err != nil {
		writeRecoveryUnavailable(w, r)
		return
	}
	if !verified {
		writeRecoveryInvalid(w, r)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"proof": proof})
}

func writeRecoveryValidationFailed(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusUnprocessableEntity, errorResponse{Error: errorDetail{Code: "validation_failed", Message: "The request violates business or format rules", RequestID: httpserver.RequestIDFromContext(r.Context())}})
}
