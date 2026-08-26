package identity

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"

	"github.com/ccauepereira/SysAP/apps/api/internal/platform/email"
)

type OTPProvider interface {
	IsExternal() bool
	NewCode() (string, error)
	Start(ctx context.Context, phone, code string) error
	Verify(ctx context.Context, phone, code string) error
}

// ChannelOTPProvider is optional for adapters that distinguish SMS and email.
// Legacy providers continue to work through the OTPProvider methods.
type ChannelOTPProvider interface {
	StartChannel(ctx context.Context, channel, destination, code string) error
	VerifyChannel(ctx context.Context, channel, destination, code string) error
}

type localOTPProvider struct {
	hook func() (string, error)
}

// unavailableOTPProvider prevents a production process from silently using a
// development-only local code generator. Production delivery is configured in
// Supabase Auth, outside the API and this repository.
type unavailableOTPProvider struct{}

func (unavailableOTPProvider) IsExternal() bool { return true }
func (unavailableOTPProvider) NewCode() (string, error) {
	return localOTPProvider{}.NewCode()
}
func (unavailableOTPProvider) Start(context.Context, string, string) error {
	return errors.New("provider_error")
}
func (unavailableOTPProvider) Verify(context.Context, string, string) error {
	return errors.New("provider_error")
}

func (p localOTPProvider) IsExternal() bool                                     { return false }
func (p localOTPProvider) Start(ctx context.Context, phone, code string) error  { return nil }
func (p localOTPProvider) Verify(ctx context.Context, phone, code string) error { return nil }
func (p localOTPProvider) NewCode() (string, error) {
	if p.hook != nil {
		return p.hook()
	}
	var b [6]byte
	if _, e := rand.Read(b[:]); e != nil {
		return "", e
	}
	for i := range b {
		b[i] = '0' + b[i]%10
	}
	return string(b[:]), nil
}

func (p localOTPProvider) StartChannel(ctx context.Context, channel, destination, code string) error {
	return p.Start(ctx, destination, code)
}
func (p localOTPProvider) VerifyChannel(ctx context.Context, channel, destination, code string) error {
	return p.Verify(ctx, destination, code)
}

func startChannelOTP(ctx context.Context, provider OTPProvider, channel, destination, code string) error {
	if p, ok := provider.(ChannelOTPProvider); ok {
		return p.StartChannel(ctx, channel, destination, code)
	}
	return provider.Start(ctx, destination, code)
}

func verifyChannelOTP(ctx context.Context, provider OTPProvider, channel, destination, code string) error {
	if p, ok := provider.(ChannelOTPProvider); ok {
		return p.VerifyChannel(ctx, channel, destination, code)
	}
	return provider.Verify(ctx, destination, code)
}

// EmailOTPProvider implements OTPProvider for email using the generic email provider
type EmailOTPProvider struct {
	emailProvider *email.BrevoEmailProvider
}

func NewEmailOTPProvider(ep *email.BrevoEmailProvider) *EmailOTPProvider {
	return &EmailOTPProvider{emailProvider: ep}
}

func (p *EmailOTPProvider) IsExternal() bool {
	return false // We want the API to track challenges locally and verify them against the DB
}

func (p *EmailOTPProvider) NewCode() (string, error) {
	var b [6]byte
	if _, e := rand.Read(b[:]); e != nil {
		return "", e
	}
	for i := range b {
		b[i] = '0' + b[i]%10
	}
	return string(b[:]), nil
}

func (p *EmailOTPProvider) Start(ctx context.Context, phone, code string) error {
	return nil // SMS not supported natively by this provider, handled by SMS if any
}

func (p *EmailOTPProvider) Verify(ctx context.Context, phone, code string) error {
	return nil // Handled locally
}

func (p *EmailOTPProvider) StartChannel(ctx context.Context, channel, destination, code string) error {
	if channel != "email" && channel != "sms" {
		return errors.New("unsupported_channel")
	}

	htmlContent := fmt.Sprintf(`
		<p>Olá,</p>
		<p>Seu código de ativação no SysAP é:</p>
		<h2>%s</h2>
		<p>Use este código para concluir a ativação.</p>
	`, code)

	textContent := fmt.Sprintf("Olá,\n\nSeu código de ativação no SysAP é:\n%s\n\nUse este código para concluir a ativação.", code)

	msg := email.TransactionalEmail{
		To:      destination,
		Subject: "Código de Ativação SysAP",
		Text:    textContent,
		HTML:    htmlContent,
	}

	return p.emailProvider.Send(ctx, msg)
}

func (p *EmailOTPProvider) VerifyChannel(ctx context.Context, channel, destination, code string) error {
	return nil // Handled locally by DB verification
}
