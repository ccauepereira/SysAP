package identity

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestNewActivationChallengeInvalidatesPreviousProof(t *testing.T) {
	pool, teardown, enrollment, _, _ := setupActivationIntegrationTest(t)
	defer teardown()
	now := time.Now()
	provider := &mockOTPProvider{code: "123456"}
	handler := newActivationHandler(pool, []byte("test-pepper"), provider, &mockAuthAdmin{}, slog.Default(), func() time.Time { return now })
	post := func(path string, payload map[string]string) *httptest.ResponseRecorder {
		body, err := json.Marshal(payload)
		if err != nil {
			t.Fatal(err)
		}
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, httptest.NewRequest(http.MethodPost, path, bytes.NewReader(body)))
		return response
	}
	start := map[string]string{"enrollment_number": enrollment}
	if got := post("/v1/activation/start", start).Code; got != http.StatusAccepted {
		t.Fatalf("start status = %d", got)
	}
	verified := post("/v1/activation/verify", map[string]string{"enrollment_number": enrollment, "code": "123456"})
	if verified.Code != http.StatusOK {
		t.Fatalf("verify status = %d", verified.Code)
	}
	var response struct {
		Proof string `json:"activation_proof"`
	}
	if err := json.Unmarshal(verified.Body.Bytes(), &response); err != nil || response.Proof == "" {
		t.Fatal("missing activation proof")
	}
	now = now.Add(61 * time.Second)
	if got := post("/v1/activation/start", start).Code; got != http.StatusAccepted {
		t.Fatalf("resend status = %d", got)
	}
	completed := post("/v1/activation/complete", map[string]string{"activation_proof": response.Proof, "password": "TestPassword1234!!"})
	if completed.Code != http.StatusUnauthorized {
		t.Fatalf("old proof status = %d, want 401", completed.Code)
	}
}
