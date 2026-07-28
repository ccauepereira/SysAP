begin;

-- Forward-only: this migration introduces the minimum server-only state for
-- password login. A rollback must be a compensating migration after active
-- sessions and rate-limit records have expired.

create table app.login_enrollments (
    enrollment_number text primary key check (enrollment_number ~ '^[0-9]{10}$'),
    profile_id uuid not null references app.profiles(id) on delete restrict,
    organization_id uuid not null references app.organizations(id) on delete restrict,
    created_at timestamptz not null default now(),
    constraint login_enrollments_profile_organization_key unique (profile_id, organization_id)
);

comment on table app.login_enrollments is
    'Server-only mapping from a global enrollment number to one SysAP profile and organization. It contains no password, token, phone, email, or provider secret.';

alter table app.login_enrollments enable row level security;
alter table app.login_enrollments force row level security;
revoke all on table app.login_enrollments from public, anon, authenticated, service_role;
grant select, insert on table app.login_enrollments to sysap_api;

create policy select_login_enrollments_for_login
    on app.login_enrollments
    for select to sysap_api
    using (current_setting('sysap.login_flow', true) = 'true');

create policy insert_login_enrollments_for_activation
    on app.login_enrollments
    for insert to sysap_api
    with check (
        current_setting('sysap.activation_flow', true) = 'true'
        and exists (
            select 1
            from app.athlete_profiles athlete
            where athlete.id = app.login_enrollments.profile_id
              and athlete.organization_id = app.login_enrollments.organization_id
              and athlete.enrollment_number = app.login_enrollments.enrollment_number
              and athlete.status = 'pending_activation'
        )
    );

-- Existing fictional/legacy athletes are mapped when their profile is already
-- active. Future staff provisioning creates the same server-only mapping.
insert into app.login_enrollments (enrollment_number, profile_id, organization_id)
select athlete.enrollment_number, membership.profile_id, athlete.organization_id
from app.athletes athlete
join app.organization_memberships membership
  on membership.id = athlete.membership_id
 and membership.organization_id = athlete.organization_id
where membership.status = 'active'
on conflict (enrollment_number) do nothing;

insert into app.login_enrollments (enrollment_number, profile_id, organization_id)
select athlete.enrollment_number, profile.id, athlete.organization_id
from app.athlete_profiles athlete
join app.profiles profile on profile.id = athlete.id
where athlete.status = 'activated'
on conflict (enrollment_number) do nothing;

create table app.security_rate_limits (
    key_fingerprint bytea primary key check (octet_length(key_fingerprint) = 32),
    window_started_at timestamptz not null,
    failure_count integer not null check (failure_count between 0 and 5),
    blocked_until timestamptz,
    updated_at timestamptz not null default now()
);

comment on table app.security_rate_limits is
    'Persistent password-login limit state keyed only by an HMAC of normalized IP and enrollment number.';

alter table app.security_rate_limits enable row level security;
alter table app.security_rate_limits force row level security;
revoke all on table app.security_rate_limits from public, anon, authenticated, service_role;
grant select, insert, update on table app.security_rate_limits to sysap_api;

create policy select_login_rate_limit
    on app.security_rate_limits
    for select to sysap_api
    using (current_setting('sysap.login_flow', true) = 'true');
create policy insert_login_rate_limit
    on app.security_rate_limits
    for insert to sysap_api
    with check (current_setting('sysap.login_flow', true) = 'true');
create policy update_login_rate_limit
    on app.security_rate_limits
    for update to sysap_api
    using (current_setting('sysap.login_flow', true) = 'true')
    with check (current_setting('sysap.login_flow', true) = 'true');

-- Login failures for an unknown enrollment have no organization to reference.
alter table app.security_audit_events alter column organization_id drop not null;

create policy select_profiles_for_login
    on app.profiles
    for select to sysap_api
    using (current_setting('sysap.login_flow', true) = 'true');
create policy select_memberships_for_login
    on app.organization_memberships
    for select to sysap_api
    using (current_setting('sysap.login_flow', true) = 'true');
create policy select_organizations_for_login
    on app.organizations
    for select to sysap_api
    using (current_setting('sysap.login_flow', true) = 'true');

grant insert on table app.auth_sessions to sysap_api;
create policy insert_auth_session_for_login
    on app.auth_sessions
    for insert to sysap_api
    with check (
        current_setting('sysap.login_flow', true) = 'true'
        and assurance_level = 'aal1'
        and revoked_at is null
        and revocation_reason is null
        and exists (select 1 from app.profiles where id = app.auth_sessions.profile_id)
    );

create policy insert_security_audit_for_login
    on app.security_audit_events
    for insert to sysap_api
    with check (
        current_setting('sysap.login_flow', true) = 'true'
        and event_type in ('login_success', 'login_failed')
        and result in ('success', 'failure')
        and reason_code in ('authenticated', 'invalid_credentials', 'rate_limited')
        and jsonb_typeof(metadata) = 'object'
    );

commit;
