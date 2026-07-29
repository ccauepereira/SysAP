package identity

import (
	"context"
	"crypto/rand"
	"errors"
)

type OTPProvider interface {
	IsExternal() bool
	NewCode() (string, error)
	Start(ctx context.Context, phone string) error
	Verify(ctx context.Context, phone, code string) error
}

// ChannelOTPProvider is optional for adapters that distinguish SMS and email.
// Legacy providers continue to work through the OTPProvider methods.
type ChannelOTPProvider interface {
	StartChannel(ctx context.Context, channel, destination string) error
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
func (unavailableOTPProvider) Start(context.Context, string) error {
	return errors.New("provider_error")
}
func (unavailableOTPProvider) Verify(context.Context, string, string) error {
	return errors.New("provider_error")
}

func (p localOTPProvider) IsExternal() bool                                     { return false }
func (p localOTPProvider) Start(ctx context.Context, phone string) error        { return nil }
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

func (p localOTPProvider) StartChannel(ctx context.Context, channel, destination string) error {
	return p.Start(ctx, destination)
}
func (p localOTPProvider) VerifyChannel(ctx context.Context, channel, destination, code string) error {
	return p.Verify(ctx, destination, code)
}

func startChannelOTP(ctx context.Context, provider OTPProvider, channel, destination string) error {
	if p, ok := provider.(ChannelOTPProvider); ok {
		return p.StartChannel(ctx, channel, destination)
	}
	return provider.Start(ctx, destination)
}

func verifyChannelOTP(ctx context.Context, provider OTPProvider, channel, destination, code string) error {
	if p, ok := provider.(ChannelOTPProvider); ok {
		return p.VerifyChannel(ctx, channel, destination, code)
	}
	return provider.Verify(ctx, destination, code)
}
