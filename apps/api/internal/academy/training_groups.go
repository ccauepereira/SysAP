// Package academy implements the coach-facing academy workflows.
package academy

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"unicode/utf8"

	"github.com/ccauepereira/SysAP/apps/api/internal/identity"
	"github.com/ccauepereira/SysAP/apps/api/internal/platform/auth"
	"github.com/ccauepereira/SysAP/apps/api/internal/platform/authorization"
	"github.com/ccauepereira/SysAP/apps/api/internal/platform/database"
	"github.com/ccauepereira/SysAP/apps/api/internal/platform/httpserver"
	"github.com/ccauepereira/SysAP/apps/api/internal/platform/tenant"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
)

var errAccessDenied = errors.New("access denied")
var errGroupConflict = errors.New("training group already exists")

type trainingGroup struct {
	ID   uuid.UUID `json:"id"`
	Name string    `json:"name"`
}

type groupService struct{ pool *database.Pool }

func (s groupService) create(ctx context.Context, actor auth.AuthenticatedContext, organizationID, name string) (trainingGroup, error) {
	var group trainingGroup
	dbIdentity := database.AuthenticatedContext{SubjectID: pgtype.UUID{Bytes: actor.SubjectID, Valid: true}, SessionID: pgtype.UUID{Bytes: actor.SessionID, Valid: true}}
	err := authorization.WithMembershipContext(ctx, s.pool, dbIdentity, organizationID, actor.ProfileID.String(), func(tx pgx.Tx, membership authorization.Membership) error {
		if membership.Role != authorization.RoleOwner {
			return errAccessDenied
		}
		var active bool
		if err := tx.QueryRow(ctx, `select status = 'active' from app.organizations where id = $1`, organizationID).Scan(&active); err != nil {
			return err
		}
		if !active {
			return errAccessDenied
		}
		if err := identity.RequireAdministrativeAAL2(ctx, tx, actor, string(membership.Role)); err != nil {
			return errAccessDenied
		}
		err := tx.QueryRow(ctx, `insert into app.training_groups (organization_id, name) values ($1, $2) returning id, name`, organizationID, name).Scan(&group.ID, &group.Name)
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return errGroupConflict
		}
		return err
	})
	return group, err
}

type createTrainingGroupHandler struct{ service groupService }

func NewCreateTrainingGroupHandler(pool *database.Pool) http.Handler {
	return &createTrainingGroupHandler{service: groupService{pool: pool}}
}

func (h *createTrainingGroupHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	actor, ok := auth.AuthenticatedContextFromContext(r.Context())
	if !ok {
		httpserver.WriteAuthenticationRequired(w, r.Context())
		return
	}
	organizationID, err := tenant.ParseOrganizationHeader(r.Header)
	if err != nil || organizationID != r.PathValue("organization_id") {
		writeError(w, r, http.StatusBadRequest, "invalid_request", "invalid organization context")
		return
	}
	var request struct {
		Name string `json:"name"`
	}
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 4096))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&request); err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid_request", "invalid request")
		return
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		writeError(w, r, http.StatusBadRequest, "invalid_request", "invalid request")
		return
	}
	name := strings.TrimSpace(request.Name)
	if name == "" || utf8.RuneCountInString(name) > 120 {
		writeError(w, r, http.StatusBadRequest, "invalid_request", "invalid training group name")
		return
	}
	group, err := h.service.create(r.Context(), actor, organizationID, name)
	switch {
	case errors.Is(err, authorization.ErrUnauthorized), errors.Is(err, errAccessDenied), errors.Is(err, pgx.ErrNoRows):
		writeError(w, r, http.StatusForbidden, "access_denied", "access is denied")
	case errors.Is(err, errGroupConflict):
		writeError(w, r, http.StatusConflict, "resource_conflict", "training group already exists")
	case err != nil:
		writeError(w, r, http.StatusInternalServerError, "internal_error", "an internal error occurred")
	default:
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(group)
	}
}

func writeError(w http.ResponseWriter, r *http.Request, status int, code, message string) {
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]any{"error": map[string]string{"code": code, "message": message, "request_id": httpserver.RequestIDFromContext(r.Context())}})
}
