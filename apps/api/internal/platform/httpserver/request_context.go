package httpserver

import (
	"context"
	"net/http"
)

func contextWithRequestID(r *http.Request, requestID string) context.Context {
	return context.WithValue(r.Context(), requestIDContextKey{}, requestID)
}

func requestIDFromContext(ctx context.Context) string {
	requestID, _ := ctx.Value(requestIDContextKey{}).(string)
	return requestID
}

// RequestIDFromContext exposes only the server-generated correlation ID to
// reusable middleware. Authentication state is intentionally not stored here.
func RequestIDFromContext(ctx context.Context) string {
	return requestIDFromContext(ctx)
}
