package httpserver

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"time"
)

const (
	readHeaderTimeout = 5 * time.Second
	readTimeout       = 10 * time.Second
	writeTimeout      = 10 * time.Second
	idleTimeout       = 60 * time.Second
	maxHeaderBytes    = 1 << 20
)

type DatabaseChecker interface {
	Ping(context.Context) error
}

type handler struct {
	databasePingTimeout time.Duration
	database            DatabaseChecker
	logger              *slog.Logger
}

func New(
	database DatabaseChecker,
	logger *slog.Logger,
	databasePingTimeout time.Duration,
	authMiddleware func(http.Handler) http.Handler,
	meHandler http.Handler,
	invitationHandler http.Handler,
) http.Handler {
	return newHandler(database, logger, databasePingTimeout, authMiddleware, meHandler, invitationHandler, newRequestID)
}

func newHandler(
	database DatabaseChecker,
	logger *slog.Logger,
	databasePingTimeout time.Duration,
	authMiddleware func(http.Handler) http.Handler,
	meHandler http.Handler,
	invitationHandler http.Handler,
	generateRequestID requestIDGenerator,
) http.Handler {
	handler := &handler{
		database:            database,
		logger:              logger,
		databasePingTimeout: databasePingTimeout,
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", handler.health)
	mux.HandleFunc("GET /readyz", handler.readiness)

	if authMiddleware != nil {
		if meHandler != nil {
			mux.Handle("GET /v1/me", authMiddleware(meHandler))
		}
		if invitationHandler != nil {
			mux.Handle("POST /v1/organizations/{organization_id}/athlete-invitations", authMiddleware(invitationHandler))
		}
	} else {
		notAuthFunc := func(w http.ResponseWriter, r *http.Request) {
			WriteAuthenticationRequired(w, r.Context())
		}
		mux.HandleFunc("GET /v1/me", notAuthFunc)
		mux.HandleFunc("POST /v1/organizations/{organization_id}/athlete-invitations", notAuthFunc)
	}

	return withRequestContext(logRequests(mux, logger), generateRequestID)
}

func NewServer(address string, handler http.Handler) *http.Server {
	return &http.Server{
		Addr:              address,
		Handler:           handler,
		ReadHeaderTimeout: readHeaderTimeout,
		ReadTimeout:       readTimeout,
		WriteTimeout:      writeTimeout,
		IdleTimeout:       idleTimeout,
		MaxHeaderBytes:    maxHeaderBytes,
	}
}

func (h *handler) health(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, healthResponse{
		Status:  "ok",
		Service: "sysap-api",
	})
}

func (h *handler) readiness(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), h.databasePingTimeout)
	defer cancel()

	if err := h.database.Ping(ctx); err != nil {
		h.logger.WarnContext(r.Context(), "database readiness check failed",
			"request_id", requestIDFromContext(r.Context()),
			"dependency", "database",
		)
		writeJSON(w, http.StatusServiceUnavailable, errorResponse{
			Error: errorDetail{
				Code:      "service_not_ready",
				Message:   "service is not ready",
				RequestID: requestIDFromContext(r.Context()),
			},
		})
		return
	}

	writeJSON(w, http.StatusOK, readinessResponse{
		Status:  "ready",
		Service: "sysap-api",
		Checks: checksResponse{
			Database: "up",
		},
	})
}

func writeJSON(w http.ResponseWriter, status int, response any) {
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(response)
}

// WriteAuthenticationRequired is the sole reusable representation of an
// authentication failure. Its input is a request context rather than an error
// so callers cannot accidentally disclose verifier or database details.
func WriteAuthenticationRequired(w http.ResponseWriter, ctx context.Context) {
	w.Header().Set("WWW-Authenticate", "Bearer")
	writeJSON(w, http.StatusUnauthorized, errorResponse{
		Error: errorDetail{
			Code:      "authentication_required",
			Message:   "authentication is required",
			RequestID: requestIDFromContext(ctx),
		},
	})
}

type healthResponse struct {
	Status  string `json:"status"`
	Service string `json:"service"`
}

type readinessResponse struct {
	Status  string         `json:"status"`
	Service string         `json:"service"`
	Checks  checksResponse `json:"checks"`
}

type checksResponse struct {
	Database string `json:"database"`
}

type errorResponse struct {
	Error errorDetail `json:"error"`
}

type errorDetail struct {
	Code      string `json:"code"`
	Message   string `json:"message"`
	RequestID string `json:"request_id"`
}
