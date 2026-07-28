begin;

-- Corrective forward-only migration. The previous migration is already
-- applied in development environments and is deliberately left untouched.
drop policy if exists select_activation_challenges on app.activation_challenges;
drop policy if exists insert_activation_challenges on app.activation_challenges;
drop policy if exists update_activation_challenges on app.activation_challenges;
drop policy if exists insert_profile_for_activation on app.profiles;
drop policy if exists insert_membership_for_activation on app.organization_memberships;
drop policy if exists update_athlete_profile_for_activation on app.athlete_profiles;
drop policy if exists update_activation_invitation_for_activation on app.activation_invitations;

create policy select_activation_challenges on app.activation_challenges
    for select to sysap_api
    using (
        current_setting('sysap.activation_flow', true) = 'true'
        and exists (
            select 1
            from app.athlete_profiles athlete
            join app.activation_invitations invitation on invitation.id = app.activation_challenges.invitation_id
            where athlete.id = app.activation_challenges.athlete_profile_id
              and invitation.profile_id = athlete.id
              and (
                  (athlete.status = 'pending_activation' and invitation.status = 'pending')
                  or (athlete.status = 'activated' and invitation.status = 'consumed')
              )
        )
    );

create policy insert_activation_challenges on app.activation_challenges
    for insert to sysap_api
    with check (
        current_setting('sysap.activation_flow', true) = 'true'
        and exists (
            select 1
            from app.athlete_profiles athlete
            join app.activation_invitations invitation on invitation.id = app.activation_challenges.invitation_id
            where athlete.id = app.activation_challenges.athlete_profile_id
              and invitation.profile_id = athlete.id
              and athlete.status = 'pending_activation'
              and invitation.status = 'pending'
        )
    );

create policy update_activation_challenges on app.activation_challenges
    for update to sysap_api
    using (
        current_setting('sysap.activation_flow', true) = 'true'
        and exists (
            select 1
            from app.athlete_profiles athlete
            join app.activation_invitations invitation on invitation.id = app.activation_challenges.invitation_id
            where athlete.id = app.activation_challenges.athlete_profile_id
              and invitation.profile_id = athlete.id
              and (
                  (athlete.status = 'pending_activation' and invitation.status = 'pending')
                  or (athlete.status = 'activated' and invitation.status = 'consumed')
              )
        )
    )
    with check (
        current_setting('sysap.activation_flow', true) = 'true'
        and exists (
            select 1
            from app.athlete_profiles athlete
            join app.activation_invitations invitation on invitation.id = app.activation_challenges.invitation_id
            where athlete.id = app.activation_challenges.athlete_profile_id
              and invitation.profile_id = athlete.id
              and (
                  (athlete.status = 'pending_activation' and invitation.status = 'pending')
                  or (athlete.status = 'activated' and invitation.status = 'consumed')
              )
        )
    );

create policy insert_profile_for_activation on app.profiles
    for insert to sysap_api
    with check (
        current_setting('sysap.activation_flow', true) = 'true'
        and exists (
            select 1 from app.athlete_profiles athlete
            join app.activation_invitations invitation on invitation.profile_id = athlete.id
            where athlete.id = app.profiles.id
              and athlete.status = 'pending_activation'
              and invitation.status = 'pending'
        )
    );

create policy insert_membership_for_activation on app.organization_memberships
    for insert to sysap_api
    with check (
        current_setting('sysap.activation_flow', true) = 'true'
        and app.organization_memberships.role = 'athlete'
        and app.organization_memberships.status = 'active'
        and exists (
            select 1 from app.athlete_profiles athlete
            join app.activation_invitations invitation on invitation.profile_id = athlete.id
            where athlete.id = app.organization_memberships.profile_id
              and invitation.organization_id = app.organization_memberships.organization_id
              and athlete.status = 'pending_activation'
              and invitation.status = 'pending'
        )
    );

create policy update_athlete_profile_for_activation on app.athlete_profiles
    for update to sysap_api
    using (
        current_setting('sysap.activation_flow', true) = 'true'
        and status = 'pending_activation'
        and exists (
            select 1 from app.activation_invitations invitation
            where invitation.profile_id = app.athlete_profiles.id
              and invitation.status = 'pending'
        )
    )
    with check (
        current_setting('sysap.activation_flow', true) = 'true'
        and status in ('pending_activation', 'activated')
        and exists (
            select 1 from app.activation_invitations invitation
            where invitation.profile_id = app.athlete_profiles.id
              and invitation.status = 'pending'
        )
    );

create policy update_activation_invitation_for_activation on app.activation_invitations
    for update to sysap_api
    using (
        current_setting('sysap.activation_flow', true) = 'true'
        and status = 'pending'
        and exists (
            select 1 from app.athlete_profiles athlete
            where athlete.id = app.activation_invitations.profile_id
              and athlete.status in ('pending_activation', 'activated')
        )
    )
    with check (
        current_setting('sysap.activation_flow', true) = 'true'
        and status in ('pending', 'consumed')
        and exists (
            select 1 from app.athlete_profiles athlete
            where athlete.id = app.activation_invitations.profile_id
              and athlete.status in ('pending_activation', 'activated')
        )
    );

drop index if exists app.activation_challenges_invitation_active_idx;
create unique index activation_challenges_invitation_active_idx
    on app.activation_challenges(invitation_id)
    where invalidated_at is null and consumed_at is null;

create table app.identity_repair_tasks (
    repair_id uuid primary key default gen_random_uuid(),
    auth_user_id uuid not null,
    operation text not null check (operation = 'delete_auth_user'),
    status text not null check (status = 'pending'),
    created_at timestamptz not null default now(),
    updated_at timestamptz not null default now()
);
create unique index identity_repair_tasks_pending_auth_operation_idx
    on app.identity_repair_tasks(auth_user_id, operation) where status = 'pending';
alter table app.identity_repair_tasks enable row level security;
alter table app.identity_repair_tasks force row level security;
revoke all on app.identity_repair_tasks from public, anon, authenticated, service_role;
grant select, insert on app.identity_repair_tasks to sysap_api;
create policy select_identity_repair_task on app.identity_repair_tasks
    for select to sysap_api
    using (current_setting('sysap.activation_flow', true) = 'true');
create policy insert_identity_repair_task on app.identity_repair_tasks
    for insert to sysap_api
    with check (current_setting('sysap.activation_flow', true) = 'true');

commit;
