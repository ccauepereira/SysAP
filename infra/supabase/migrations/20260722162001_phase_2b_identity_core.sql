-- 20260722162001_phase_2b_identity_core.sql

create table app.organizations (
    id uuid not null default gen_random_uuid(),
    name text not null check (trim(name) <> ''),
    timezone text not null check (timezone <> ''),
    status text not null,
    created_at timestamptz not null default now(),
    updated_at timestamptz not null default now(),
    constraint organizations_pkey primary key (id)
);
alter table app.organizations enable row level security;
alter table app.organizations force row level security;

revoke all on table app.organizations from public, anon, authenticated, service_role;
grant select, update on table app.organizations to sysap_api;

create table app.profiles (
    id uuid not null default gen_random_uuid(),
    auth_user_id uuid,
    full_name text not null check (trim(full_name) <> ''),
    created_at timestamptz not null default now(),
    updated_at timestamptz not null default now(),
    constraint profiles_pkey primary key (id),
    constraint profiles_auth_user_id_key unique (auth_user_id),
    constraint profiles_auth_user_id_fkey foreign key (auth_user_id) references auth.users(id) on delete restrict
);
alter table app.profiles enable row level security;
alter table app.profiles force row level security;

revoke all on table app.profiles from public, anon, authenticated, service_role;
grant select, update on table app.profiles to sysap_api;

create table app.organization_memberships (
    id uuid not null default gen_random_uuid(),
    organization_id uuid not null,
    profile_id uuid not null,
    role text not null check (role in ('owner', 'trainer', 'athlete')),
    status text not null check (status in ('pending_activation', 'active', 'suspended', 'closed')),
    created_by_membership_id uuid,
    activated_at timestamptz,
    suspended_at timestamptz,
    closed_at timestamptz,
    created_at timestamptz not null default now(),
    updated_at timestamptz not null default now(),
    constraint organization_memberships_pkey primary key (id),
    constraint organization_memberships_org_profile_key unique (organization_id, profile_id),
    constraint organization_memberships_org_id_key unique (organization_id, id),
    constraint organization_memberships_org_fkey foreign key (organization_id) references app.organizations(id),
    constraint organization_memberships_profile_fkey foreign key (profile_id) references app.profiles(id),
    constraint organization_memberships_created_by_fkey foreign key (organization_id, created_by_membership_id) references app.organization_memberships(organization_id, id)
);
alter table app.organization_memberships enable row level security;
alter table app.organization_memberships force row level security;

revoke all on table app.organization_memberships from public, anon, authenticated, service_role;
grant select, insert, update on table app.organization_memberships to sysap_api;

create table app.athletes (
    id uuid not null default gen_random_uuid(),
    organization_id uuid not null,
    membership_id uuid not null,
    enrollment_number text not null,
    phone_e164 text,
    birth_date date,
    created_by_membership_id uuid,
    created_at timestamptz not null default now(),
    updated_at timestamptz not null default now(),
    constraint athletes_pkey primary key (id),
    constraint athletes_enrollment_key unique (enrollment_number),
    constraint athletes_org_membership_key unique (organization_id, membership_id),
    constraint athletes_org_id_key unique (organization_id, id),
    constraint athletes_org_fkey foreign key (organization_id) references app.organizations(id),
    constraint athletes_membership_fkey foreign key (organization_id, membership_id) references app.organization_memberships(organization_id, id),
    constraint athletes_created_by_fkey foreign key (organization_id, created_by_membership_id) references app.organization_memberships(organization_id, id),
    constraint athletes_enrollment_format check (enrollment_number ~ '^[0-9]{10}$')
);
alter table app.athletes enable row level security;
alter table app.athletes force row level security;

revoke all on table app.athletes from public, anon, authenticated, service_role;
grant select, insert, update on table app.athletes to sysap_api;

create table app.trainer_athlete_assignments (
    id uuid not null default gen_random_uuid(),
    organization_id uuid not null,
    trainer_membership_id uuid not null,
    athlete_id uuid not null,
    assigned_by_membership_id uuid not null,
    created_at timestamptz not null default now(),
    revoked_at timestamptz,
    constraint trainer_athlete_assignments_pkey primary key (id),
    constraint trainer_athlete_assignments_org_fkey foreign key (organization_id) references app.organizations(id),
    constraint trainer_athlete_assignments_trainer_fkey foreign key (organization_id, trainer_membership_id) references app.organization_memberships(organization_id, id),
    constraint trainer_athlete_assignments_athlete_fkey foreign key (organization_id, athlete_id) references app.athletes(organization_id, id),
    constraint trainer_athlete_assignments_assigned_by_fkey foreign key (organization_id, assigned_by_membership_id) references app.organization_memberships(organization_id, id)
);

create unique index trainer_athlete_assignments_active_idx on app.trainer_athlete_assignments(organization_id, trainer_membership_id, athlete_id) where revoked_at is null;

alter table app.trainer_athlete_assignments enable row level security;
alter table app.trainer_athlete_assignments force row level security;

revoke all on table app.trainer_athlete_assignments from public, anon, authenticated, service_role;
grant select, insert, update on table app.trainer_athlete_assignments to sysap_api;
