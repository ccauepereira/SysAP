package identity

import (
	"errors"
	"net/http"

	"github.com/ccauepereira/SysAP/apps/api/internal/platform/auth"
	"github.com/ccauepereira/SysAP/apps/api/internal/platform/authorization"
	"github.com/ccauepereira/SysAP/apps/api/internal/platform/httpserver"
	"github.com/ccauepereira/SysAP/apps/api/internal/platform/tenant"
	"github.com/jackc/pgx/v5"
)

type athleteListItem struct {
	InvitationID string `json:"invitation_id"`
	AthleteID    string `json:"athlete_id"`
	Name         string `json:"name"`
	Enrollment   string `json:"enrollment_number"`
	Group        string `json:"group"`
	Status       string `json:"status"`
}

func (h *invitationHandler) listAthleteInvitations(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	ctx := r.Context()
	actor, ok := auth.AuthenticatedContextFromContext(ctx)
	if !ok {
		httpserver.WriteAuthenticationRequired(w, ctx)
		return
	}
	org, err := tenant.ParseOrganizationHeader(r.Header)
	if err != nil || org != r.PathValue("organization_id") {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: errorDetail{Code: "invalid_request", Message: "invalid organization", RequestID: httpserver.RequestIDFromContext(ctx)}})
		return
	}
	items := make([]athleteListItem, 0)
	err = authorization.WithMembershipContext(ctx, h.databasePool, databaseIdentity(actor), org, actor.ProfileID.String(), func(tx pgx.Tx, membership authorization.Membership) error {
		if membership.Role != authorization.RoleOwner {
			return errInvitationForbidden
		}
		if err := validateActiveInvitationOrganization(ctx, tx, org); err != nil {
			return err
		}
		if err := RequireAdministrativeAAL2(ctx, tx, actor, string(membership.Role)); err != nil {
			return err
		}
		rows, err := tx.Query(ctx, `select i.id::text,p.id::text,p.display_name,p.enrollment_number,coalesce(g.name,''),case when m.status='suspended' then 'suspended' when m.status='closed' then 'closed' when p.status='activated' then 'activated' when i.status='pending' and i.expires_at<=now() then 'expired' else p.status end from app.activation_invitations i join app.athlete_profiles p on p.id=i.profile_id and p.organization_id=i.organization_id join app.organization_memberships m on m.profile_id=p.id and m.organization_id=i.organization_id and m.role='athlete' left join app.training_group_athletes a on a.athlete_profile_id=p.id and a.organization_id=i.organization_id left join app.training_groups g on g.id=a.training_group_id and g.organization_id=a.organization_id where i.organization_id=$1 order by p.display_name,g.name`, org)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			var item athleteListItem
			if err := rows.Scan(&item.InvitationID, &item.AthleteID, &item.Name, &item.Enrollment, &item.Group, &item.Status); err != nil {
				return err
			}
			items = append(items, item)
		}
		return rows.Err()
	})
	if err != nil {
		if errors.Is(err, authorization.ErrUnauthorized) || errors.Is(err, errInvitationForbidden) || errors.Is(err, errMFARequired) {
			writeJSON(w, http.StatusForbidden, errorResponse{Error: errorDetail{Code: "access_denied", Message: "access denied", RequestID: httpserver.RequestIDFromContext(ctx)}})
			return
		}
		writeJSON(w, http.StatusServiceUnavailable, errorResponse{Error: errorDetail{Code: "identity_verification_unavailable", Message: "identity verification unavailable", RequestID: httpserver.RequestIDFromContext(ctx)}})
		return
	}
	writeJSON(w, http.StatusOK, struct {
		Items []athleteListItem `json:"items"`
	}{items})
}
