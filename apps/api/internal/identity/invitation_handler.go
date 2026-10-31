package identity

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strings"
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
	FullName                  string    `json:"full_name,omitempty"`
	DisplayName               string    `json:"display_name,omitempty"`
	PhoneE164                 string    `json:"phone_e164"`
	Email                     string    `json:"email"`
	BirthDate                 string    `json:"birth_date"`
	Locality                  string    `json:"locality"`
	TeamID                    uuid.UUID `json:"team_id"`
	FootballPosition          string    `json:"football_position"`
	EnrollmentDeliveryChannel string    `json:"enrollment_delivery_channel"`
}

type athleteInvitationResponse struct {
	InvitationID     uuid.UUID `json:"invitation_id"`
	ProfileID        uuid.UUID `json:"profile_id"`
	EnrollmentNumber string    `json:"enrollment_number,omitempty"`
	Status           string    `json:"status"`
	ExpiresAt        time.Time `json:"expires_at"`
	DeliveryChannel  string    `json:"delivery_channel"`
	DeliveryStatus   string    `json:"delivery_status"`
}

type invitationHandler struct {
	databasePool *database.Pool
	logger       *slog.Logger
	clock        func() time.Time
	enrollGen    EnrollmentNumberGenerator
	delivery     EnrollmentDeliveryProvider
}

func NewInvitationHandler(db *database.Pool, logger *slog.Logger, clock func() time.Time, enrollGen EnrollmentNumberGenerator) http.Handler {
	return newInvitationHandler(db, logger, clock, enrollGen, noOpEnrollmentDeliveryProvider{})
}

func NewInvitationHandlerWithDelivery(db *database.Pool, logger *slog.Logger, clock func() time.Time, enrollGen EnrollmentNumberGenerator, delivery EnrollmentDeliveryProvider) http.Handler {
	if delivery == nil {
		delivery = unavailableEnrollmentDeliveryProvider{}
	}
	return newInvitationHandler(db, logger, clock, enrollGen, delivery)
}

func newInvitationHandler(db *database.Pool, logger *slog.Logger, clock func() time.Time, enrollGen EnrollmentNumberGenerator, delivery EnrollmentDeliveryProvider) http.Handler {
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
		delivery:     delivery,
	}
}

func fingerprintRequest(req createInvitationRequest) string {
	h := sha256.New()
	h.Write([]byte(req.fullName()))
	h.Write([]byte(req.PhoneE164))
	h.Write([]byte(req.Email))
	h.Write([]byte(req.BirthDate))
	h.Write([]byte(req.Locality))
	h.Write([]byte(req.TeamID.String()))
	h.Write([]byte(req.FootballPosition))
	h.Write([]byte(req.EnrollmentDeliveryChannel))
	return hex.EncodeToString(h.Sum(nil))
}

func (r createInvitationRequest) fullName() string {
	if strings.TrimSpace(r.FullName) != "" {
		return strings.TrimSpace(r.FullName)
	}
	return strings.TrimSpace(r.DisplayName)
}

func normalizePhoneE164(raw string) string {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return ""
	}
	hasPlus := strings.HasPrefix(trimmed, "+")
	var digits strings.Builder
	for _, r := range trimmed {
		if r >= '0' && r <= '9' {
			digits.WriteRune(r)
		} else if r != ' ' && r != '-' && r != '(' && r != ')' && r != '.' && r != '+' {
			return trimmed
		}
	}
	d := digits.String()
	if len(d) < 8 || len(d) > 15 {
		return trimmed
	}
	if hasPlus {
		return "+" + d
	}
	if len(d) == 10 || len(d) == 11 {
		return "+55" + d
	}
	if (len(d) == 12 || len(d) == 13) && strings.HasPrefix(d, "55") {
		return "+" + d
	}
	return trimmed
}

