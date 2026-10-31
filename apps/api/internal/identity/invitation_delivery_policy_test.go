package identity

import (
	"errors"
	"testing"
)

type testClassifiedDeliveryError struct{ outcome string }

func (e testClassifiedDeliveryError) Error() string           { return "safe provider failure" }
func (e testClassifiedDeliveryError) DeliveryOutcome() string { return e.outcome }

func TestClassifyEnrollmentDelivery(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want string
	}{
		{name: "accepted", want: "sent"},
		{name: "known rejection", err: testClassifiedDeliveryError{outcome: "failed"}, want: "failed"},
		{name: "ambiguous provider result", err: testClassifiedDeliveryError{outcome: "unknown"}, want: "unknown"},
		{name: "unclassified error defaults ambiguous", err: errors.New("private provider detail"), want: "unknown"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := classifyEnrollmentDelivery(test.err); got != test.want {
				t.Fatalf("classifyEnrollmentDelivery() = %q, want %q", got, test.want)
			}
		})
	}
}

func TestNormalizeInvitationDeliveryStatus(t *testing.T) {
	if got := normalizeInvitationDeliveryStatus("sending"); got != "unknown" {
		t.Fatalf("transient delivery state leaked to the API: got %q", got)
	}
	for _, status := range []string{"sent", "failed", "unknown"} {
		if got := normalizeInvitationDeliveryStatus(status); got != status {
			t.Errorf("stable status %q changed to %q", status, got)
		}
	}
}
