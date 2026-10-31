package identity

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
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

var errInvitationForbidden = errors.New("invitation_forbidden")

func (h *invitationHandler) serveDeliveryRoute(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		h.getDeliveryStatus(w, r)
	case http.MethodPost:
		h.resendInvitation(w, r)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (h *invitationHandler) deliveryContext(w http.ResponseWriter, r *http.Request) (auth.AuthenticatedContext, database.AuthenticatedContext, string, uuid.UUID, bool) {
	ctx := r.Context()
	requestID := httpserver.RequestIDFromContext(ctx)
	identity, ok := auth.AuthenticatedContextFromContext(ctx)
	if !ok {
		httpserver.WriteAuthenticationRequired(w, ctx)
		return auth.AuthenticatedContext{}, database.AuthenticatedContext{}, "", uuid.Nil, false
	}
	orgID, err := tenant.ParseOrganizationHeader(r.Header)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: errorDetail{Code: "invalid_request", Message: "invalid organization context", RequestID: requestID}})
		return auth.AuthenticatedContext{}, database.AuthenticatedContext{}, "", uuid.Nil, false
	}
	invitationID, err := uuid.Parse(r.PathValue("invitation_id"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: errorDetail{Code: "invalid_request", Message: "invalid invitation identifier", RequestID: requestID}})
		return auth.AuthenticatedContext{}, database.AuthenticatedContext{}, "", uuid.Nil, false
	}
	dbIdentity := database.AuthenticatedContext{SubjectID: pgtype.UUID{Bytes: identity.SubjectID, Valid: true}, SessionID: pgtype.UUID{Bytes: identity.SessionID, Valid: true}}
	return identity, dbIdentity, orgID, invitationID, true
}

func (h *invitationHandler) requireInvitationOwner(ctx context.Context, tx pgx.Tx, identity auth.AuthenticatedContext, role string) error {
	if role != "owner" {
		return errInvitationForbidden
	}
	return RequireAdministrativeAAL2(ctx, tx, identity, role)
}

func (h *invitationHandler) getDeliveryStatus(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	ctx := r.Context()
	requestID := httpserver.RequestIDFromContext(ctx)
	identity, dbIdentity, orgID, invitationID, ok := h.deliveryContext(w, r)
	if !ok {
		return
	}
	var response athleteInvitationResponse
	err := authorization.WithMembershipContext(ctx, h.databasePool, dbIdentity, orgID, identity.ProfileID.String(), func(tx pgx.Tx, membership authorization.Membership) error {
		if err := h.requireInvitationOwner(ctx, tx, identity, string(membership.Role)); err != nil {
			return err
		}
		if err := validateActiveInvitationOrganization(ctx, tx, orgID); err != nil {
			return err
		}
		return tx.QueryRow(ctx, `
			select i.id, i.profile_id, case when i.status='pending' and i.expires_at <= $3 then 'expired' when i.status='consumed' then 'activated' else 'pending_activation' end,
			       i.expires_at, ap.enrollment_delivery_channel, i.delivery_status
			from app.activation_invitations i join app.athlete_profiles ap on ap.id=i.profile_id
			where i.id=$1 and i.organization_id=$2
		`, invitationID, orgID, h.clock().UTC()).Scan(&response.InvitationID, &response.ProfileID, &response.Status, &response.ExpiresAt, &response.DeliveryChannel, &response.DeliveryStatus)
	})
	if errors.Is(err, authorization.ErrUnauthorized) || err == pgx.ErrNoRows {
		writeJSON(w, http.StatusNotFound, errorResponse{Error: errorDetail{Code: "resource_not_found", Message: "resource not found", RequestID: requestID}})
		return
	}
	if errors.Is(err, errMFARequired) {
		writeJSON(w, http.StatusForbidden, errorResponse{Error: errorDetail{Code: "mfa_required", Message: "multi-factor authentication is required", RequestID: requestID}})
		return
	}
	if errors.Is(err, errInvitationForbidden) {
		writeJSON(w, http.StatusForbidden, errorResponse{Error: errorDetail{Code: "access_denied", Message: "access denied", RequestID: requestID}})
		return
	}
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, errorResponse{Error: errorDetail{Code: "internal_error", Message: "an internal error occurred", RequestID: requestID}})
		return
	}
	response.DeliveryStatus = normalizeInvitationDeliveryStatus(response.DeliveryStatus)
	w.Header().Set("Cache-Control", "no-store")
	writeJSON(w, http.StatusOK, response)
}

