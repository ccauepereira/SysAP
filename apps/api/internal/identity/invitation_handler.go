package identity

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/ccauepereira/SysAP/apps/api/internal/platform/auth"
	"github.com/ccauepereira/SysAP/apps/api/internal/platform/authorization"
	"github.com/ccauepereira/SysAP/apps/api/internal/platform/database"
	"github.com/ccauepereira/SysAP/apps/api/internal/platform/httpserver"
	"github.com/ccauepereira/SysAP/apps/api/internal/platform/tenant"
)

type createInvitationRequest struct {
	DisplayName string `json:"display_name"`
	PhoneE164   string `json:"phone_e164"`
	Email       string `json:"email,omitempty"`
}

type athleteInvitationResponse struct {
	InvitationID     uuid.UUID `json:"invitation_id"`
	ProfileID        uuid.UUID `json:"profile_id"`
	EnrollmentNumber string    `json:"enrollment_number"`
	Status           string    `json:"status"`
	ExpiresAt        time.Time `json:"expires_at"`
}

type invitationHandler struct {
	databasePool *database.Pool
	logger       *slog.Logger
	clock        func() time.Time
	enrollGen    EnrollmentNumberGenerator
}

func NewInvitationHandler(db *database.Pool, logger *slog.Logger, clock func() time.Time, enrollGen EnrollmentNumberGenerator) http.Handler {
	if clock == nil {
		clock = time.Now
	}
	if enrollGen == nil {
		enrollGen = NewEnrollmentGenerator(clock)
	}
	return &invitationHandler{
		databasePool: db,
		logger:       logger,
		clock:        clock,
		enrollGen:    enrollGen,
	}
}

func fingerprintRequest(req createInvitationRequest) string {
	h := sha256.New()
	h.Write([]byte(req.DisplayName))
	h.Write([]byte(req.PhoneE164))
	h.Write([]byte(req.Email))
	return hex.EncodeToString(h.Sum(nil))
}

