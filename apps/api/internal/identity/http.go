package identity

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/ccauepereira/SysAP/apps/api/internal/platform/auth"
	"github.com/ccauepereira/SysAP/apps/api/internal/platform/authorization"
	"github.com/ccauepereira/SysAP/apps/api/internal/platform/database"
	"github.com/ccauepereira/SysAP/apps/api/internal/platform/httpserver"
	"github.com/ccauepereira/SysAP/apps/api/internal/platform/tenant"
)

type currentProfile struct {
	ID          uuid.UUID `json:"id"`
	DisplayName string    `json:"display_name"`
}

type currentMembership struct {
	OrganizationID uuid.UUID `json:"organization_id"`
	Role           string    `json:"role"`
	Status         string    `json:"status"`
}

type currentIdentityResponse struct {
	Profile     currentProfile      `json:"profile"`
	Memberships []currentMembership `json:"memberships"`
}

type errorResponse struct {
	Error errorDetail `json:"error"`
}

type errorDetail struct {
	Code      string `json:"code"`
	Message   string `json:"message"`
	RequestID string `json:"request_id"`
}

func writeJSON(w http.ResponseWriter, status int, response any) {
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(response)
}

type meHandler struct {
	databasePool *database.Pool
	logger       *slog.Logger
}

func NewMeHandler(databasePool *database.Pool, logger *slog.Logger) http.Handler {
	return &meHandler{
		databasePool: databasePool,
		logger:       logger,
	}
}

func (h *meHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	authenticated, ok := auth.AuthenticatedContextFromContext(ctx)
	if !ok {
		httpserver.WriteAuthenticationRequired(w, ctx)
		return
	}

	organizationID := ""
	if r.Header.Get("X-Organization-ID") != "" {
		parsedOrg, err := tenant.ParseOrganizationHeader(r.Header)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, errorResponse{
				Error: errorDetail{
					Code:      "invalid_request",
					Message:   "missing or invalid X-Organization-ID header",
					RequestID: httpserver.RequestIDFromContext(ctx),
				},
			})
			return
		}
		organizationID = parsedOrg
	}

	var response currentIdentityResponse
	response.Memberships = make([]currentMembership, 0)

	dbIdentity := database.AuthenticatedContext{
		SubjectID: pgtype.UUID{Bytes: authenticated.SubjectID, Valid: true},
		SessionID: pgtype.UUID{Bytes: authenticated.SessionID, Valid: true},
	}

	if organizationID != "" {
		var activeMembership authorization.Membership
		var profileName string
		var profileID uuid.UUID

		err := authorization.WithTenantContext(ctx, h.databasePool, dbIdentity, organizationID, func(tx pgx.Tx, membership authorization.Membership) error {
			activeMembership = membership
			return tx.QueryRow(ctx, `
				select id, full_name
				from app.profiles
				where id = $1
			`, authenticated.ProfileID).Scan(&profileID, &profileName)
		})

		if err != nil {
			if errors.Is(err, authorization.ErrUnauthorized) || errors.Is(err, pgx.ErrNoRows) {
				writeJSON(w, http.StatusForbidden, errorResponse{
					Error: errorDetail{
						Code:      "access_denied",
						Message:   "access is denied",
						RequestID: httpserver.RequestIDFromContext(ctx),
					},
				})
				return
			}
			h.logger.ErrorContext(ctx, "failed to query tenant identity", "error", err)
			writeJSON(w, http.StatusServiceUnavailable, errorResponse{
				Error: errorDetail{
					Code:      "identity_verification_unavailable",
					Message:   "identity verification is temporarily unavailable",
					RequestID: httpserver.RequestIDFromContext(ctx),
				},
			})
			return
		}

		response.Profile = currentProfile{
			ID:          profileID,
			DisplayName: profileName,
		}

		orgUUID, _ := uuid.Parse(activeMembership.OrganizationID)
		response.Memberships = []currentMembership{
			{
				OrganizationID: orgUUID,
				Role:           string(activeMembership.Role),
				Status:         activeMembership.Status,
			},
		}

		writeJSON(w, http.StatusOK, response)
		return
	}

	err := h.databasePool.WithAuthenticatedContext(ctx, dbIdentity, func(tx pgx.Tx) error {
		err := tx.QueryRow(ctx, `
			select id, full_name
			from app.profiles
			where id = $1
		`, authenticated.ProfileID).Scan(&response.Profile.ID, &response.Profile.DisplayName)
		if err != nil {
			return err
		}

		rows, err := tx.Query(ctx, `
			select organization_id, role, status
			from app.organization_memberships
			where profile_id = $1
			order by created_at desc
		`, authenticated.ProfileID)
		if err != nil {
			return err
		}
		defer rows.Close()

		for rows.Next() {
			var m currentMembership
			if err := rows.Scan(&m.OrganizationID, &m.Role, &m.Status); err != nil {
				return err
			}
			response.Memberships = append(response.Memberships, m)
		}
		return rows.Err()
	})

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			httpserver.WriteAuthenticationRequired(w, ctx)
			return
		}
		h.logger.ErrorContext(ctx, "failed to query identity", "error", err)
		writeJSON(w, http.StatusServiceUnavailable, errorResponse{
			Error: errorDetail{
				Code:      "identity_verification_unavailable",
				Message:   "identity verification is temporarily unavailable",
				RequestID: httpserver.RequestIDFromContext(ctx),
			},
		})
		return
	}

	writeJSON(w, http.StatusOK, response)
}