func (h *invitationHandler) resendInvitation(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	ctx := r.Context()
	requestID := httpserver.RequestIDFromContext(ctx)
	identity, dbIdentity, orgID, invitationID, ok := h.deliveryContext(w, r)
	if !ok {
		return
	}
	key, err := uuid.Parse(r.Header.Get("Idempotency-Key"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: errorDetail{Code: "invalid_request", Message: "invalid idempotency key", RequestID: requestID}})
		return
	}

	now := h.clock().UTC()
	const operation = "resend_athlete_invitation"
	fingerprint := sha256.Sum256([]byte(invitationID.String()))
	var response athleteInvitationResponse
	var enrollmentNumber, destination string
	var shouldSend bool
	var appErr string
	err = authorization.WithMembershipContext(ctx, h.databasePool, dbIdentity, orgID, identity.ProfileID.String(), func(tx pgx.Tx, membership authorization.Membership) error {
		if err := h.requireInvitationOwner(ctx, tx, identity, string(membership.Role)); err != nil {
			return err
		}
		if err := validateActiveInvitationOrganization(ctx, tx, orgID); err != nil {
			return err
		}
		// Serialize uses of an idempotency key before reading its record. This
		// covers concurrent replays even when they target different invitation
		// rows, without holding a provider request inside the transaction.
		lockKey := orgID + ":" + membership.ProfileID + ":" + operation + ":" + key.String()
		if _, err := tx.Exec(ctx, `select pg_advisory_xact_lock(hashtextextended($1, 0))`, lockKey); err != nil {
			return err
		}
		var existingFingerprint, resourceID string
		err := tx.QueryRow(ctx, `select request_fingerprint, resource_id from app.idempotency_records where organization_id=$1 and actor_profile_id=$2 and operation=$3 and idempotency_key=$4`, orgID, membership.ProfileID, operation, key).Scan(&existingFingerprint, &resourceID)
		if err == nil {
			if existingFingerprint != fmt.Sprintf("%x", fingerprint[:]) || resourceID != invitationID.String() {
				appErr = "idempotency_conflict"
				return nil
			}
			return tx.QueryRow(ctx, `select i.id,i.profile_id,case when i.status='pending' and i.expires_at <= $3 then 'expired' when i.status='consumed' then 'activated' else 'pending_activation' end,i.expires_at,ap.enrollment_delivery_channel,i.delivery_status from app.activation_invitations i join app.athlete_profiles ap on ap.id=i.profile_id where i.id=$1 and i.organization_id=$2`, invitationID, orgID, now).Scan(&response.InvitationID, &response.ProfileID, &response.Status, &response.ExpiresAt, &response.DeliveryChannel, &response.DeliveryStatus)
		}
		if err != pgx.ErrNoRows {
			return err
		}
		var windowStarted, lastAttempt, expiresAt time.Time
		var resendCount int
		var phone, email, deliveryStatus string
		err = tx.QueryRow(ctx, `select i.id,i.profile_id,i.status,i.expires_at,ap.enrollment_delivery_channel,ap.enrollment_number,ap.phone_e164,ap.email,i.delivery_status,i.delivery_resend_count,i.delivery_window_started_at,i.delivery_last_attempt_at from app.activation_invitations i join app.athlete_profiles ap on ap.id=i.profile_id where i.id=$1 and i.organization_id=$2 for update of i`, invitationID, orgID).Scan(&response.InvitationID, &response.ProfileID, &response.Status, &expiresAt, &response.DeliveryChannel, &enrollmentNumber, &phone, &email, &deliveryStatus, &resendCount, &windowStarted, &lastAttempt)
		if err != nil {
			return err
		}
		if response.Status != "pending" || !expiresAt.After(now) || deliveryStatus != "failed" || response.DeliveryChannel != "email" {
			appErr = "resend_not_allowed"
			return nil
		}
		if now.Sub(lastAttempt) < time.Minute {
			appErr = "resend_rate_limited"
			return nil
		}
		if now.Sub(windowStarted) >= 24*time.Hour {
			resendCount = 0
			windowStarted = now
		}
		if resendCount >= 3 {
			appErr = "resend_rate_limited"
			return nil
		}
		if response.DeliveryChannel == "email" {
			destination = email
		} else {
			destination = phone
		}
		if _, err := tx.Exec(ctx, `update app.activation_invitations set delivery_status='sending',delivery_attempt_count=delivery_attempt_count+1,delivery_resend_count=$1,delivery_window_started_at=$2,delivery_last_attempt_at=$3,updated_at=$3 where id=$4`, resendCount+1, windowStarted, now, invitationID); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `insert into app.security_audit_events (organization_id,actor_profile_id,event_type,result,reason_code,resource_type,resource_id,request_id,network_fingerprint,metadata) values ($1,$2,'invitation_delivery_attempted','accepted','manual_resend','activation_invitation',$3,$4,'','{}'::jsonb)`, orgID, membership.ProfileID, invitationID.String(), requestID); err != nil {
			return err
		}
		_, err = tx.Exec(ctx, `insert into app.idempotency_records (organization_id,actor_profile_id,operation,idempotency_key,request_fingerprint,resource_type,resource_id,response_status,expires_at) values ($1,$2,$3,$4,$5,'activation_invitation',$6,202,$7)`, orgID, membership.ProfileID, operation, key, fmt.Sprintf("%x", fingerprint[:]), invitationID.String(), now.Add(24*time.Hour))
		if err != nil {
			return err
		}
		response.Status = "pending_activation"
		response.ExpiresAt = expiresAt
		response.DeliveryStatus = "sending"
		shouldSend = true
		return nil
	})
	if errors.Is(err, errMFARequired) {
		writeJSON(w, http.StatusForbidden, errorResponse{Error: errorDetail{Code: "mfa_required", Message: "multi-factor authentication is required", RequestID: requestID}})
		return
	}
	if errors.Is(err, authorization.ErrUnauthorized) {
		writeJSON(w, http.StatusForbidden, errorResponse{Error: errorDetail{Code: "access_denied", Message: "access denied", RequestID: requestID}})
		return
	}
	if errors.Is(err, errInvitationForbidden) {
		writeJSON(w, http.StatusForbidden, errorResponse{Error: errorDetail{Code: "access_denied", Message: "access denied", RequestID: requestID}})
		return
	}
	if errors.Is(err, pgx.ErrNoRows) {
		writeJSON(w, http.StatusNotFound, errorResponse{Error: errorDetail{Code: "resource_not_found", Message: "resource not found", RequestID: requestID}})
		return
	}
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, errorResponse{Error: errorDetail{Code: "internal_error", Message: "an internal error occurred", RequestID: requestID}})
		return
	}
	if appErr == "idempotency_conflict" || appErr == "resend_not_allowed" {
		writeJSON(w, http.StatusConflict, errorResponse{Error: errorDetail{Code: appErr, Message: "invitation delivery cannot be retried", RequestID: requestID}})
		return
	}
	if appErr == "resend_rate_limited" {
		writeJSON(w, http.StatusTooManyRequests, errorResponse{Error: errorDetail{Code: appErr, Message: "delivery retry limit reached", RequestID: requestID}})
		return
	}
	if shouldSend {
		outcome := classifyEnrollmentDelivery(h.delivery.SendEnrollment(ctx, response.DeliveryChannel, destination, enrollmentNumber))
		persistErr := authorization.WithMembershipContext(ctx, h.databasePool, dbIdentity, orgID, identity.ProfileID.String(), func(tx pgx.Tx, membership authorization.Membership) error {
			if err := h.requireInvitationOwner(ctx, tx, identity, string(membership.Role)); err != nil {
				return err
			}
			if err := validateActiveInvitationOrganization(ctx, tx, orgID); err != nil {
				return err
			}
			if _, err := tx.Exec(ctx, `update app.activation_invitations set delivery_status=$1,updated_at=$2 where id=$3 and organization_id=$4 and status='pending'`, outcome, h.clock().UTC(), invitationID, orgID); err != nil {
				return err
			}
			_, err := tx.Exec(ctx, `insert into app.security_audit_events (organization_id,actor_profile_id,event_type,result,reason_code,resource_type,resource_id,request_id,network_fingerprint,metadata) values ($1,$2,'invitation_delivery_attempted',$3,'provider_outcome','activation_invitation',$4,$5,'','{}'::jsonb)`, orgID, membership.ProfileID, outcome, invitationID.String(), requestID)
			return err
		})
		if persistErr != nil {
			outcome = "unknown"
		}
		response.DeliveryStatus = outcome
	}
	response.DeliveryStatus = normalizeInvitationDeliveryStatus(response.DeliveryStatus)
	w.Header().Set("Cache-Control", "no-store")
	writeJSON(w, http.StatusAccepted, response)
}

func normalizeInvitationDeliveryStatus(status string) string {
	if status == "sending" {
		return "unknown"
	}
	return status
}

func classifyEnrollmentDelivery(err error) string {
	if err == nil {
		return "sent"
	}
	var classified interface{ DeliveryOutcome() string }
	if errors.As(err, &classified) && classified.DeliveryOutcome() == "failed" {
		return "failed"
	}
	return "unknown"
}
