package email

import (
	"context"
	"errors"
	"net/mail"
	"strings"
)

// AllowlistedSender prevents development and staging sends to unapproved recipients.
type AllowlistedSender struct {
	sender  TransactionalEmailSender
	allowed map[string]struct{}
}

func NewAllowlistedSender(sender TransactionalEmailSender, recipients string) (*AllowlistedSender, error) {
	if sender == nil {
		return nil, errors.New("email: sender is required")
	}
	allowed := make(map[string]struct{})
	for _, recipient := range strings.Split(recipients, ",") {
		address, err := canonicalRecipient(recipient)
		if err != nil {
			return nil, errors.New("email: configured recipient allowlist is invalid")
		}
		allowed[address] = struct{}{}
	}
	return &AllowlistedSender{sender: sender, allowed: allowed}, nil
}

func canonicalRecipient(value string) (string, error) {
	value = strings.TrimSpace(value)
	parsed, err := mail.ParseAddress(value)
	if err != nil || parsed.Name != "" || parsed.Address != value {
		return "", errors.New("invalid recipient")
	}
	local, domain, found := strings.Cut(value, "@")
	if !found || local == "" || domain == "" {
		return "", errors.New("invalid recipient")
	}
	return local + "@" + strings.ToLower(domain), nil
}

func (s *AllowlistedSender) Send(ctx context.Context, message TransactionalEmail) error {
	address, err := canonicalRecipient(message.To)
	if err != nil {
		return sendFailure("email: recipient is not allowed", "failed")
	}
	if _, ok := s.allowed[address]; !ok {
		return sendFailure("email: recipient is not allowed", "failed")
	}
	return s.sender.Send(ctx, message)
}
