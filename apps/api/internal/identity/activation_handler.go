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
	"net/http"
	"regexp"
	"time"
)

var enrollmentPattern = regexp.MustCompile(`^[0-9]{10}$`)
var codePattern = regexp.MustCompile(`^[0-9]{6}$`)

type LocalOTPProvider interface{ NewCode() (string, error) }
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
	otp    LocalOTPProvider
	now    func() time.Time
}

func NewActivationHandler(db *database.Pool, pepper string) http.Handler {
	return &activationHandler{db: db, pepper: []byte(pepper), otp: localOTPProvider{}, now: time.Now}
}

type activationStartRequest struct {
	EnrollmentNumber string `json:"enrollment_number"`
}
type activationVerifyRequest struct {
	EnrollmentNumber string `json:"enrollment_number"`
	Code             string `json:"code"`
}

func (h *activationHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path == "/v1/activation/start" {
		h.start(w, r)
		return
	}
	h.verify(w, r)
}
func (h *activationHandler) start(w http.ResponseWriter, r *http.Request) {
	var q activationStartRequest
	_ = json.NewDecoder(r.Body).Decode(&q)
	if !enrollmentPattern.MatchString(q.EnrollmentNumber) {
		h.accepted(w, r)
		return
	}
	code, e := h.otp.NewCode()
	if e == nil {
		_ = h.db.WithTransaction(r.Context(), func(tx pgx.Tx) error {
			now := h.now().UTC()
			var pid, iid uuid.UUID
			e := tx.QueryRow(r.Context(), `select p.id,i.id from app.athlete_profiles p join app.activation_invitations i on i.profile_id=p.id where p.enrollment_number=$1 and p.status='pending_activation' and i.status='pending' and i.expires_at>=$2 order by i.created_at desc limit 1`, q.EnrollmentNumber, now).Scan(&pid, &iid)
			if e != nil {
				return nil
			}
			_, _ = tx.Exec(r.Context(), `update app.activation_challenges set invalidated_at=$1 where invitation_id=$2 and invalidated_at is null`, now, iid)
			_, e = tx.Exec(r.Context(), `insert into app.activation_challenges(athlete_profile_id,invitation_id,otp_hmac,expires_at) values($1,$2,$3,$4)`, pid, iid, h.hash(code), now.Add(10*time.Minute))
			return e
		})
	}
	h.accepted(w, r)
}
func (h *activationHandler) verify(w http.ResponseWriter, r *http.Request) {
	var q activationVerifyRequest
	if json.NewDecoder(r.Body).Decode(&q) != nil || !enrollmentPattern.MatchString(q.EnrollmentNumber) || !codePattern.MatchString(q.Code) {
		h.fail(w, r)
		return
	}
	var proof string
	e := h.db.WithTransaction(r.Context(), func(tx pgx.Tx) error {
		now := h.now().UTC()
		var id uuid.UUID
		var hash []byte
		e := tx.QueryRow(r.Context(), `select c.id,c.otp_hmac from app.activation_challenges c join app.athlete_profiles p on p.id=c.athlete_profile_id where p.enrollment_number=$1 and c.invalidated_at is null and c.verified_at is null and c.expires_at>$2 and c.attempt_count<5 for update`, q.EnrollmentNumber, now).Scan(&id, &hash)
		if e != nil {
			return errors.New("invalid")
		}
		if !hmac.Equal(hash, h.hash(q.Code)) {
			_, _ = tx.Exec(r.Context(), `update app.activation_challenges set attempt_count=attempt_count+1 where id=$1`, id)
			return errors.New("invalid")
		}
		var raw [32]byte
		if _, e = rand.Read(raw[:]); e != nil {
			return e
		}
		proof = hex.EncodeToString(raw[:])
		_, e = tx.Exec(r.Context(), `update app.activation_challenges set verified_at=$1,activation_proof_hash=$2,proof_expires_at=$3 where id=$4 and verified_at is null`, now, h.hash(proof), now.Add(10*time.Minute), id)
		return e
	})
	if e != nil {
		h.fail(w, r)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"activation_proof": proof})
}
func (h *activationHandler) hash(v string) []byte {
	m := hmac.New(sha256.New, h.pepper)
	m.Write([]byte(v))
	return m.Sum(nil)
}
func (h *activationHandler) accepted(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusAccepted, map[string]string{"status": "accepted"})
}
func (h *activationHandler) fail(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusUnauthorized, map[string]any{"error": map[string]string{"code": "activation_failed", "message": "activation failed", "request_id": httpserver.RequestIDFromContext(r.Context())}})
}

var _ context.Context
