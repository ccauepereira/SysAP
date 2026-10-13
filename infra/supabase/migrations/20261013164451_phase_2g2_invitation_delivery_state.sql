begin;

-- Delivery is independent from athlete activation. Existing invitations were
-- committed only after a successful provider response, so their safe baseline
-- is "sent"; all new invitations explicitly begin as "sending".
alter table app.activation_invitations
  add column delivery_status text not null default 'sent',
  add column delivery_attempt_count integer not null default 1,
  add column delivery_resend_count integer not null default 0,
  add column delivery_window_started_at timestamptz,
  add column delivery_last_attempt_at timestamptz;

update app.activation_invitations
set delivery_window_started_at = created_at,
    delivery_last_attempt_at = created_at;

alter table app.activation_invitations
  alter column delivery_window_started_at set default now(),
  alter column delivery_window_started_at set not null,
  alter column delivery_last_attempt_at set default now(),
  alter column delivery_last_attempt_at set not null;

alter table app.activation_invitations
  add constraint activation_invitations_delivery_status_check
    check (delivery_status in ('sending', 'sent', 'failed', 'unknown')),
  add constraint activation_invitations_delivery_attempt_count_check
    check (delivery_attempt_count >= 0),
  add constraint activation_invitations_delivery_resend_count_check
    check (delivery_resend_count between 0 and 3);

-- Bring already-persisted pending invitations into the same account lifecycle.
insert into app.profiles (id, full_name)
select athlete.id, athlete.display_name
from app.athlete_profiles athlete
join app.activation_invitations invitation on invitation.profile_id = athlete.id
where athlete.status = 'pending_activation'
  and invitation.status = 'pending'
on conflict (id) do nothing;

insert into app.organization_memberships (organization_id, profile_id, role, status)
select invitation.organization_id, invitation.profile_id, 'athlete', 'pending_activation'
from app.activation_invitations invitation
join app.athlete_profiles athlete on athlete.id = invitation.profile_id
where athlete.status = 'pending_activation'
  and invitation.status = 'pending'
on conflict (organization_id, profile_id) do nothing;

alter table app.profiles enable row level security;
alter table app.profiles force row level security;

-- New invitations reserve the profile identity and membership before delivery.
-- A pending row is invisible to normal session resolution because it has no
-- auth_user_id and its membership is not active.
create policy insert_pending_athlete_profile_for_invitation on app.profiles
  for insert to sysap_api
  with check (
    auth_user_id is null
    and exists (
      select 1 from app.athlete_profiles athlete
      join app.activation_invitations invitation on invitation.profile_id = athlete.id
      where athlete.id = profiles.id
        and athlete.status = 'pending_activation'
        and invitation.status = 'pending'
        and invitation.organization_id = app.current_tenant_id()
    )
  );

create policy select_pending_athlete_profile_for_invitation on app.profiles
  for select to sysap_api
  using (
    exists (
      select 1 from app.athlete_profiles athlete
      join app.activation_invitations invitation on invitation.profile_id = athlete.id
      where athlete.id = profiles.id
        and athlete.status = 'pending_activation'
        and invitation.status = 'pending'
        and invitation.organization_id = app.current_tenant_id()
    )
  );

-- This must be restrictive: the pre-existing permissive isolate_tenant
-- policy otherwise ORs with a permissive invitation policy and allows any
-- membership insert in the current tenant.
create policy insert_membership_only_for_invitation_or_activation on app.organization_memberships
  as restrictive for insert to sysap_api
  with check (
    organization_id = app.current_tenant_id()
    and role = 'athlete'
    and (
      (
        status = 'pending_activation'
        and exists (
          select 1 from app.activation_invitations invitation
          join app.athlete_profiles athlete on athlete.id = invitation.profile_id
          where invitation.profile_id = organization_memberships.profile_id
            and invitation.organization_id = organization_memberships.organization_id
            and invitation.status = 'pending'
            and athlete.status = 'pending_activation'
        )
      )
      or
      (
        current_setting('sysap.activation_flow', true) = 'true'
        and status = 'active'
        and exists (
          select 1 from app.activation_invitations invitation
          join app.athlete_profiles athlete on athlete.id = invitation.profile_id
          where invitation.profile_id = organization_memberships.profile_id
            and invitation.organization_id = organization_memberships.organization_id
            and invitation.status = 'pending'
            and athlete.status = 'pending_activation'
        )
      )
    )
  );

