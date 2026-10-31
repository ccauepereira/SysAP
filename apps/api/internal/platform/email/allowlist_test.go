package email

import (
	"context"
	"testing"
)

type recordingSender struct{ calls int }

func (s *recordingSender) Send(_ context.Context, _ TransactionalEmail) error {
	s.calls++
	return nil
}

func TestAllowlistedSenderRejectsUnconfiguredAndUnapprovedRecipients(t *testing.T) {
	provider := &recordingSender{}
	if _, err := NewAllowlistedSender(provider, ""); err == nil {
		t.Fatal("empty allowlist must not enable sending")
	}
	if _, err := NewAllowlistedSender(provider, "Example <user@example.test>"); err == nil {
		t.Fatal("display names are not valid allowlist entries")
	}
	sender, err := NewAllowlistedSender(provider, "allowed@example.test")
	if err != nil {
		t.Fatal(err)
	}
	if err := sender.Send(context.Background(), TransactionalEmail{To: "other@example.test"}); err == nil {
		t.Fatal("unapproved recipient was accepted")
	}
	if provider.calls != 0 {
		t.Fatal("unapproved recipient reached provider")
	}
	if err := sender.Send(context.Background(), TransactionalEmail{To: "allowed@EXAMPLE.TEST"}); err != nil {
		t.Fatal(err)
	}
	if provider.calls != 1 {
		t.Fatalf("approved send count = %d, want 1", provider.calls)
	}
}
