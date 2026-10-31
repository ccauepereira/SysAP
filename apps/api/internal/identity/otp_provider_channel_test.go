package identity

import (
	"bytes"
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/ccauepereira/SysAP/apps/api/internal/platform/email"
)

type channelRecordingSender struct{ calls int }

func (s *channelRecordingSender) Send(_ context.Context, _ email.TransactionalEmail) error {
	s.calls++
	return nil
}

func TestUnavailableSMSCannotStart(t *testing.T) {
	sender := &channelRecordingSender{}
	handler := newActivationHandler(nil, []byte("test-pepper"), NewEmailOTPProvider(sender), nil, slog.Default(), time.Now)
	r := httptest.NewRecorder()
	handler.ServeHTTP(r, httptest.NewRequest(http.MethodPost, "/v1/activation/start", bytes.NewBufferString(`{"enrollment_number":"2026000001","channel":"sms"}`)))
	if r.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503", r.Code)
	}
	if sender.calls != 0 {
		t.Fatal("unavailable SMS reached email provider")
	}
}

func TestEmailOTPProviderRejectsSMS(t *testing.T) {
	sender := &channelRecordingSender{}
	provider := NewEmailOTPProvider(sender)
	if err := provider.StartChannel(context.Background(), "sms", "+15550000000", "123456"); err == nil {
		t.Fatal("email provider must reject SMS")
	}
	if err := provider.VerifyChannel(context.Background(), "sms", "+15550000000", "123456"); err == nil {
		t.Fatal("email provider must not verify SMS")
	}
	if sender.calls != 0 {
		t.Fatal("SMS request reached email provider")
	}
	if err := provider.StartChannel(context.Background(), "email", "user@example.test", "123456"); err != nil {
		t.Fatal(err)
	}
	if sender.calls != 1 {
		t.Fatalf("email send count = %d, want 1", sender.calls)
	}
}
