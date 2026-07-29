package identity

import (
	"context"
	"errors"
	"fmt"

	"github.com/ccauepereira/SysAP/apps/api/internal/platform/email"
)

// EnrollmentDeliveryProvider is the server-side boundary for the controlled
// delivery of a newly generated enrollment number. Implementations must never
// log or persist the enrollment number.
type EnrollmentDeliveryProvider interface {
	SendEnrollment(ctx context.Context, channel, destination, enrollment string) error
}

type unavailableEnrollmentDeliveryProvider struct{}

func (unavailableEnrollmentDeliveryProvider) SendEnrollment(context.Context, string, string, string) error {
	return errors.New("delivery_unavailable")
}

// noOpEnrollmentDeliveryProvider is used only by legacy unit fixtures that
// construct the handler directly. Production wiring uses the unavailable
// provider until an approved adapter is configured.
type noOpEnrollmentDeliveryProvider struct{}

func (noOpEnrollmentDeliveryProvider) SendEnrollment(context.Context, string, string, string) error {
	return nil
}

// EmailEnrollmentDeliveryProvider adapts the generic platform email sender to
// the domain-specific enrollment delivery interface.
type EmailEnrollmentDeliveryProvider struct {
	emailProvider *email.BrevoEmailProvider
}

func NewEmailEnrollmentDeliveryProvider(ep *email.BrevoEmailProvider) *EmailEnrollmentDeliveryProvider {
	return &EmailEnrollmentDeliveryProvider{emailProvider: ep}
}

func (p *EmailEnrollmentDeliveryProvider) SendEnrollment(ctx context.Context, channel, destination, enrollment string) error {
	if channel != "email" {
		return errors.New("unsupported_delivery_channel")
	}

	htmlContent := fmt.Sprintf(`
		<p>Olá,</p>
		<p>Seu cadastro no SysAP foi criado. Sua matrícula é:</p>
		<h2>%s</h2>
		<p>Use esta matrícula para ativar sua conta.</p>
	`, enrollment)

	textContent := fmt.Sprintf("Olá,\n\nSeu cadastro no SysAP foi criado. Sua matrícula é:\n%s\n\nUse esta matrícula para ativar sua conta.", enrollment)

	msg := email.TransactionalEmail{
		To:      destination,
		Subject: "Sua matrícula no SysAP",
		Text:    textContent,
		HTML:    htmlContent,
	}

	return p.emailProvider.Send(ctx, msg)
}
