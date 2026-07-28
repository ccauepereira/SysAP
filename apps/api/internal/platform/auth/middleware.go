package auth

import (
	"net/http"
	"strings"

	"github.com/ccauepereira/SysAP/apps/api/internal/platform/httpserver"
)

// Middleware requires a strictly formatted Bearer token and resolves it to a
// currently active local session. It is intentionally not attached to a route
// yet; future protected routes compose it explicitly while health endpoints
// remain public.
func Middleware(verifier TokenVerifier, resolver SessionResolver) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			rawToken, ok := bearerToken(r.Header.Values("Authorization"))
			if !ok || verifier == nil || resolver == nil {
				httpserver.WriteAuthenticationRequired(w, r.Context())
				return
			}

			verified, err := verifier.Verify(r.Context(), rawToken)
			if err != nil {
				httpserver.WriteAuthenticationRequired(w, r.Context())
				return
			}

			authenticated, err := resolver.Resolve(r.Context(), verified, httpserver.RequestIDFromContext(r.Context()))
			if err != nil {
				httpserver.WriteAuthenticationRequired(w, r.Context())
				return
			}

			next.ServeHTTP(w, r.WithContext(contextWithAuthenticatedContext(r.Context(), authenticated)))
		})
	}
}

func bearerToken(values []string) (string, bool) {
	if len(values) != 1 {
		return "", false
	}
	parts := strings.Split(values[0], " ")
	if len(parts) != 2 || parts[0] != "Bearer" || parts[1] == "" || strings.ContainsAny(parts[1], " \t\r\n") {
		return "", false
	}
	return parts[1], true
}
