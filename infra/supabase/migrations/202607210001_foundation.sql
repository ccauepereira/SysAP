do $$
begin
  if not exists (select 1 from pg_roles where rolname = 'sysap_api') then
    create role sysap_api with
      nologin
      nosuperuser
      nocreatedb
      nocreaterole
      noreplication
      nobypassrls
      noinherit;
  end if;
end
$$;

alter role sysap_api with
  nologin
  nocreatedb
  nocreaterole
  noinherit;

do $$
begin
  if exists (
    select 1
    from pg_roles
    where rolname = 'sysap_api'
      and (rolsuper or rolreplication or rolbypassrls)
  ) then
    raise exception using
      errcode = '42501',
      message = 'sysap_api has protected attributes that the migration executor cannot safely remove';
  end if;
end
$$;

do $$
declare
  membership record;
begin
  for membership in
    select granted_role.rolname
    from pg_auth_members
    join pg_roles as member_role
      on member_role.oid = pg_auth_members.member
    join pg_roles as granted_role
      on granted_role.oid = pg_auth_members.roleid
    where member_role.rolname = 'sysap_api'
  loop
    execute format('revoke %I from sysap_api', membership.rolname);
  end loop;
end
$$;

grant sysap_api to postgres with inherit false, set true;

create schema app;

comment on schema app is
  'Private SysAP schema. Clients access business data only through the API.';

revoke create on schema public from public;
revoke all on schema app from public, anon, authenticated, service_role;
grant usage on schema app to sysap_api;

alter default privileges in schema app
  revoke all on tables from public, anon, authenticated, service_role;
alter default privileges in schema app
  revoke all on sequences from public, anon, authenticated, service_role;
alter default privileges
  revoke execute on functions from public, anon, authenticated, service_role;

create table app.bootstrap_metadata (
  singleton boolean primary key default true check (singleton),
  schema_version integer not null check (schema_version > 0),
  initialized_at timestamptz not null default now()
);

comment on table app.bootstrap_metadata is
  'Singleton metadata used to verify the local database foundation.';

alter table app.bootstrap_metadata enable row level security;

revoke all on table app.bootstrap_metadata
  from public, anon, authenticated, service_role;
grant select on table app.bootstrap_metadata to sysap_api;

create policy bootstrap_metadata_select_for_api
  on app.bootstrap_metadata
  for select
  to sysap_api
  using (true);

insert into app.bootstrap_metadata (singleton, schema_version)
values (true, 1);

-- Rollback strategy:
-- drop table app.bootstrap_metadata;
-- drop schema app;
-- revoke sysap_api from postgres;
-- drop role sysap_api; -- Only when this migration introduced the role.