func (h *invitationHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	reqID := httpserver.RequestIDFromContext(ctx)
	w.Header().Set("Cache-Control", "no-store")
	if r.PathValue("invitation_id") != "" {
		h.serveDeliveryRoute(w, r)
		return
	}
	if r.Method == http.MethodGet && strings.HasSuffix(r.URL.Path, "/training-groups") {
		h.listTrainingGroups(w, r)
		return
	}
	if r.Method == http.MethodGet && r.PathValue("organization_id") != "" {
		h.listAthleteInvitations(w, r)
		return
	}

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

	req.PhoneE164 = normalizePhoneE164(req.PhoneE164)
	if req.fullName() == "" || req.PhoneE164 == "" || req.Email == "" ||
		(req.EnrollmentDeliveryChannel != "email" && req.EnrollmentDeliveryChannel != "manual") ||
		req.TeamID == uuid.Nil || req.FootballPosition == "" || req.BirthDate == "" || req.Locality == "" {
		writeJSON(w, http.StatusBadRequest, errorResponse{
			Error: errorDetail{
				Code:      "invalid_request",
				Message:   "missing required fields",
				RequestID: reqID,
			},
		})
		return
	}
	if _, err := time.Parse(time.DateOnly, req.BirthDate); err != nil || validateE164(req.PhoneE164) != nil ||
		(req.FootballPosition != "goalkeeper" && req.FootballPosition != "defender" && req.FootballPosition != "midfielder" && req.FootballPosition != "forward") {
		writeJSON(w, http.StatusUnprocessableEntity, errorResponse{
			Error: errorDetail{Code: "validation_failed", Message: "request validation failed", RequestID: reqID},
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
	var enrollmentNumber, destination string
	var createdInvitation bool

	err = authorization.WithMembershipContext(ctx, h.databasePool, dbIdentity, organizationID, authenticated.ProfileID.String(), func(tx pgx.Tx, membership authorization.Membership) error {
		if membership.Role != "owner" {
			conflictErr = errors.New("forbidden")
			return conflictErr
		}
		if err := validateActiveInvitationOrganization(ctx, tx, organizationID); err != nil {
			conflictErr = err
			return err
		}
		if err := RequireAdministrativeAAL2(ctx, tx, authenticated, string(membership.Role)); err != nil {
			conflictErr = err
			return err
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
				SELECT i.id, i.profile_id, p.enrollment_number,
					CASE WHEN i.status = 'consumed' THEN 'activated'
					     WHEN i.status = 'pending' AND i.expires_at <= $3 THEN 'expired'
					     ELSE 'pending_activation' END,
					i.expires_at, COALESCE(p.enrollment_delivery_channel, 'sms'), i.delivery_status
				FROM app.activation_invitations i
				JOIN app.athlete_profiles p ON i.profile_id = p.id
				WHERE i.id = $1 AND i.organization_id = $2
			`, resourceID, organizationID, now).Scan(
				&response.InvitationID, &response.ProfileID, &enrollmentNumber, &response.Status, &response.ExpiresAt, &response.DeliveryChannel, &response.DeliveryStatus,
			)
			if err != nil {
				internalErr = err
				return err
			}
			if response.DeliveryChannel == "manual" {
				response.EnrollmentNumber = enrollmentNumber
			}
			return nil
		}

		var groupExists bool
		if err := tx.QueryRow(ctx, `select exists(select 1 from app.training_groups where id=$1 and organization_id=$2 and status='active')`, req.TeamID, organizationID).Scan(&groupExists); err != nil || !groupExists {
			conflictErr = errors.New("data_conflict")
			return conflictErr
		}

		// Try inserting new profile & invitation
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

			if _, err := tx.Exec(ctx, "SAVEPOINT sp_athlete_enrollment"); err != nil {
				internalErr = err
				return err
			}

			var existsInLogin bool
			if err := tx.QueryRow(ctx, `SELECT set_config('sysap.login_flow', 'true', true) IS NOT NULL AND EXISTS (SELECT 1 FROM app.login_enrollments WHERE enrollment_number = $1)`, enrollmentNumber).Scan(&existsInLogin); err != nil {
				_, _ = tx.Exec(ctx, "ROLLBACK TO SAVEPOINT sp_athlete_enrollment")
				_, _ = tx.Exec(ctx, "RELEASE SAVEPOINT sp_athlete_enrollment")
				internalErr = err
				return err
			}
			_, _ = tx.Exec(ctx, "SELECT set_config('sysap.login_flow', '', true)")
			if existsInLogin {
				_, _ = tx.Exec(ctx, "ROLLBACK TO SAVEPOINT sp_athlete_enrollment")
				_, _ = tx.Exec(ctx, "RELEASE SAVEPOINT sp_athlete_enrollment")
				continue
			}

			err = tx.QueryRow(ctx, `
			INSERT INTO app.athlete_profiles (organization_id, enrollment_number, display_name, phone_e164, email, birth_date, locality, football_position, enrollment_delivery_channel, status)
				VALUES ($1, $2, $3, $4, $5, $6::date, $7, $8, $9, 'pending_activation')
				RETURNING id
			`, organizationID, enrollmentNumber, req.fullName(), req.PhoneE164, req.Email, req.BirthDate, req.Locality, req.FootballPosition, req.EnrollmentDeliveryChannel).Scan(&profileID)

			if err != nil {
				_, _ = tx.Exec(ctx, "ROLLBACK TO SAVEPOINT sp_athlete_enrollment")
				_, _ = tx.Exec(ctx, "RELEASE SAVEPOINT sp_athlete_enrollment")
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
			_, _ = tx.Exec(ctx, "RELEASE SAVEPOINT sp_athlete_enrollment")
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
		if _, err := tx.Exec(ctx, `insert into app.profiles (id, full_name) values ($1, $2)`, profileID, req.fullName()); err != nil {
			internalErr = err
			return err
		}
		if _, err := tx.Exec(ctx, `insert into app.organization_memberships (organization_id, profile_id, role, status) values ($1, $2, 'athlete', 'pending_activation')`, organizationID, profileID); err != nil {
			internalErr = err
			return err
		}

		if _, err := tx.Exec(ctx, `insert into app.training_group_athletes (organization_id, training_group_id, athlete_profile_id) values ($1,$2,$3)`, organizationID, req.TeamID, profileID); err != nil {
			internalErr = err
			return err
		}

		destination = req.PhoneE164
		if req.EnrollmentDeliveryChannel == "email" {
			destination = req.Email
		}
		if req.EnrollmentDeliveryChannel == "manual" {
			_, err = tx.Exec(ctx, `update app.activation_invitations set delivery_status='not_applicable', delivery_attempt_count=0 where id=$1`, invitationID)
		} else {
			_, err = tx.Exec(ctx, `update app.activation_invitations set delivery_status='sending', delivery_attempt_count=1, delivery_last_attempt_at=$1 where id=$2`, now, invitationID)
		}
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
		if req.EnrollmentDeliveryChannel != "manual" {
			if _, err := tx.Exec(ctx, `insert into app.security_audit_events (organization_id,actor_profile_id,event_type,result,reason_code,resource_type,resource_id,request_id,network_fingerprint,metadata) values ($1,$2,'invitation_delivery_attempted','accepted','attempt_started','activation_invitation',$3,$4,'','{}'::jsonb)`, organizationID, membership.ProfileID, invitationID.String(), reqID); err != nil {
				internalErr = err
				return err
			}
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
			InvitationID:    invitationID,
			ProfileID:       profileID,
			Status:          "pending_activation",
			ExpiresAt:       expiresAt,
			DeliveryChannel: req.EnrollmentDeliveryChannel,
			DeliveryStatus:  "sending",
		}
		if req.EnrollmentDeliveryChannel == "manual" {
			response.DeliveryStatus = "not_applicable"
			response.EnrollmentNumber = enrollmentNumber
		}
		createdInvitation = true

		return nil
	})

	if conflictErr != nil {
		if errors.Is(conflictErr, errMFARequired) {
			writeJSON(w, http.StatusForbidden, errorResponse{
				Error: errorDetail{Code: "mfa_required", Message: "multi-factor authentication is required", RequestID: reqID},
			})
			return
		}
		if conflictErr.Error() == "forbidden" || errors.Is(conflictErr, errInvitationForbidden) {
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
	if !createdInvitation && response.DeliveryStatus == "sending" {
		response.DeliveryStatus = "unknown"
	}
	if createdInvitation && req.EnrollmentDeliveryChannel != "manual" {
		outcome := classifyEnrollmentDelivery(h.delivery.SendEnrollment(ctx, req.EnrollmentDeliveryChannel, destination, enrollmentNumber))
		persistErr := authorization.WithMembershipContext(ctx, h.databasePool, dbIdentity, organizationID, authenticated.ProfileID.String(), func(tx pgx.Tx, membership authorization.Membership) error {
			if membership.Role != "owner" {
				return errors.New("forbidden")
			}
			if err := validateActiveInvitationOrganization(ctx, tx, organizationID); err != nil {
				return err
			}
			if _, err := tx.Exec(ctx, `update app.activation_invitations set delivery_status=$1, updated_at=$2 where id=$3 and organization_id=$4 and status='pending'`, outcome, h.clock().UTC(), response.InvitationID, organizationID); err != nil {
				return err
			}
			_, err := tx.Exec(ctx, `insert into app.security_audit_events (organization_id, actor_profile_id, event_type, result, reason_code, resource_type, resource_id, request_id, network_fingerprint, metadata) values ($1,$2,'invitation_delivery_attempted',$3,'provider_outcome','activation_invitation',$4,$5,'','{}'::jsonb)`, organizationID, membership.ProfileID, outcome, response.InvitationID.String(), reqID)
			return err
		})
		if persistErr != nil {
			// If the result cannot be durably recorded, the committed "sending"
			// marker prevents an automatic retry; expose only the safe state.
			outcome = "unknown"
		}
		response.DeliveryStatus = outcome
	}

	writeJSON(w, http.StatusCreated, response)
}

func validateActiveInvitationOrganization(ctx context.Context, tx pgx.Tx, organizationID string) error {
	var status string
	if err := tx.QueryRow(ctx, `select status from app.organizations where id=$1`, organizationID).Scan(&status); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return errInvitationForbidden
		}
		return err
	}
	if status != "active" {
		return errInvitationForbidden
	}
	return nil
}

type trainingGroupItem struct {
	ID   uuid.UUID `json:"id"`
	Name string    `json:"name"`
}

type trainingGroupListResponse struct {
	Items []trainingGroupItem `json:"items"`
}

func (h *invitationHandler) listTrainingGroups(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	reqID := httpserver.RequestIDFromContext(ctx)

	authenticated, ok := auth.AuthenticatedContextFromContext(ctx)
	if !ok {
		httpserver.WriteAuthenticationRequired(w, ctx)
		return
	}

	pathOrgID := r.PathValue("organization_id")
	organizationID, err := tenant.ParseOrganizationHeader(r.Header)
	if pathOrgID == "" || err != nil || organizationID != pathOrgID {
		writeJSON(w, http.StatusBadRequest, errorResponse{
			Error: errorDetail{
				Code:      "invalid_request",
				Message:   "missing, invalid, or mismatched X-Organization-ID header",
				RequestID: reqID,
			},
		})
		return
	}

	dbIdentity := database.AuthenticatedContext{
		SubjectID: pgtype.UUID{Bytes: authenticated.SubjectID, Valid: true},
		SessionID: pgtype.UUID{Bytes: authenticated.SessionID, Valid: true},
	}

	items := make([]trainingGroupItem, 0)
	var forbiddenErr error
	err = authorization.WithMembershipContext(ctx, h.databasePool, dbIdentity, organizationID, authenticated.ProfileID.String(), func(tx pgx.Tx, membership authorization.Membership) error {
		if membership.Role != "owner" && membership.Role != "trainer" {
			forbiddenErr = errInvitationForbidden
			return forbiddenErr
		}
		if err := validateActiveInvitationOrganization(ctx, tx, organizationID); err != nil {
			forbiddenErr = err
			return err
		}
		rows, err := tx.Query(ctx, `
			SELECT id, name
			FROM app.training_groups
			WHERE organization_id = $1 AND status = 'active'
			ORDER BY name ASC
		`, organizationID)
		if err != nil {
			return err
		}
		defer rows.Close()

		for rows.Next() {
			var item trainingGroupItem
			if err := rows.Scan(&item.ID, &item.Name); err != nil {
				return err
			}
			items = append(items, item)
		}
		return rows.Err()
	})
	if forbiddenErr != nil || errors.Is(err, authorization.ErrUnauthorized) {
		writeJSON(w, http.StatusForbidden, errorResponse{
			Error: errorDetail{Code: "access_denied", Message: "access is denied", RequestID: reqID},
		})
		return
	}
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, errorResponse{
			Error: errorDetail{Code: "internal_error", Message: "an internal error occurred", RequestID: reqID},
		})
		return
	}

	writeJSON(w, http.StatusOK, trainingGroupListResponse{Items: items})
}
