-- Forward-only: server-side MFA state and password-recovery proof registry.
-- Supabase Auth remains the only holder of TOTP secrets, OTPs, and passwords.
begin;

create table app.mfa_factors (
    profile_id uuid primary key references app.profiles(id) on delete restrict,
    provider_factor_id uuid not null unique,
    status text not null check (status in ('pending', 'verified', 'disabled')),
    verified_at timestamptz,
    disabled_at timestamptz,
    created_at timestamptz not null default now(),
    updated_at timestamptz not null default now(),
    constraint mfa_factors_verified_state_check check (
        (status = 'pending' and verified_at is null and disabled_at is null)
        or (status = 'verified' and verified_at is not null and disabled_at is null)
        or (status = 'disabled' and disabled_at is not null)
    )
);

comment on table app.mfa_factors is
    'Server-side MFA factor state. It stores only the provider factor identifier and state, never a TOTP secret, QR code, recovery code, token, or challenge.';

alter table app.mfa_factors enable row level security;
alter table app.mfa_factors force row level security;
revoke all on table app.mfa_factors from public, anon, authenticated, service_role;
grant select, insert on table app.mfa_factors to sysap_api;
grant update (provider_factor_id, status, verified_at, disabled_at, updated_at) on table app.mfa_factors to sysap_api;

create policy select_mfa_factor_for_current_subject
    on app.mfa_factors for select to sysap_api
    using (exists (
        select 1 from app.profiles
        where app.profiles.id = app.mfa_factors.profile_id
          and app.profiles.auth_user_id = app.current_auth_subject_id()
    ));
create policy insert_mfa_factor_for_current_subject
    on app.mfa_factors for insert to sysap_api
    with check (
        current_setting('sysap.mfa_flow', true) = 'true'
        and exists (
            select 1 from app.profiles
            where app.profiles.id = app.mfa_factors.profile_id
              and app.profiles.auth_user_id = app.current_auth_subject_id()
        )
    );
create policy update_mfa_factor_for_current_subject
    on app.mfa_factors for update to sysap_api
    using (
        current_setting('sysap.mfa_flow', true) = 'true'
        and exists (
            select 1 from app.profiles
            where app.profiles.id = app.mfa_factors.profile_id
              and app.profiles.auth_user_id = app.current_auth_subject_id()
        )
    )
    with check (
        current_setting('sysap.mfa_flow', true) = 'true'
        and exists (
            select 1 from app.profiles
            where app.profiles.id = app.mfa_factors.profile_id
              and app.profiles.auth_user_id = app.current_auth_subject_id()
        )
    );

grant update (assurance_level) on table app.auth_sessions to sysap_api;
create policy auth_sessions_elevate_for_mfa_flow
    on app.auth_sessions for update to sysap_api
    using (
        current_setting('sysap.mfa_flow', true) = 'true'
        and assurance_level = 'aal1'
        and exists (
            select 1 from app.profiles
            where app.profiles.id = app.auth_sessions.profile_id
              and app.profiles.auth_user_id = app.current_auth_subject_id()
        )
    )
    with check (
        current_setting('sysap.mfa_flow', true) = 'true'
        and assurance_level = 'aal2'
        and revoked_at is null
        and revocation_reason is null
        and exists (
            select 1 from app.profiles
            where app.profiles.id = app.auth_sessions.profile_id
              and app.profiles.auth_user_id = app.current_auth_subject_id()
        )
    );

create table app.password_recovery_challenges (
    id uuid primary key default gen_random_uuid(),
    profile_id uuid not null references app.profiles(id) on delete restrict,
    organization_id uuid not null references app.organizations(id) on delete restrict,
    proof_digest bytea check (proof_digest is null or octet_length(proof_digest) = 32),
    expires_at timestamptz not null,
    verified_at timestamptz,
    completing_at timestamptz,
    used_at timestamptz,
    created_at timestamptz not null default now(),
    constraint password_recovery_challenges_state_check check (
        used_at is null or (verified_at is not null and completing_at is not null)
    )
);