func (h *invitationHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	reqID := httpserver.RequestIDFromContext(ctx)

	authenticated, ok := auth.AuthenticatedContextFromContext(ctx)
	if !ok {
		httpserver.WriteAuthenticationRequired(w, ctx)
		return
	}

	pathOrgID := r.PathValue("organization_id")
	if pathOrgID == "" {
		writeJSON(w, http.StatusBadRequest, errorResponse{
			Error: errorDetail{
				Code:      "invalid_request",
				Message:   "missing organization_id in path",
				RequestID: reqID,
			},
		})
		return
	}

	organizationID, err := tenant.ParseOrganizationHeader(r.Header)
	if err != nil || organizationID != pathOrgID {
		writeJSON(w, http.StatusBadRequest, errorResponse{
			Error: errorDetail{
				Code:      "invalid_request",
				Message:   "missing, invalid, or mismatched X-Organization-ID header",
				RequestID: reqID,
			},
		})
		return
	}

	idempotencyKeyStr := r.Header.Get("Idempotency-Key")
	idempotencyKey, err := uuid.Parse(idempotencyKeyStr)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{
			Error: errorDetail{
				Code:      "invalid_request",
				Message:   "missing or invalid Idempotency-Key header",
				RequestID: reqID,
			},
		})
		return
	}

	var req createInvitationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{
			Error: errorDetail{
				Code:      "invalid_request",
				Message:   "invalid request body",
				RequestID: reqID,
			},
		})
		return
	}

	if req.DisplayName == "" || req.PhoneE164 == "" {
		writeJSON(w, http.StatusBadRequest, errorResponse{
			Error: errorDetail{
				Code:      "invalid_request",
				Message:   "missing required fields",
				RequestID: reqID,
			},
		})
		return
	}

	dbIdentity := database.AuthenticatedContext{
		SubjectID: pgtype.UUID{Bytes: authenticated.SubjectID, Valid: true},
		SessionID: pgtype.UUID{Bytes: authenticated.SessionID, Valid: true},
	}

	var response athleteInvitationResponse
	var conflictErr error
	var internalErr error

	err = authorization.WithMembershipContext(ctx, h.databasePool, dbIdentity, organizationID, authenticated.ProfileID.String(), func(tx pgx.Tx, membership authorization.Membership) error {
		if membership.Role != "owner" {
			conflictErr = errors.New("forbidden")
			return conflictErr
		}

		operation := "create_athlete_invitation"
		fingerprint := fingerprintRequest(req)
		now := h.clock().UTC()

		// Check idempotency first
		var existingFingerprint, resourceID string
		err := tx.QueryRow(ctx,
			"SELECT request_fingerprint, resource_id FROM app.idempotency_records WHERE organization_id = $1 AND actor_profile_id = $2 AND operation = $3 AND idempotency_key = $4",
			organizationID, membership.ProfileID, operation, idempotencyKey).Scan(&existingFingerprint, &resourceID)

		if err != nil && err != pgx.ErrNoRows {
			internalErr = err
			return err
		}

		if err == nil {
			// Record found
			if existingFingerprint != fingerprint {
				conflictErr = errors.New("idempotency_conflict")
				return conflictErr
			}
			// Fetch the existing invitation
			err = tx.QueryRow(ctx, `
				SELECT i.id, i.profile_id, p.enrollment_number, i.status, i.expires_at
				FROM app.activation_invitations i
				JOIN app.athlete_profiles p ON i.profile_id = p.id
				WHERE i.id = $1 AND i.organization_id = $2
			`, resourceID, organizationID).Scan(
				&response.InvitationID, &response.ProfileID, &response.EnrollmentNumber, &response.Status, &response.ExpiresAt,
			)
			if err != nil {
				internalErr = err
				return err
			}
			response.Status = "pending_activation"
			return nil
		}

		// Try inserting new profile & invitation
		var enrollmentNumber string
		var profileID uuid.UUID

		// Attempt to insert profile with generated enrollment number
		// We have to retry if enrollment number collides
		for attempt := 0; attempt < 5; attempt++ {
			enroll, err := h.enrollGen.Generate()
			if err != nil {
				internalErr = err
				return err
			}
			enrollmentNumber = enroll

			err = tx.QueryRow(ctx, `
				INSERT INTO app.athlete_profiles (organization_id, enrollment_number, display_name, phone_e164, email, status)
				VALUES ($1, $2, $3, $4, $5, 'pending_activation')
				RETURNING id
			`, organizationID, enrollmentNumber, req.DisplayName, req.PhoneE164, req.Email).Scan(&profileID)

			if err != nil {
				var pgErr *pgconn.PgError
				if errors.As(err, &pgErr) && pgErr.Code == "23505" && pgErr.ConstraintName == "athlete_profiles_enrollment_number_key" {
					continue // Collision, try again
				}
				if errors.As(err, &pgErr) && pgErr.Code == "23505" {
					// Other collision (e.g. email or phone if unique? Although not marked unique in prompt except enrollment)
					conflictErr = errors.New("data_conflict")
					return err
				}
				internalErr = err
				return err
			}
			break
		}

		if profileID == uuid.Nil {
			internalErr = errors.New("failed to generate unique enrollment number")
			return internalErr
		}

		var invitationID uuid.UUID
		expiresAt := now.Add(7 * 24 * time.Hour) // 7 days valid

		err = tx.QueryRow(ctx, `
			INSERT INTO app.activation_invitations (profile_id, organization_id, role, invited_by_profile_id, status, expires_at)
			VALUES ($1, $2, 'athlete', $3, 'pending', $4)
			RETURNING id
		`, profileID, organizationID, membership.ProfileID, expiresAt).Scan(&invitationID)

		if err != nil {
			internalErr = err
			return err
		}

		// Insert Idempotency record
		_, err = tx.Exec(ctx, `
			INSERT INTO app.idempotency_records (organization_id, actor_profile_id, operation, idempotency_key, request_fingerprint, resource_type, resource_id, response_status, expires_at)
			VALUES ($1, $2, $3, $4, $5, 'activation_invitation', $6, 201, $7)
		`, organizationID, membership.ProfileID, operation, idempotencyKey, fingerprint, invitationID.String(), now.Add(24*time.Hour))

		if err != nil {
			var pgErr *pgconn.PgError
			if errors.As(err, &pgErr) && pgErr.Code == "23505" {
				conflictErr = errors.New("idempotency_conflict_concurrent")
				return err
			}
			internalErr = err
			return err
		}

		// Write audit event
		_, err = tx.Exec(ctx, `
			INSERT INTO app.security_audit_events (
				organization_id, actor_profile_id, target_profile_id, event_type,
				result, reason_code, resource_type, resource_id, request_id,
				network_fingerprint, metadata
			)
			VALUES ($1, $2, NULL, 'athlete_provisioned', 'success', 'created', 'athlete_profile', $3, $4, '', '{}'::jsonb)
		`, organizationID, membership.ProfileID, profileID.String(), reqID)

		if err != nil {
			internalErr = err
			return err
		}

		response = athleteInvitationResponse{
			InvitationID:     invitationID,
			ProfileID:        profileID,
			EnrollmentNumber: enrollmentNumber,
			Status:           "pending_activation",
			ExpiresAt:        expiresAt,
		}

		return nil
	})

	if conflictErr != nil {
		if conflictErr.Error() == "forbidden" {
			writeJSON(w, http.StatusForbidden, errorResponse{
				Error: errorDetail{Code: "access_denied", Message: "access denied", RequestID: reqID},
			})
			return
		}
		if conflictErr.Error() == "idempotency_conflict" {
			writeJSON(w, http.StatusConflict, errorResponse{
				Error: errorDetail{Code: "idempotency_conflict", Message: "idempotency key used with different payload", RequestID: reqID},
			})
			return
		}
		if conflictErr.Error() == "idempotency_conflict_concurrent" {
			writeJSON(w, http.StatusConflict, errorResponse{
				Error: errorDetail{Code: "idempotency_conflict", Message: "concurrent request with same key", RequestID: reqID},
			})
			return
		}
		if conflictErr.Error() == "data_conflict" {
			writeJSON(w, http.StatusConflict, errorResponse{
				Error: errorDetail{Code: "data_conflict", Message: "contact data already exists", RequestID: reqID},
			})
			return
		}
	}

	defer func() {
		if internalErr != nil {
			h.logger.Error("failed to create athlete invitation",
				"organization_id", organizationID,
				"request_id", reqID,
				"operation", "create_invitation")
		}
	}()

	if internalErr != nil || err != nil {
		if errors.Is(err, authorization.ErrUnauthorized) {
			writeJSON(w, http.StatusForbidden, errorResponse{
				Error: errorDetail{Code: "access_denied", Message: "access denied", RequestID: reqID},
			})
			return
		}
		writeJSON(w, http.StatusInternalServerError, errorResponse{
			Error: errorDetail{Code: "internal_error", Message: "an internal error occurred", RequestID: reqID},
		})
		return
	}

	writeJSON(w, http.StatusCreated, response)
}
