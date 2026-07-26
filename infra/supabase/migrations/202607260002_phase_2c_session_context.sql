begin;

create table app.auth_sessions (
    session_id uuid primary key,
    profile_id uuid not null references app.profiles(id) on delete restrict,
    assurance_level text not null check (assurance_level in ('aal1', 'aal2')),
    registered_at timestamptz not null default now(),
    revoked_at timestamptz,
    revocation_reason text,
    constraint auth_sessions_revocation_state_check check (
        (revoked_at is null and revocation_reason is null)
        or (
            revoked_at is not null
            and revocation_reason is not null
            and btrim(revocation_reason) <> ''
        )
    )
);

comment on table app.auth_sessions is
    'Local session registry for immediate SysAP revocation while a cryptographically valid JWT has not yet expired. It never stores access tokens, refresh tokens, passwords, OTPs, or TOTP secrets.';

create index auth_sessions_profile_id_idx on app.auth_sessions (profile_id);

alter table app.auth_sessions enable row level security;
alter table app.auth_sessions force row level security;

revoke all on table app.auth_sessions from public, anon, authenticated, service_role;
grant select on table app.auth_sessions to sysap_api;

create function app.current_auth_subject_id() returns uuid
language plpgsql
stable
security invoker
set search_path = pg_catalog
as $$
declare
    configured_subject_id text;
begin
    configured_subject_id := current_setting('app.current_auth_subject_id', true);

    if configured_subject_id is null or btrim(configured_subject_id) = '' then
        return null;
    end if;

    begin
        return configured_subject_id::uuid;
    exception
        when invalid_text_representation then
            return null;
    end;
end;
$$;

comment on function app.current_auth_subject_id() is
    'Returns the transaction-local authenticated subject. Only the API may set this GUC after future cryptographic JWT validation; an absent, empty, or invalid value is denied as NULL.';

create function app.current_auth_session_id() returns uuid
language plpgsql
stable
security invoker
set search_path = pg_catalog
as $$
declare
    configured_session_id text;
begin
    configured_session_id := current_setting('app.current_auth_session_id', true);

    if configured_session_id is null or btrim(configured_session_id) = '' then
        return null;
    end if;

    begin
        return configured_session_id::uuid;
    exception
        when invalid_text_representation then
            return null;
    end;
end;
$$;

comment on function app.current_auth_session_id() is
    'Returns the transaction-local authenticated session. Only the API may set this GUC after future cryptographic JWT validation; an absent, empty, or invalid value is denied as NULL.';

revoke all on function app.current_auth_subject_id()
    from public, anon, authenticated, service_role;
grant execute on function app.current_auth_subject_id() to sysap_api;

revoke all on function app.current_auth_session_id()
    from public, anon, authenticated, service_role;
grant execute on function app.current_auth_session_id() to sysap_api;

do $$
begin
    if exists (
        select 1
        from pg_proc as procedure
        cross join lateral aclexplode(
            coalesce(procedure.proacl, acldefault('f', procedure.proowner))
        ) as privilege
        where procedure.oid in (
            'app.current_auth_subject_id()'::regprocedure,
            'app.current_auth_session_id()'::regprocedure
        )
          and privilege.grantee = 0
          and privilege.privilege_type = 'EXECUTE'
    ) then
        raise exception 'identity context functions must not grant EXECUTE to PUBLIC';
    end if;
end;
$$;

create policy profiles_select_current_auth_subject
    on app.profiles
    for select
    to sysap_api
    using (auth_user_id = app.current_auth_subject_id());

create policy organization_memberships_select_current_auth_subject
    on app.organization_memberships
    for select
    to sysap_api
    using (
        exists (
            select 1
            from app.profiles
            where app.profiles.id = app.organization_memberships.profile_id
              and app.profiles.auth_user_id = app.current_auth_subject_id()
        )
    );

create policy auth_sessions_select_current_auth_subject_and_session
    on app.auth_sessions
    for select
    to sysap_api
    using (
        session_id = app.current_auth_session_id()
        and exists (
            select 1
            from app.profiles
            where app.profiles.id = app.auth_sessions.profile_id
              and app.profiles.auth_user_id = app.current_auth_subject_id()
        )
    );

-- Forward-only migration. A rollback would require dropping the table,
-- index, policies, and functions after proving no later migration depends on
-- them; production rollback is therefore a new compensating migration.
commit;
