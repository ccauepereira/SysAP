package identity

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"github.com/ccauepereira/SysAP/apps/api/internal/platform/database"
	"github.com/ccauepereira/SysAP/apps/api/internal/platform/httpserver"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"log/slog"
	"net/http"
	"os"
	"regexp"
	"time"
)

var activationEnrollment = regexp.MustCompile(`^[0-9]{10}$`)
var activationCode = regexp.MustCompile(`^[0-9]{6}$`)

type activationHandler struct {
	db     *database.Pool
	pepper []byte
	otp    OTPProvider
	now    func() time.Time
	admin  SupabaseAuthAdmin
	logger *slog.Logger
}

func newActivationHandler(db *database.Pool, pepper []byte, otp OTPProvider, admin SupabaseAuthAdmin, logger *slog.Logger, now func() time.Time) http.Handler {
	return &activationHandler{db: db, pepper: pepper, otp: otp, admin: admin, logger: logger, now: now}
}

func NewActivationHandler(db *database.Pool, pepper string, logger *slog.Logger) http.Handler {
	otp := OTPProvider(unavailableOTPProvider{})
	if environment := os.Getenv("SYSAP_ENV"); environment == "development" || environment == "test" || environment == "local" {
		otp = localOTPProvider{}
	}
	return newActivationHandler(db, []byte(pepper), otp, NewSupabaseAuthAdmin(), logger, time.Now)
}
func (h *activationHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if len(h.pepper) == 0 {
		h.unavailable(w, r)
		return
	}
	switch r.URL.Path {
	case "/v1/activation/start":
		h.start(w, r)
	case "/v1/activation/verify":
		h.verify(w, r)
	case "/v1/activation/complete":
		h.complete(w, r)
	default:
		h.fail(w, r)
	}
}
func (h *activationHandler) start(w http.ResponseWriter, r *http.Request) {
	var q struct {
		Enrollment string `json:"enrollment_number"`
	}
	_ = json.NewDecoder(r.Body).Decode(&q)
	if !activationEnrollment.MatchString(q.Enrollment) {
		h.accept(w)
		return
	}
	code, e := h.otp.NewCode()
	if e == nil {
		err := h.db.WithTransaction(r.Context(), func(tx pgx.Tx) error {
			now := h.now().UTC()
			var p, i uuid.UUID
			_, _ = tx.Exec(r.Context(), `set local sysap.activation_flow = 'true'`)
			var phone string
			e := tx.QueryRow(r.Context(), `select p.id, i.id, p.phone_e164 from app.athlete_profiles p join app.activation_invitations i on i.profile_id=p.id where p.enrollment_number=$1 and p.status='pending_activation' and i.status='pending' and i.expires_at>$2 limit 1 for update of i`, q.Enrollment, now).Scan(&p, &i, &phone)
			if e != nil {
				return nil
			}
			var resendCount int
			var resendWindow time.Time
			var createdAt time.Time
			var existingID uuid.UUID
			e = tx.QueryRow(r.Context(), `select challenge_id, resend_count, resend_window_started_at, created_at from app.activation_challenges where invitation_id=$1 and invalidated_at is null and consumed_at is null`, i).Scan(&existingID, &resendCount, &resendWindow, &createdAt)
			if e == nil {
				if now.Sub(createdAt) < 60*time.Second {
					return nil // 60s minimum interval
				}
				if now.Sub(resendWindow) > 24*time.Hour {
					resendCount = 0
					resendWindow = now
				} else if resendCount >= 3 {
					return nil // 4th resend blocked within 24h
				} else {
					resendCount++
				}
				_, _ = tx.Exec(r.Context(), `update app.activation_challenges set invalidated_at=$1 where challenge_id=$2`, now, existingID)
			} else {
				resendCount = 0
				resendWindow = now
			}
			if h.otp.IsExternal() {
				if err := validateE164(phone); err != nil {
					return errInvalidActivationPhone
				}
				if err := h.otp.Start(r.Context(), phone); err != nil {
					return err
				}
			}
			_, e = tx.Exec(r.Context(), `insert into app.activation_challenges(athlete_profile_id, invitation_id, otp_hmac, expires_at, resend_count, resend_window_started_at) values($1,$2,$3,$4,$5,$6)`, p, i, h.mac(code), now.Add(10*time.Minute), resendCount, resendWindow)
			return e
		})
		if err != nil {
			if err.Error() == "provider_error" {
				h.unavailable(w, r)
				return
			}
			if errors.Is(err, errInvalidActivationPhone) {
				h.fail(w, r)
				return
			}
		}
	}
	h.accept(w)
}
func (h *activationHandler) verify(w http.ResponseWriter, r *http.Request) {
	var q struct {
		Enrollment string `json:"enrollment_number"`
		Code       string `json:"code"`
	}
	if json.NewDecoder(r.Body).Decode(&q) != nil || !activationEnrollment.MatchString(q.Enrollment) || !activationCode.MatchString(q.Code) {
		h.fail(w, r)
		return
	}
	var proof string
	var hmacFailed bool
	e := h.db.WithTransaction(r.Context(), func(tx pgx.Tx) error {
		now := h.now().UTC()
		var id uuid.UUID
		var phone string
		var mac []byte
		_, _ = tx.Exec(r.Context(), `set local sysap.activation_flow = 'true'`)
		e := tx.QueryRow(r.Context(), `select c.challenge_id,c.otp_hmac,p.phone_e164 from app.activation_challenges c join app.athlete_profiles p on p.id=c.athlete_profile_id where p.enrollment_number=$1 and c.invalidated_at is null and c.consumed_at is null and c.expires_at>$2 and c.attempt_count<5 for update of c`, q.Enrollment, now).Scan(&id, &mac, &phone)
		if e != nil {
			return errors.New("failed")
		}

		if h.otp.IsExternal() {
			if err := validateE164(phone); err != nil {
				return errInvalidActivationPhone
			}
			if err := h.otp.Verify(r.Context(), phone, q.Code); err != nil {
				if err.Error() == "provider_error" {
					return err
				}
				hmacFailed = true
			}
		} else {
			if !hmac.Equal(mac, h.mac(q.Code)) {
				hmacFailed = true
			}
		}

		if hmacFailed {
			_, _ = tx.Exec(r.Context(), `update app.activation_challenges set attempt_count=attempt_count+1 where challenge_id=$1`, id)
			return nil
		}
		var b [24]byte
		if _, e = rand.Read(b[:]); e != nil {
			return e
		}
		proof = hex.EncodeToString(b[:])
		_, e = tx.Exec(r.Context(), `update app.activation_challenges set consumed_at=$1,activation_proof_hmac=$2,proof_expires_at=$3 where challenge_id=$4`, now, h.mac(proof), now.Add(10*time.Minute), id)
		return e
	})
	if e != nil {
		if e.Error() == "provider_error" {
			h.unavailable(w, r)
			return
		}
		if errors.Is(e, errInvalidActivationPhone) {
			h.fail(w, r)
			return
		}
	}
	if e != nil || hmacFailed {
		h.fail(w, r)
		return
	}
	writeJSON(w, http.StatusOK, activationVerifyResponse{ActivationProof: proof})
}

