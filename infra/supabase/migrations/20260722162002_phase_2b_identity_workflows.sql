-- 20260722162002_phase_2b_identity_workflows.sql

create table app.athlete_invitations (
    id uuid not null default gen_random_uuid(),
    organization_id uuid not null,
    athlete_id uuid not null,
    created_by_membership_id uuid not null,
    status text not null,
    expires_at timestamptz not null,
    accepted_at timestamptz,
    cancelled_at timestamptz,
    created_at timestamptz not null default now(),
    updated_at timestamptz not null default now(),
    constraint athlete_invitations_pkey primary key (id),
    constraint athlete_invitations_org_id_key unique (organization_id, id),
    constraint athlete_invitations_org_fkey foreign key (organization_id) references app.organizations(id),
    constraint athlete_invitations_athlete_fkey foreign key (organization_id, athlete_id) references app.athletes(organization_id, id),
    constraint athlete_invitations_created_by_fkey foreign key (organization_id, created_by_membership_id) references app.organization_memberships(organization_id, id),
    constraint athlete_invitations_status_check check (status in (
        'pending_provisioning',
        'provisioning_failed',
        'pending_activation',
        'activation_finalizing',
        'accepted',
        'cancelled',
        'expired'
    ))
);
alter table app.athlete_invitations enable row level security;
alter table app.athlete_invitations force row level security;

revoke all on table app.athlete_invitations from public, anon, authenticated, service_role;
grant select, insert, update on table app.athlete_invitations to sysap_api;

create or replace function app.check_athlete_invitation_transition()
returns trigger as $$
begin
    if old.status = 'accepted' or old.status = 'cancelled' or old.status = 'expired' then
        raise exception 'cannot transition from terminal state %', old.status;
    end if;

    if old.status = 'pending_provisioning' and new.status not in ('pending_activation', 'provisioning_failed', 'cancelled', 'expired') then
        raise exception 'invalid transition from % to %', old.status, new.status;
    end if;

    if old.status = 'provisioning_failed' and new.status not in ('pending_provisioning', 'cancelled') then
        raise exception 'invalid transition from % to %', old.status, new.status;
    end if;

    if old.status = 'pending_activation' and new.status not in ('activation_finalizing', 'cancelled', 'expired') then
        raise exception 'invalid transition from % to %', old.status, new.status;
    end if;

    if old.status = 'activation_finalizing' and new.status not in ('accepted', 'pending_activation') then
        raise exception 'invalid transition from % to %', old.status, new.status;
    end if;

    return new;
end;
$$ language plpgsql;

create trigger trg_athlete_invitations_transition
before update of status on app.athlete_invitations
for each row
execute function app.check_athlete_invitation_transition();


create table app.identity_operations (
    id uuid not null default gen_random_uuid(),
    organization_id uuid not null,
    profile_id uuid,
    invitation_id uuid,
    purpose text not null check (purpose in ('provision_identity', 'activate_identity', 'account_recovery', 'revoke_sessions')),
    status text not null check (status in ('pending', 'processing', 'succeeded', 'failed')),
    attempt_count integer not null default 0 check (attempt_count >= 0),
    run_after timestamptz not null default now(),
    external_reference text,
    last_error_code text,
    created_at timestamptz not null default now(),
    updated_at timestamptz not null default now(),
    completed_at timestamptz,
    constraint identity_operations_pkey primary key (id),
    constraint identity_operations_org_fkey foreign key (organization_id) references app.organizations(id),
    -- We reference organization_id, profile_id as composite to avoid cross-tenant leaks.
    -- Profiles table doesn't have organization_id natively, it's global! Oh wait.
    -- In `profiles`, is it global? Yes, a person can be in multiple organizations.
    -- Wait, the prompt says: profiles tem id, auth_user_id, full_name, created_at, updated_at.
    -- So profile is global. But identity_operations has organization_id, profile_id. 
    -- Is profile_id constrained by tenant here? 
    -- "operacao de identidade nao mistura organizacao, perfil e convite".
    constraint identity_operations_profile_fkey foreign key (profile_id) references app.profiles(id),
    constraint identity_operations_invitation_fkey foreign key (organization_id, invitation_id) references app.athlete_invitations(organization_id, id)
);

