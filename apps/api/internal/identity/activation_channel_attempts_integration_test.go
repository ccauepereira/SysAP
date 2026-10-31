package identity

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestSMSWrongCodeAttemptsArePersisted(t *testing.T) {
	pool, cleanup, enrollment, profileID, invitationID := setupActivationIntegrationTest(t)
	defer cleanup()
	admin, err := pgxpool.New(context.Background(), os.Getenv("SYSAP_TEST_DATABASE_URL"))
	if err != nil {
		t.Fatal(err)
	}
	defer admin.Close()
	pepper := []byte("test-channel-pepper")
	mac := hmac.New(sha256.New, pepper)
	_, _ = mac.Write([]byte("654321"))
	_, err = admin.Exec(context.Background(), `insert into app.activation_challenges(athlete_profile_id, invitation_id, channel, otp_hmac, expires_at) values ($1,$2,'sms',$3,$4)`, profileID, invitationID, mac.Sum(nil), time.Now().Add(time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	handler := newActivationHandler(pool, pepper, &mockOTPProvider{}, &mockAuthAdmin{}, slog.Default(), time.Now)
	body, err := json.Marshal(map[string]string{"enrollment_number": enrollment, "code": "111111"})
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 5; i++ {
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/v1/activation/verify-sms", bytes.NewReader(body)))
		if response.Code != http.StatusUnauthorized {
			t.Fatalf("wrong code status = %d", response.Code)
		}
	}
	var count int
	if err := admin.QueryRow(context.Background(), `select attempt_count from app.activation_challenges where invitation_id=$1`, invitationID).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 5 {
		t.Fatalf("attempt count = %d, want 5", count)
	}
	valid, _ := json.Marshal(map[string]string{"enrollment_number": enrollment, "code": "654321"})
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/v1/activation/verify-sms", bytes.NewReader(valid)))
	if response.Code != http.StatusUnauthorized {
		t.Fatalf("locked challenge status = %d", response.Code)
	}
}

func TestActivationChannelsIndependentlyReachCompletion(t *testing.T) {
	for _, channel := range []string{"email", "sms"} {
		t.Run(channel, func(t *testing.T) {
			pool, cleanup, enrollment, _, _ := setupActivationIntegrationTest(t)
			defer cleanup()
			adminPool, err := pgxpool.New(context.Background(), os.Getenv("SYSAP_TEST_DATABASE_URL"))
			if err != nil {
				t.Fatal(err)
			}
			defer adminPool.Close()
			authID := uuid.New()
			_, err = adminPool.Exec(context.Background(), `insert into auth.users (id,aud,role,email,raw_app_meta_data,raw_user_meta_data,created_at,updated_at) values ($1,'authenticated','authenticated',$2,'{}'::jsonb,'{}'::jsonb,now(),now())`, authID, "fake-"+authID.String()+"@example.test")
			if err != nil {
				t.Fatal(err)
			}
			handler := newActivationHandler(pool, []byte("test-pepper"), &mockOTPProvider{code: "654321"}, &mockAuthAdmin{createdID: authID}, slog.Default(), time.Now)
			post := func(path string, payload map[string]string) *httptest.ResponseRecorder {
				body, marshalErr := json.Marshal(payload)
				if marshalErr != nil {
					t.Fatal(marshalErr)
				}
				r := httptest.NewRecorder()
				handler.ServeHTTP(r, httptest.NewRequest(http.MethodPost, path, bytes.NewReader(body)))
				return r
			}
			if got := post("/v1/activation/start", map[string]string{"enrollment_number": enrollment, "channel": channel}).Code; got != http.StatusAccepted {
				t.Fatalf("start = %d", got)
			}
			path := "/v1/activation/verify"
			if channel == "sms" {
				path = "/v1/activation/verify-sms"
			}
			verified := post(path, map[string]string{"enrollment_number": enrollment, "code": "654321"})
			if verified.Code != http.StatusOK {
				t.Fatalf("verify = %d", verified.Code)
			}
			var proof struct {
				Value string `json:"activation_proof"`
			}
			if err := json.Unmarshal(verified.Body.Bytes(), &proof); err != nil || proof.Value == "" {
				t.Fatal("missing proof")
			}
			payload := map[string]string{"activation_proof": proof.Value, "password": "ValidPassword1234!!"}
			if got := post("/v1/activation/complete", payload).Code; got != http.StatusNoContent {
				t.Fatalf("complete = %d", got)
			}
			if got := post("/v1/activation/complete", payload).Code; got != http.StatusUnauthorized {
				t.Fatalf("reuse = %d", got)
			}
		})
	}
}

func TestSwitchChannelInvalidatesPreviousChallenge(t *testing.T) {
	pool, cleanup, enrollment, _, _ := setupActivationIntegrationTest(t)
	defer cleanup()
	now := time.Now().UTC()
	provider := &mockOTPProvider{code: "111111"}
	handler := newActivationHandler(pool, []byte("test-pepper"), provider, &mockAuthAdmin{}, slog.Default(), func() time.Time { return now })
	post := func(path string, payload map[string]string) *httptest.ResponseRecorder {
		body, err := json.Marshal(payload)
		if err != nil {
			t.Fatal(err)
		}
		r := httptest.NewRecorder()
		handler.ServeHTTP(r, httptest.NewRequest(http.MethodPost, path, bytes.NewReader(body)))
		return r
	}
	if got := post("/v1/activation/start", map[string]string{"enrollment_number": enrollment, "channel": "sms"}).Code; got != http.StatusAccepted {
		t.Fatalf("SMS start = %d", got)
	}
	now = now.Add(61 * time.Second)
	provider.code = "222222"
	if got := post("/v1/activation/start", map[string]string{"enrollment_number": enrollment, "channel": "email"}).Code; got != http.StatusAccepted {
		t.Fatalf("email start = %d", got)
	}
	if got := post("/v1/activation/verify-sms", map[string]string{"enrollment_number": enrollment, "code": "111111"}).Code; got != http.StatusUnauthorized {
		t.Fatalf("old SMS status = %d", got)
	}
	if got := post("/v1/activation/verify", map[string]string{"enrollment_number": enrollment, "code": "111111"}).Code; got != http.StatusUnauthorized {
		t.Fatalf("old code status = %d", got)
	}
	if got := post("/v1/activation/verify", map[string]string{"enrollment_number": enrollment, "code": "222222"}).Code; got != http.StatusOK {
		t.Fatalf("new code status = %d", got)
	}
}

func TestActivationChannelsRejectExpiredCodes(t *testing.T) {
	for _, channel := range []string{"email", "sms"} {
		t.Run(channel, func(t *testing.T) {
			pool, cleanup, enrollment, _, _ := setupActivationIntegrationTest(t)
			defer cleanup()
			now := time.Now().UTC()
			handler := newActivationHandler(pool, []byte("test-pepper"), &mockOTPProvider{code: "654321"}, &mockAuthAdmin{}, slog.Default(), func() time.Time { return now })
			body, _ := json.Marshal(map[string]string{"enrollment_number": enrollment, "channel": channel})
			start := httptest.NewRecorder()
			handler.ServeHTTP(start, httptest.NewRequest(http.MethodPost, "/v1/activation/start", bytes.NewReader(body)))
			if start.Code != http.StatusAccepted {
				t.Fatalf("start = %d", start.Code)
			}
			now = now.Add(11 * time.Minute)
			path := "/v1/activation/verify"
			if channel == "sms" {
				path = "/v1/activation/verify-sms"
			}
			body, _ = json.Marshal(map[string]string{"enrollment_number": enrollment, "code": "654321"})
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, httptest.NewRequest(http.MethodPost, path, bytes.NewReader(body)))
			if response.Code != http.StatusUnauthorized {
				t.Fatalf("expired code status = %d", response.Code)
			}
		})
	}
}