func (h *activationHandler) complete(w http.ResponseWriter, r *http.Request) {
	var q struct {
		Proof    string `json:"activation_proof"`
		Password string `json:"password"`
	}
	if json.NewDecoder(r.Body).Decode(&q) != nil || !isValidPassword(q.Password) {
		h.validationFailed(w, r)
		return
	}
	proofBytes, err := hex.DecodeString(q.Proof)
	if err != nil || len(proofBytes) != 24 {
		h.fail(w, r)
		return
	}

	type result struct {
		ProfileID      uuid.UUID `json:"profile_id"`
		Enrollment     string    `json:"enrollment_number"`
		Role           string    `json:"role"`
		OrganizationID uuid.UUID `json:"organization_id"`
	}
	var res result

	var authID uuid.UUID
	e := h.db.WithTransaction(r.Context(), func(tx pgx.Tx) error {
		now := h.now().UTC()
		_, _ = tx.Exec(r.Context(), `set local sysap.activation_flow = 'true'`)

		var challengeID, profileID, invID, orgID uuid.UUID
		var phone, enrollment, name string

		e := tx.QueryRow(r.Context(), `
			select c.challenge_id, c.athlete_profile_id, c.invitation_id, p.phone_e164, p.enrollment_number, p.display_name, i.organization_id
			from app.activation_challenges c
			join app.athlete_profiles p on p.id = c.athlete_profile_id
			join app.activation_invitations i on i.id = c.invitation_id
			where c.activation_proof_hmac = $1
			  and c.proof_expires_at > $2
			  and c.proof_consumed_at is null
			  and p.status = 'pending_activation'
			for update of c`, h.mac(q.Proof), now).Scan(&challengeID, &profileID, &invID, &phone, &enrollment, &name, &orgID)
		if e != nil {
			return errors.New("invalid_proof")
		}

		var err error
		authID, err = h.admin.CreateUser(r.Context(), phone, q.Password)
		if err != nil {
			return errors.New("auth_failed")
		}

		_, e = tx.Exec(r.Context(), `
			insert into app.profiles (id, auth_user_id, full_name, created_at, updated_at)
			values ($1, $2, $3, $4, $4)`, profileID, authID, name, now)
		if e != nil {
			return errors.New("profile_failed")
		}

		_, e = tx.Exec(r.Context(), `
			insert into app.organization_memberships (organization_id, profile_id, role, status, activated_at, created_at, updated_at)
			values ($1, $2, 'athlete', 'active', $3, $3, $3)`, orgID, profileID, now)
		if e != nil {
			return errors.New("membership_failed")
		}

		_, e = tx.Exec(r.Context(), `update app.athlete_profiles set status='activated', updated_at=$1 where id=$2`, now, profileID)
		if e != nil {
			return errors.New("athlete_failed")
		}

		_, e = tx.Exec(r.Context(), `update app.activation_invitations set status='consumed', updated_at=$1 where id=$2`, now, invID)
		if e != nil {
			return errors.New("invitation_failed")
		}

		_, e = tx.Exec(r.Context(), `update app.activation_challenges set proof_consumed_at=$1 where challenge_id=$2`, now, challengeID)
		if e != nil {
			return errors.New("challenge_failed")
		}

		res = result{
			ProfileID:      profileID,
			Enrollment:     enrollment,
			Role:           "athlete",
			OrganizationID: orgID,
		}
		return nil
	})

	if e != nil {
		if authID != uuid.Nil {
			delErr := h.admin.DeleteUser(context.Background(), authID)
			if delErr != nil {
				if repairErr := h.recordRepair(r.Context(), authID); repairErr != nil {
					h.safeLogger().Error("activation_repair_record_unavailable", slog.String("request_id", httpserver.RequestIDFromContext(r.Context())))
				} else {
					h.safeLogger().Error("activation_auth_compensation_pending", slog.String("request_id", httpserver.RequestIDFromContext(r.Context())), slog.String("auth_user_id", authID.String()))
				}
			}
		}

		if e.Error() == "invalid_proof" {
			h.fail(w, r)
			return
		}
		if e.Error() == "auth_failed" {
			h.unavailable(w, r)
			return
		}
		writeJSON(w, http.StatusConflict, errorResponse{
			Error: errorDetail{
				Code:      "resource_conflict",
				Message:   "Failed to complete activation",
				RequestID: httpserver.RequestIDFromContext(r.Context()),
			},
		})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(res)
}

func (h *activationHandler) mac(s string) []byte {
	m := hmac.New(sha256.New, h.pepper)
	m.Write([]byte(s))
	return m.Sum(nil)
}

var errInvalidActivationPhone = errors.New("invalid activation phone")

func (h *activationHandler) recordRepair(ctx context.Context, authUserID uuid.UUID) error {
	return h.db.WithTransaction(ctx, func(tx pgx.Tx) error {
		if _, err := tx.Exec(ctx, `set local sysap.activation_flow = 'true'`); err != nil {
			return err
		}
		_, err := tx.Exec(ctx, `
			insert into app.identity_repair_tasks (auth_user_id, operation, status)
			values ($1, 'delete_auth_user', 'pending')
			on conflict (auth_user_id, operation) where status = 'pending' do nothing`, authUserID)
		return err
	})
}

func (h *activationHandler) safeLogger() *slog.Logger {
	if h.logger != nil {
		return h.logger
	}
	return slog.Default()
}
func validateE164(phone string) error {
	if len(phone) < 2 || len(phone) > 16 {
		return errors.New("invalid_phone")
	}
	if phone[0] != '+' {
		return errors.New("invalid_phone")
	}
	for _, c := range phone[1:] {
		if c < '0' || c > '9' {
			return errors.New("invalid_phone")
		}
	}
	return nil
}

type activationStartResponse struct {
	Status string `json:"status"`
}
type activationVerifyResponse struct {
	ActivationProof string `json:"activation_proof"`
}

func (h *activationHandler) accept(w http.ResponseWriter) {
	writeJSON(w, http.StatusAccepted, activationStartResponse{Status: "accepted"})
}
func (h *activationHandler) validationFailed(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusUnprocessableEntity, errorResponse{
		Error: errorDetail{
			Code:      "validation_failed",
			Message:   "The request violates business or format rules",
			RequestID: httpserver.RequestIDFromContext(r.Context()),
		},
	})
}

func (h *activationHandler) fail(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusUnauthorized, errorResponse{
		Error: errorDetail{
			Code:      "activation_failed",
			Message:   "activation failed",
			RequestID: httpserver.RequestIDFromContext(r.Context()),
		},
	})
}
func (h *activationHandler) unavailable(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusServiceUnavailable, errorResponse{
		Error: errorDetail{
			Code:      "verification_unavailable",
			Message:   "verification is unavailable",
			RequestID: httpserver.RequestIDFromContext(r.Context()),
		},
	})
}
