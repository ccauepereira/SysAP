-- 20260722162003_phase_2b_identity_security.sql

create table app.security_audit_events (
    id uuid not null default gen_random_uuid(),
    organization_id uuid not null,
    actor_profile_id uuid,
    target_profile_id uuid,
    event_type text not null,
    result text not null,
    reason_code text not null,
    resource_type text,
    resource_id text,
    request_id text not null,
    network_fingerprint text not null,
    metadata jsonb not null default '{}'::jsonb,
    created_at timestamptz not null default now(),
    constraint security_audit_events_pkey primary key (id),
    constraint security_audit_events_org_fkey foreign key (organization_id) references app.organizations(id),
    constraint security_audit_events_actor_fkey foreign key (actor_profile_id) references app.profiles(id),
    constraint security_audit_events_target_fkey foreign key (target_profile_id) references app.profiles(id)
);

alter table app.security_audit_events enable row level security;
alter table app.security_audit_events force row level security;

revoke all on table app.security_audit_events from public, anon, authenticated, service_role;
grant select, insert on table app.security_audit_events to sysap_api;
-- explicit negação de UPDATE, DELETE e TRUNCATE já ocorre pela ausência de grant

create or replace function app.check_audit_metadata_secrets()
returns trigger as $$
begin
    if app.contains_forbidden_keys(new.metadata) then
        raise exception 'audit metadata contains forbidden keys';
    end if;
    return new;
end;
$$ language plpgsql;

create trigger trg_audit_metadata_secrets
before insert on app.security_audit_events
for each row
execute function app.check_audit_metadata_secrets();


-- Last owner protection
create or replace function app.protect_last_owner()
returns trigger as $$
declare
    active_owners integer;
begin
    if (tg_op = 'DELETE') or (old.role = 'owner' and old.status = 'active' and (new.role <> 'owner' or new.status <> 'active')) then
        if old.role = 'owner' and old.status = 'active' then
            -- Lock organization to serialize
            perform 1 from app.organizations where id = old.organization_id for update;
            
            -- Count other active owners
            select count(*) into active_owners
            from app.organization_memberships
            where organization_id = old.organization_id
              and role = 'owner'
              and status = 'active'
              and id <> old.id;
              
            if active_owners = 0 then
                raise exception 'cannot remove the last active owner of the organization';
            end if;
        end if;
    end if;
    
    if (tg_op = 'DELETE') then
        return old;
    end if;
    return new;
end;
$$ language plpgsql;

create trigger trg_protect_last_owner
before update or delete on app.organization_memberships
for each row
execute function app.protect_last_owner();


-- RLS Functions
create or replace function app.current_tenant_id() returns uuid as $$
begin
    return nullif(current_setting('app.current_organization_id', true), '')::uuid;
exception
    when others then
        return null;
end;
$$ language plpgsql stable;

-- Apply RLS policies
create policy isolate_tenant on app.organizations
for all to sysap_api
using (id = app.current_tenant_id())
with check (id = app.current_tenant_id());

create policy isolate_tenant on app.organization_memberships
for all to sysap_api
using (organization_id = app.current_tenant_id())
with check (organization_id = app.current_tenant_id());

create policy isolate_tenant on app.athletes
for all to sysap_api
using (organization_id = app.current_tenant_id())
with check (organization_id = app.current_tenant_id());

create policy isolate_tenant on app.trainer_athlete_assignments
for all to sysap_api
using (organization_id = app.current_tenant_id())
with check (organization_id = app.current_tenant_id());

create policy isolate_tenant on app.athlete_invitations
for all to sysap_api
using (organization_id = app.current_tenant_id())
with check (organization_id = app.current_tenant_id());

create policy isolate_tenant on app.identity_operations
for all to sysap_api
using (organization_id = app.current_tenant_id())
with check (organization_id = app.current_tenant_id());

create policy isolate_tenant on app.outbox_events
for all to sysap_api
using (organization_id = app.current_tenant_id())
with check (organization_id = app.current_tenant_id());

create policy isolate_tenant on app.idempotency_records
for all to sysap_api
using (organization_id = app.current_tenant_id())
with check (organization_id = app.current_tenant_id());

create policy isolate_tenant on app.security_audit_events
for select to sysap_api
using (organization_id = app.current_tenant_id());

create policy insert_tenant on app.security_audit_events
for insert to sysap_api
with check (organization_id = app.current_tenant_id());

-- Profiles policy: visible if they are linked to the tenant, insertable by the API
create policy select_tenant on app.profiles
for select to sysap_api
using (
    exists (
        select 1 from app.organization_memberships
        where profile_id = app.profiles.id
          and organization_id = app.current_tenant_id()
    )
    or
    exists (
        select 1 from app.identity_operations
        where profile_id = app.profiles.id
          and organization_id = app.current_tenant_id()
    )
);

create policy update_tenant on app.profiles
for update to sysap_api
using (
    exists (
        select 1 from app.organization_memberships
        where profile_id = app.profiles.id
          and organization_id = app.current_tenant_id()
    )
)
with check (
    exists (
        select 1 from app.organization_memberships
        where profile_id = app.profiles.id
          and organization_id = app.current_tenant_id()
    )
);


