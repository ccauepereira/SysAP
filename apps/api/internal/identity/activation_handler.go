package identity

import (
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
	"net/http"
	"regexp"
	"time"
)

var activationEnrollment = regexp.MustCompile(`^[0-9]{10}$`)
var activationCode = regexp.MustCompile(`^[0-9]{6}$`)

type OTPProvider interface {
	NewCode() (string, error)
}
type localOTPProvider struct{}

func (localOTPProvider) NewCode() (string, error) {
	var b [6]byte
	if _, e := rand.Read(b[:]); e != nil {
		return "", e
	}
	for i := range b {
		b[i] = '0' + b[i]%10
	}
	return string(b[:]), nil
}

type activationHandler struct {
	db     *database.Pool
	pepper []byte
	otp    OTPProvider
	now    func() time.Time
}

func newActivationHandler(db *database.Pool, pepper []byte, otp OTPProvider, now func() time.Time) http.Handler {
	return &activationHandler{db: db, pepper: pepper, otp: otp, now: now}
}

func NewActivationHandler(db *database.Pool, pepper string) http.Handler {
	return newActivationHandler(db, []byte(pepper), localOTPProvider{}, time.Now)
}
func (h *activationHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if len(h.pepper) == 0 {
		h.unavailable(w, r)
		return
	}
	if r.URL.Path == "/v1/activation/start" {
		h.start(w, r)
		return
	}
	h.verify(w, r)
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
		_ = h.db.WithTransaction(r.Context(), func(tx pgx.Tx) error {
			now := h.now().UTC()
			var p, i uuid.UUID
			_, _ = tx.Exec(r.Context(), `set local sysap.activation_flow = 'true'`)
			e := tx.QueryRow(r.Context(), `select p.id, i.id from app.athlete_profiles p join app.activation_invitations i on i.profile_id=p.id where p.enrollment_number=$1 and p.status='pending_activation' and i.status='pending' and i.expires_at>$2 limit 1`, q.Enrollment, now).Scan(&p, &i)
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
			_, e = tx.Exec(r.Context(), `insert into app.activation_challenges(athlete_profile_id, invitation_id, otp_hmac, expires_at, resend_count, resend_window_started_at) values($1,$2,$3,$4,$5,$6)`, p, i, h.mac(code), now.Add(10*time.Minute), resendCount, resendWindow)
			return e
		})
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
		var mac []byte
		_, _ = tx.Exec(r.Context(), `set local sysap.activation_flow = 'true'`)
		e := tx.QueryRow(r.Context(), `select c.challenge_id,c.otp_hmac from app.activation_challenges c join app.athlete_profiles p on p.id=c.athlete_profile_id where p.enrollment_number=$1 and c.invalidated_at is null and c.consumed_at is null and c.expires_at>$2 and c.attempt_count<5 for update of c`, q.Enrollment, now).Scan(&id, &mac)
		if e != nil {
			return errors.New("failed")
		}
		if !hmac.Equal(mac, h.mac(q.Code)) {
			_, _ = tx.Exec(r.Context(), `update app.activation_challenges set attempt_count=attempt_count+1 where challenge_id=$1`, id)
			hmacFailed = true
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
	if e != nil || hmacFailed {
		h.fail(w, r)
		return
	}
	writeJSON(w, http.StatusOK, activationVerifyResponse{ActivationProof: proof})
}
func (h *activationHandler) mac(s string) []byte {
	m := hmac.New(sha256.New, h.pepper)
	m.Write([]byte(s))
	return m.Sum(nil)
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
func (h *activationHandler) fail(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusUnauthorized, errorResponse{
		Error: struct {
			Code      string `json:"code"`
			Message   string `json:"message"`
			RequestID string `json:"request_id"`
		}{
			Code:      "activation_failed",
			Message:   "activation failed",
			RequestID: httpserver.RequestIDFromContext(r.Context()),
		},
	})
}
func (h *activationHandler) unavailable(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusServiceUnavailable, errorResponse{
		Error: struct {
			Code      string `json:"code"`
			Message   string `json:"message"`
			RequestID string `json:"request_id"`
		}{
			Code:      "verification_unavailable",
			Message:   "verification is unavailable",
			RequestID: httpserver.RequestIDFromContext(r.Context()),
		},
	})
}
