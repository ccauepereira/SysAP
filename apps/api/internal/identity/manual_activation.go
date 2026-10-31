package identity

import (
	"crypto/rand"
	"errors"
	"fmt"
	"math/big"
	"net/http"
	"time"

	"github.com/ccauepereira/SysAP/apps/api/internal/platform/auth"
	"github.com/ccauepereira/SysAP/apps/api/internal/platform/authorization"
	"github.com/ccauepereira/SysAP/apps/api/internal/platform/database"
	"github.com/ccauepereira/SysAP/apps/api/internal/platform/httpserver"
	"github.com/ccauepereira/SysAP/apps/api/internal/platform/tenant"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

func (h *activationHandler) issueManualActivation(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	actor, ok := auth.AuthenticatedContextFromContext(ctx)
	if !ok {
		httpserver.WriteAuthenticationRequired(w, ctx)
		return
	}
	organizationID, err := tenant.ParseOrganizationHeader(r.Header)
	if err != nil || organizationID != r.PathValue("organization_id") {
		h.validationFailed(w, r)
		return
	}
	invitationID, err := uuid.Parse(r.PathValue("invitation_id"))
	if err != nil {
		h.validationFailed(w, r)
		return
	}
	if len(h.pepper) < 16 {
		h.unavailable(w, r)
		return
	}
	var code string
	now := h.now().UTC()
	expires := now.Add(10 * time.Minute)
	var enrollment string
	dbIdentity := database.AuthenticatedContext{SubjectID: pgtype.UUID{Bytes: actor.SubjectID, Valid: true}, SessionID: pgtype.UUID{Bytes: actor.SessionID, Valid: true}}
	err = authorization.WithMembershipContext(ctx, h.db, dbIdentity, organizationID, actor.ProfileID.String(), func(tx pgx.Tx, membership authorization.Membership) error {
		if membership.Role != authorization.RoleOwner {
			return errInvitationForbidden
		}
		if err := validateActiveInvitationOrganization(ctx, tx, organizationID); err != nil {
			return err
		}
		if err := RequireAdministrativeAAL2(ctx, tx, actor, string(membership.Role)); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `set local sysap.activation_flow = 'true'`); err != nil {
			return err
		}
		var profileID uuid.UUID
		var invitationExpires time.Time
		if err := tx.QueryRow(ctx, `select p.id, p.enrollment_number, i.expires_at from app.activation_invitations i join app.athlete_profiles p on p.id=i.profile_id where i.id=$1 and i.organization_id=$2 and i.status='pending' and i.expires_at>$3 and p.status='pending_activation' for update of i`, invitationID, organizationID, now).Scan(&profileID, &enrollment, &invitationExpires); err != nil {
			return err
		}
		if invitationExpires.Before(expires) {
			expires = invitationExpires
		}
		// A replacement must not accidentally reuse a still-unexpired earlier code.
		for attempt := 0; attempt < 8; attempt++ {
			value, err := rand.Int(rand.Reader, big.NewInt(1_000_000))
			if err != nil {
				return err
			}
			candidate := fmt.Sprintf("%06d", value.Int64())
			var collision bool
			if err := tx.QueryRow(ctx, `select exists(select 1 from app.activation_challenges where invitation_id=$1 and expires_at>$2 and otp_hmac=$3)`, invitationID, now, h.mac(candidate)).Scan(&collision); err != nil {
				return err
			}
			if !collision {
				code = candidate
				break
			}
		}
		if code == "" {
			return errors.New("unable to generate distinct activation code")
		}
		if _, err := tx.Exec(ctx, `update app.activation_challenges set invalidated_at=$1 where invitation_id=$2 and invalidated_at is null and consumed_at is null`, now, invitationID); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `update app.activation_challenges set proof_consumed_at=$1 where invitation_id=$2 and activation_proof_hmac is not null and proof_consumed_at is null`, now, invitationID); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `update app.activation_proofs set consumed_at=$1 where invitation_id=$2 and consumed_at is null`, now, invitationID); err != nil {
			return err
		}
		_, err := tx.Exec(ctx, `insert into app.activation_challenges(athlete_profile_id,invitation_id,channel,otp_hmac,expires_at,resend_count,resend_window_started_at) values($1,$2,'manual',$3,$4,0,$5)`, profileID, invitationID, h.mac(code), expires, now)
		return err
	})
	if err != nil {
		if errors.Is(err, errMFARequired) {
			writeJSON(w, http.StatusForbidden, errorResponse{Error: errorDetail{Code: "mfa_required", Message: "administrative mfa required", RequestID: httpserver.RequestIDFromContext(ctx)}})
			return
		}
		if errors.Is(err, authorization.ErrUnauthorized) || errors.Is(err, errInvitationForbidden) || errors.Is(err, pgx.ErrNoRows) {
			writeJSON(w, http.StatusForbidden, errorResponse{Error: errorDetail{Code: "access_denied", Message: "access denied", RequestID: httpserver.RequestIDFromContext(ctx)}})
			return
		}
		h.unavailable(w, r)
		return
	}
	writeJSON(w, http.StatusCreated, struct {
		Enrollment string    `json:"enrollment_number"`
		Code       string    `json:"code"`
		ExpiresAt  time.Time `json:"expires_at"`
	}{enrollment, code, expires})
}