create policy select_pending_athlete_membership_for_invitation on app.organization_memberships
  for select to sysap_api
  using (
    role = 'athlete'
    and status = 'pending_activation'
    and organization_id = app.current_tenant_id()
    and exists (
      select 1 from app.activation_invitations invitation
      where invitation.profile_id = organization_memberships.profile_id
        and invitation.organization_id = organization_memberships.organization_id
        and invitation.status = 'pending'
    )
  );

create policy update_pending_athlete_membership_for_activation on app.organization_memberships
  for update to sysap_api
  using (
    current_setting('sysap.activation_flow', true) = 'true'
    and organization_id = app.current_tenant_id()
    and role = 'athlete'
    and status = 'pending_activation'
    and exists (
      select 1 from app.activation_invitations invitation
      where invitation.profile_id = organization_memberships.profile_id
        and invitation.organization_id = organization_memberships.organization_id
        and invitation.status = 'pending'
    )
  )
  with check (
    current_setting('sysap.activation_flow', true) = 'true'
    and organization_id = app.current_tenant_id()
    and role = 'athlete'
    and status = 'active'
    and exists (
      select 1 from app.activation_invitations invitation
      where invitation.profile_id = organization_memberships.profile_id
        and invitation.organization_id = organization_memberships.organization_id
        and invitation.status in ('pending', 'consumed')
    )
  );

create policy update_athlete_profile_for_activation_completion on app.profiles
  for update to sysap_api
  using (
    current_setting('sysap.activation_flow', true) = 'true'
    and auth_user_id is null
    and exists (
      select 1 from app.activation_invitations invitation
      where invitation.profile_id = profiles.id
        and invitation.status = 'pending'
    )
  )
  with check (
    current_setting('sysap.activation_flow', true) = 'true'
    and auth_user_id is not null
    and exists (
      select 1 from app.activation_invitations invitation
      where invitation.profile_id = profiles.id
        and invitation.status = 'pending'
    )
  );

-- The owner can inspect and mutate invitation delivery only in the active
-- tenant and only while authorized as an active owner.
create policy update_activation_invitation_delivery_for_owner on app.activation_invitations
  for update to sysap_api
  using (
    organization_id = app.current_tenant_id()
    and exists (
      select 1 from app.organization_memberships membership
      join app.profiles actor on actor.id = membership.profile_id
      where membership.organization_id = activation_invitations.organization_id
        and membership.role = 'owner'
        and membership.status = 'active'
        and actor.auth_user_id = app.current_auth_subject_id()
    )
  )
  with check (
    organization_id = app.current_tenant_id()
    and status = 'pending'
    and exists (
      select 1 from app.organization_memberships membership
      join app.profiles actor on actor.id = membership.profile_id
      where membership.organization_id = activation_invitations.organization_id
        and membership.role = 'owner'
        and membership.status = 'active'
        and actor.auth_user_id = app.current_auth_subject_id()
    )
  );

revoke update on app.activation_invitations from sysap_api;
grant update (status, updated_at, delivery_status, delivery_attempt_count,
  delivery_resend_count, delivery_window_started_at, delivery_last_attempt_at)
  on app.activation_invitations to sysap_api;

grant insert on app.profiles to sysap_api;
grant update (auth_user_id, full_name, updated_at) on app.profiles to sysap_api;
grant insert on app.organization_memberships to sysap_api;
grant update (status, activated_at, updated_at) on app.organization_memberships to sysap_api;

commit;
