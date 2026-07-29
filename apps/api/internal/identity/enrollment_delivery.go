package identity

import (
	"context"
	"errors"
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
