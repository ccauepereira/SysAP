-- SysAP 2G.2: administrative athlete data and two-channel activation.
-- Forward-only: existing 2E rows remain valid as SMS challenges.

alter table app.athlete_profiles
  add column if not exists birth_date date,
  add column if not exists locality text,
  add column if not exists football_position text,
  add column if not exists enrollment_delivery_channel text;

alter table app.athlete_profiles
  drop constraint if exists athlete_profiles_delivery_channel_check;
alter table app.athlete_profiles
  add constraint athlete_profiles_delivery_channel_check
  check (enrollment_delivery_channel in ('sms', 'email'));
alter table app.athlete_profiles
  drop constraint if exists athlete_profiles_position_check;
alter table app.athlete_profiles
  add constraint athlete_profiles_position_check
  check (football_position is null or football_position in ('goalkeeper', 'defender', 'midfielder', 'forward'));

create table if not exists app.training_groups (
  id uuid primary key default gen_random_uuid(),
  organization_id uuid not null references app.organizations(id) on delete cascade,
  name text not null check (trim(name) <> ''),
  status text not null default 'active' check (status in ('active', 'archived')),
  created_at timestamptz not null default now(),
  updated_at timestamptz not null default now(),
  unique (organization_id, id),
  unique (organization_id, name)
);

create table if not exists app.training_group_athletes (
  id uuid primary key default gen_random_uuid(),
  organization_id uuid not null references app.organizations(id) on delete cascade,
  training_group_id uuid not null,
  athlete_profile_id uuid not null references app.athlete_profiles(id) on delete cascade,
  created_at timestamptz not null default now(),
  unique (organization_id, training_group_id, athlete_profile_id),
  foreign key (organization_id, training_group_id) references app.training_groups(organization_id, id) on delete cascade
);

alter table app.training_groups enable row level security;
alter table app.training_groups force row level security;
alter table app.training_group_athletes enable row level security;
alter table app.training_group_athletes force row level security;

revoke all on app.training_groups, app.training_group_athletes from public, anon, authenticated, service_role;
grant select, insert on app.training_groups to sysap_api;
grant select, insert on app.training_group_athletes to sysap_api;

create policy training_groups_tenant_select on app.training_groups
  for select to sysap_api using (organization_id = app.current_tenant_id());
create policy training_groups_admin_insert on app.training_groups
  for insert to sysap_api with check (
    organization_id = app.current_tenant_id()
    and exists (
      select 1 from app.organization_memberships m
      join app.profiles p on p.id = m.profile_id
      where m.organization_id = training_groups.organization_id
        and m.role = 'owner' and m.status = 'active'
        and p.auth_user_id = app.current_auth_subject_id()
    )
  );
create policy training_group_athletes_tenant_select on app.training_group_athletes
  for select to sysap_api using (organization_id = app.current_tenant_id());
create policy training_group_athletes_admin_insert on app.training_group_athletes
  for insert to sysap_api with check (
    organization_id = app.current_tenant_id()
    and exists (select 1 from app.training_groups g where g.organization_id = training_group_athletes.organization_id and g.id = training_group_athletes.training_group_id)
  );

alter table app.activation_challenges
  add column if not exists channel text not null default 'sms';
alter table app.activation_challenges
  drop constraint if exists activation_challenges_channel_check;
alter table app.activation_challenges
  add constraint activation_challenges_channel_check check (channel in ('sms', 'email'));
drop index if exists app.activation_challenges_invitation_active_idx;
create unique index if not exists activation_challenges_invitation_channel_active_idx
  on app.activation_challenges(invitation_id, channel)
  where invalidated_at is null and consumed_at is null;

create table if not exists app.activation_proofs (
  id uuid primary key default gen_random_uuid(),
  invitation_id uuid not null references app.activation_invitations(id) on delete cascade,
  athlete_profile_id uuid not null references app.athlete_profiles(id) on delete cascade,
  kind text not null check (kind in ('sms', 'email', 'final')),
  proof_hmac bytea not null,
  expires_at timestamptz not null,
  consumed_at timestamptz,
  created_at timestamptz not null default now()
);
create unique index if not exists activation_proofs_active_kind_idx
  on app.activation_proofs(invitation_id, kind) where consumed_at is null;
alter table app.activation_proofs enable row level security;
alter table app.activation_proofs force row level security;
revoke all on app.activation_proofs from public, anon, authenticated, service_role;
grant select, insert, update on app.activation_proofs to sysap_api;
create policy activation_proofs_flow_select on app.activation_proofs
  for select to sysap_api using (
    current_setting('sysap.activation_flow', true) = 'true'
    and exists (
      select 1 from app.activation_invitations i
      join app.athlete_profiles p on p.id = i.profile_id
      where i.id = activation_proofs.invitation_id
        and p.id = activation_proofs.athlete_profile_id
        and i.status = 'pending' and p.status = 'pending_activation'
    )
  );
create policy activation_proofs_flow_insert on app.activation_proofs
  for insert to sysap_api with check (
    current_setting('sysap.activation_flow', true) = 'true'
    and exists (
      select 1 from app.activation_invitations i
      join app.athlete_profiles p on p.id = i.profile_id
      where i.id = activation_proofs.invitation_id
        and p.id = activation_proofs.athlete_profile_id
        and i.status = 'pending' and p.status = 'pending_activation'
    )
  );
create policy activation_proofs_flow_update on app.activation_proofs
  for update to sysap_api
  using (
    current_setting('sysap.activation_flow', true) = 'true'
    and exists (
      select 1 from app.activation_invitations i
      join app.athlete_profiles p on p.id = i.profile_id
      where i.id = activation_proofs.invitation_id
        and p.id = activation_proofs.athlete_profile_id
        and i.status = 'pending' and p.status = 'pending_activation'
    )
  )
  with check (
    current_setting('sysap.activation_flow', true) = 'true'
    and exists (
      select 1 from app.activation_invitations i
      join app.athlete_profiles p on p.id = i.profile_id
      where i.id = activation_proofs.invitation_id
        and p.id = activation_proofs.athlete_profile_id
        and i.status = 'pending' and p.status = 'pending_activation'
    )
  );

-- Activation writes are intentionally limited to the service role and guarded
-- by the activation_flow transaction setting; client roles receive nothing.
grant update (birth_date, locality, football_position, enrollment_delivery_channel, status, updated_at)
  on app.athlete_profiles to sysap_api;
grant update (status, updated_at) on app.activation_invitations to sysap_api;

-- Rollback is not automatic: drop the new tables/columns only after all
-- activation data has been exported and the deployment is deliberately retired.