comment on table app.password_recovery_challenges is
    'Password-recovery lifecycle. OTPs are verified by Supabase Auth; the only local credential-derived value is the HMAC proof digest.';

create unique index password_recovery_challenges_one_open_per_profile
    on app.password_recovery_challenges (profile_id)
    where used_at is null;

alter table app.password_recovery_challenges enable row level security;
alter table app.password_recovery_challenges force row level security;
revoke all on table app.password_recovery_challenges from public, anon, authenticated, service_role;
grant select, insert on table app.password_recovery_challenges to sysap_api;
grant update (proof_digest, expires_at, verified_at, completing_at, used_at) on table app.password_recovery_challenges to sysap_api;

create policy password_recovery_challenges_server_flow
    on app.password_recovery_challenges for all to sysap_api
    using (current_setting('sysap.recovery_flow', true) = 'true')
    with check (current_setting('sysap.recovery_flow', true) = 'true');

create table app.password_recovery_rate_limits (
    key_fingerprint bytea primary key check (octet_length(key_fingerprint) = 32),
    window_started_at timestamptz not null,
    attempt_count integer not null check (attempt_count between 0 and 5),
    blocked_until timestamptz,
    updated_at timestamptz not null default now()
);

comment on table app.password_recovery_rate_limits is
    'Persistent recovery rate-limit state keyed only by an HMAC of normalized identifier and network address.';

alter table app.password_recovery_rate_limits enable row level security;
alter table app.password_recovery_rate_limits force row level security;
revoke all on table app.password_recovery_rate_limits from public, anon, authenticated, service_role;
grant select, insert, update on table app.password_recovery_rate_limits to sysap_api;
create policy password_recovery_rate_limits_server_flow
    on app.password_recovery_rate_limits for all to sysap_api
    using (current_setting('sysap.recovery_flow', true) = 'true')
    with check (current_setting('sysap.recovery_flow', true) = 'true');

create policy select_login_enrollments_for_password_recovery
    on app.login_enrollments for select to sysap_api
    using (current_setting('sysap.recovery_flow', true) = 'true');
create policy select_profiles_for_password_recovery
    on app.profiles for select to sysap_api
    using (current_setting('sysap.recovery_flow', true) = 'true');
create policy select_memberships_for_password_recovery
    on app.organization_memberships for select to sysap_api
    using (current_setting('sysap.recovery_flow', true) = 'true');
create policy select_organizations_for_password_recovery
    on app.organizations for select to sysap_api
    using (current_setting('sysap.recovery_flow', true) = 'true');

create function app.complete_password_recovery(challenge_id uuid, expected_proof_digest bytea)
returns uuid
language plpgsql
security definer
set search_path = pg_catalog, app
as $$
declare
    recovered_profile_id uuid;
begin
    if current_setting('sysap.recovery_flow', true) <> 'true' then
        raise exception 'password recovery flow is required' using errcode = '42501';
    end if;
    update app.password_recovery_challenges
       set used_at = now()
     where id = challenge_id
       and proof_digest = expected_proof_digest
       and verified_at is not null
       and completing_at is not null
       and used_at is null
       and expires_at > now()
     returning profile_id into recovered_profile_id;
    if recovered_profile_id is null then
        raise exception 'password recovery proof is not usable' using errcode = 'P0001';
    end if;
    update app.auth_sessions
       set revoked_at = now(), revocation_reason = 'password_recovery'
     where profile_id = recovered_profile_id
       and revoked_at is null;
    return recovered_profile_id;
end;
$$;

revoke all on function app.complete_password_recovery(uuid, bytea) from public, anon, authenticated, service_role;
grant execute on function app.complete_password_recovery(uuid, bytea) to sysap_api;

-- Rollback is a compensating forward migration after all open recovery
-- challenges expire and any active MFA factors are reconciled with Auth.
commit;
