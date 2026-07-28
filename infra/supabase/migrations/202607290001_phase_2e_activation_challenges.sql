begin;

create table app.activation_challenges (
    id uuid primary key default gen_random_uuid(),
    athlete_profile_id uuid not null references app.athlete_profiles(id) on delete cascade,
    invitation_id uuid not null references app.activation_invitations(id) on delete cascade,
    otp_hmac bytea not null,
    created_at timestamptz not null default now(),
    expires_at timestamptz not null,
    attempt_count integer not null default 0 check (attempt_count between 0 and 5),
    resend_count integer not null default 0 check (resend_count between 0 and 3),
    resend_window_started_at timestamptz not null default now(),
    invalidated_at timestamptz,
    verified_at timestamptz,
    activation_proof_hash bytea,
    proof_expires_at timestamptz,
    proof_used_at timestamptz,
    constraint activation_challenges_expiry_check check (expires_at > created_at),
    constraint activation_challenges_proof_state_check check (
        (activation_proof_hash is null and proof_expires_at is null and proof_used_at is null)
        or (activation_proof_hash is not null and proof_expires_at is not null)
    )
);

create unique index activation_challenges_one_active_per_invitation
    on app.activation_challenges(invitation_id) where invalidated_at is null and verified_at is null;

alter table app.activation_challenges enable row level security;
alter table app.activation_challenges force row level security;
revoke all on table app.activation_challenges from public, anon, authenticated, service_role;
grant select, insert, update on table app.activation_challenges to sysap_api;
create policy activation_challenges_api_only on app.activation_challenges
    for all to sysap_api using (true) with check (true);

commit;
