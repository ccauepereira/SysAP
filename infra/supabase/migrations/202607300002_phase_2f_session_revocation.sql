-- Forward-only: adds the narrowly scoped local authority required by refresh
-- and revocation. Tokens are intentionally absent from every new structure.
begin;

alter table app.auth_sessions
    add column organization_id uuid references app.organizations(id) on delete restrict;

create index auth_sessions_organization_id_idx on app.auth_sessions (organization_id);

-- Sessions created before this cut can be associated only when the profile has
-- a single provisioned login organization. Ambiguous legacy sessions remain
-- unusable for refresh and must authenticate again rather than guessing.
update app.auth_sessions session
set organization_id = enrollment.organization_id
from app.login_enrollments enrollment
where enrollment.profile_id = session.profile_id
  and session.organization_id is null
  and not exists (
      select 1
      from app.login_enrollments other_enrollment
      where other_enrollment.profile_id = session.profile_id
        and other_enrollment.organization_id <> enrollment.organization_id
  );

grant update on table app.auth_sessions to sysap_api;

create policy auth_sessions_manage_for_session_flow
    on app.auth_sessions
    for update
    to sysap_api
    using (
        current_setting('sysap.session_flow', true) = 'true'
        and exists (
            select 1
            from app.profiles
            where app.profiles.id = app.auth_sessions.profile_id
              and app.profiles.auth_user_id = app.current_auth_subject_id()
        )
    )
    with check (
        current_setting('sysap.session_flow', true) = 'true'
        and exists (
            select 1
            from app.profiles
            where app.profiles.id = app.auth_sessions.profile_id
              and app.profiles.auth_user_id = app.current_auth_subject_id()
        )
    );

create policy organizations_select_for_session_flow
    on app.organizations
    for select
    to sysap_api
    using (
        current_setting('sysap.session_flow', true) = 'true'
        and exists (
            select 1
            from app.organization_memberships membership
            join app.profiles profile on profile.id = membership.profile_id
            where membership.organization_id = app.organizations.id
              and profile.auth_user_id = app.current_auth_subject_id()
        )
    );

create policy audit_session_lifecycle_for_session_flow
    on app.security_audit_events
    for insert
    to sysap_api
    with check (
        current_setting('sysap.session_flow', true) = 'true'
        and event_type in ('session_refreshed', 'session_revoked', 'sessions_revoked')
        and actor_profile_id = target_profile_id
        and exists (
            select 1
            from app.profiles
            where app.profiles.id = app.security_audit_events.actor_profile_id
              and app.profiles.auth_user_id = app.current_auth_subject_id()
        )
        and exists (
            select 1
            from app.organization_memberships membership
            where membership.organization_id = app.security_audit_events.organization_id
              and membership.profile_id = app.security_audit_events.actor_profile_id
        )
    );

-- An external-provider action is never retried with a persisted credential.
-- This record is only an operational marker for a later server-side action.
create table app.auth_session_revocation_sync (
    id uuid primary key default gen_random_uuid(),
    profile_id uuid not null references app.profiles(id) on delete restrict,
    session_id uuid references app.auth_sessions(session_id) on delete restrict,
    scope text not null check (scope in ('current_session', 'all_sessions')),
    status text not null check (status in ('pending', 'succeeded', 'failed')),
    created_at timestamptz not null default now(),
    updated_at timestamptz not null default now(),
    constraint auth_session_revocation_sync_scope_check check (
        (scope = 'current_session' and session_id is not null)
        or (scope = 'all_sessions')
    )
);

alter table app.auth_session_revocation_sync enable row level security;
alter table app.auth_session_revocation_sync force row level security;
revoke all on table app.auth_session_revocation_sync from public, anon, authenticated, service_role;
grant insert on table app.auth_session_revocation_sync to sysap_api;

create policy insert_auth_session_revocation_sync_for_session_flow
    on app.auth_session_revocation_sync
    for insert
    to sysap_api
    with check (
        current_setting('sysap.session_flow', true) = 'true'
        and status = 'pending'
        and exists (
            select 1
            from app.profiles
            where app.profiles.id = app.auth_session_revocation_sync.profile_id
              and app.profiles.auth_user_id = app.current_auth_subject_id()
        )
    );

comment on table app.auth_session_revocation_sync is
    'Credential-free marker for server-side provider revocation synchronization. It never stores tokens, passwords, OTPs, contacts, enrollment numbers, or provider errors.';

-- Rollback is a compensating forward migration after proving no pending sync
-- records remain and no later migration depends on these policies/table.
commit;