alter table app.identity_operations enable row level security;
alter table app.identity_operations force row level security;

revoke all on table app.identity_operations from public, anon, authenticated, service_role;
grant select, insert, update on table app.identity_operations to sysap_api;

create or replace function app.check_identity_operations_transition()
returns trigger as $$
begin
    if old.status = 'succeeded' then
        raise exception 'cannot transition from terminal state %', old.status;
    end if;

    if old.status = 'pending' and new.status not in ('processing') then
        raise exception 'invalid transition from % to %', old.status, new.status;
    end if;

    if old.status = 'processing' and new.status not in ('succeeded', 'failed') then
        raise exception 'invalid transition from % to %', old.status, new.status;
    end if;

    if old.status = 'failed' and new.status not in ('pending') then
        raise exception 'invalid transition from % to %', old.status, new.status;
    end if;

    return new;
end;
$$ language plpgsql;

create trigger trg_identity_operations_transition
before update of status on app.identity_operations
for each row
execute function app.check_identity_operations_transition();


create table app.outbox_events (
    id uuid not null default gen_random_uuid(),
    organization_id uuid not null,
    event_type text not null,
    aggregate_type text not null,
    aggregate_id uuid not null,
    payload jsonb not null default '{}'::jsonb,
    status text not null check (status in ('pending', 'processing', 'completed', 'failed')),
    attempt_count integer not null default 0 check (attempt_count >= 0),
    run_after timestamptz not null default now(),
    created_at timestamptz not null default now(),
    processed_at timestamptz,
    constraint outbox_events_pkey primary key (id),
    constraint outbox_events_org_fkey foreign key (organization_id) references app.organizations(id)
);

alter table app.outbox_events enable row level security;
alter table app.outbox_events force row level security;

revoke all on table app.outbox_events from public, anon, authenticated, service_role;
grant select, insert, update on table app.outbox_events to sysap_api;

create or replace function app.contains_forbidden_keys(val jsonb, current_depth integer default 0) returns boolean as $$
declare
    k text;
    v jsonb;
begin
    if current_depth > 10 then
        raise exception 'json payload depth exceeded';
    end if;

    if jsonb_typeof(val) = 'object' then
        for k, v in select * from jsonb_each(val) loop
            if k ~* '(?i)(password|passwd|otp|totp|token|authorization|cookie|secret|database_url|phone|email)' then
                return true;
            end if;
            if app.contains_forbidden_keys(v, current_depth + 1) then
                return true;
            end if;
        end loop;
    elsif jsonb_typeof(val) = 'array' then
        for v in select * from jsonb_array_elements(val) loop
            if app.contains_forbidden_keys(v, current_depth + 1) then
                return true;
            end if;
        end loop;
    end if;
    return false;
end;
$$ language plpgsql immutable;

create or replace function app.check_outbox_payload_secrets()
returns trigger as $$
begin
    if app.contains_forbidden_keys(new.payload) then
        raise exception 'payload contains forbidden keys';
    end if;
    return new;
end;
$$ language plpgsql;

create trigger trg_outbox_payload_secrets
before insert or update on app.outbox_events
for each row
execute function app.check_outbox_payload_secrets();


create table app.idempotency_records (
    id uuid not null default gen_random_uuid(),
    organization_id uuid not null,
    actor_profile_id uuid not null,
    operation text not null,
    idempotency_key uuid not null,
    request_fingerprint text not null,
    resource_type text,
    resource_id text,
    response_status integer,
    created_at timestamptz not null default now(),
    expires_at timestamptz not null,
    constraint idempotency_records_pkey primary key (id),
    constraint idempotency_records_org_fkey foreign key (organization_id) references app.organizations(id),
    constraint idempotency_records_actor_fkey foreign key (actor_profile_id) references app.profiles(id),
    constraint idempotency_records_unique unique (organization_id, actor_profile_id, operation, idempotency_key)
);

create index idempotency_records_lookup_idx on app.idempotency_records(organization_id, actor_profile_id, operation, idempotency_key);

alter table app.idempotency_records enable row level security;
alter table app.idempotency_records force row level security;

revoke all on table app.idempotency_records from public, anon, authenticated, service_role;
grant select, insert, update on table app.idempotency_records to sysap_api;
