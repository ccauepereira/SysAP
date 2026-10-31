package identity

import (
	"errors"
	"net/http"

	"github.com/ccauepereira/SysAP/apps/api/internal/platform/auth"
	"github.com/ccauepereira/SysAP/apps/api/internal/platform/authorization"
	"github.com/ccauepereira/SysAP/apps/api/internal/platform/database"
	"github.com/ccauepereira/SysAP/apps/api/internal/platform/httpserver"
	"github.com/ccauepereira/SysAP/apps/api/internal/platform/tenant"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

func (h *meHandler) serveAthleteSelf(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	ctx := r.Context()
	actor, ok := auth.AuthenticatedContextFromContext(ctx)
	if !ok {
		httpserver.WriteAuthenticationRequired(w, ctx)
		return
	}
	org, err := tenant.ParseOrganizationHeader(r.Header)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: errorDetail{Code: "invalid_request", Message: "invalid organization context", RequestID: httpserver.RequestIDFromContext(ctx)}})
		return
	}
	result := struct {
		Name       string   `json:"name"`
		Enrollment string   `json:"enrollment_number"`
		Groups     []string `json:"groups"`
	}{Groups: []string{}}
	identity := database.AuthenticatedContext{SubjectID: pgtype.UUID{Bytes: actor.SubjectID, Valid: true}, SessionID: pgtype.UUID{Bytes: actor.SessionID, Valid: true}}
	err = authorization.WithMembershipContext(ctx, h.databasePool, identity, org, actor.ProfileID.String(), func(tx pgx.Tx, membership authorization.Membership) error {
		if membership.Role != authorization.RoleAthlete {
			return authorization.ErrUnauthorized
		}
		if err := validateActiveInvitationOrganization(ctx, tx, org); err != nil {
			return err
		}
		if err := tx.QueryRow(ctx, `select display_name,enrollment_number from app.athlete_profiles where id=$1 and organization_id=$2 and status='activated'`, actor.ProfileID, org).Scan(&result.Name, &result.Enrollment); err != nil {
			return err
		}
		rows, err := tx.Query(ctx, `select g.name from app.training_group_athletes a join app.training_groups g on g.id=a.training_group_id and g.organization_id=a.organization_id where a.athlete_profile_id=$1 and a.organization_id=$2 and g.status='active' order by g.name`, actor.ProfileID, org)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			var name string
			if err := rows.Scan(&name); err != nil {
				return err
			}
			result.Groups = append(result.Groups, name)
		}
		return rows.Err()
	})
	if err != nil {
		if errors.Is(err, authorization.ErrUnauthorized) || errors.Is(err, errInvitationForbidden) || errors.Is(err, pgx.ErrNoRows) {
			writeJSON(w, http.StatusForbidden, errorResponse{Error: errorDetail{Code: "access_denied", Message: "access denied", RequestID: httpserver.RequestIDFromContext(ctx)}})
			return
		}
		writeJSON(w, http.StatusServiceUnavailable, errorResponse{Error: errorDetail{Code: "identity_verification_unavailable", Message: "identity verification unavailable", RequestID: httpserver.RequestIDFromContext(ctx)}})
		return
	}
	writeJSON(w, http.StatusOK, result)
}
