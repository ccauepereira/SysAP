package tenant

import (
	"errors"
	"net/http"

	"github.com/google/uuid"
)

var ErrInvalidOrganization = errors.New("invalid_request: missing or invalid X-Organization-ID header")

// ParseOrganizationHeader ensures the header is a canonical, non-zero UUID.
// It must be exactly one value. Missing or invalid headers return invalid_request.
func ParseOrganizationHeader(header http.Header) (string, error) {
	values := header.Values("X-Organization-ID")
	if len(values) != 1 {
		return "", ErrInvalidOrganization
	}

	parsed, err := uuid.Parse(values[0])
	if err != nil {
		return "", ErrInvalidOrganization
	}

	if parsed == uuid.Nil {
		return "", ErrInvalidOrganization
	}

	// Canonical format only
	if parsed.String() != values[0] {
		return "", ErrInvalidOrganization
	}

	return parsed.String(), nil
}
