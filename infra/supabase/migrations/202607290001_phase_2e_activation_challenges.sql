begin;

create table app.activation_challenges (
    challenge_id uuid primary key default gen_random_uuid(),
    athlete_profile_id uuid not null references app.athlete_profiles(id) on delete cascade,
    invitation_id uuid not null references app.activation_invitations(id) on delete cascade,
    otp_hmac bytea not null,
    created_at timestamptz not null default now(),
    expires_at timestamptz not null,
    attempt_count integer not null default 0 check (attempt_count between 0 and 5),
    resend_count integer not null default 0 check (resend_count between 0 and 3),
    resend_window_started_at timestamptz not null default now(),
    invalidated_at timestamptz,
    consumed_at timestamptz,
    activation_proof_hmac bytea,
    proof_expires_at timestamptz,
    proof_consumed_at timestamptz,
    constraint activation_challenges_expiry_check check (expires_at > created_at),
    constraint activation_challenges_proof_state_check check (
        (activation_proof_hmac is null and proof_expires_at is null and proof_consumed_at is null)
        or (activation_proof_hmac is not null and proof_expires_at is not null)
    )
);
create index activation_challenges_invitation_active_idx on app.activation_challenges(invitation_id) where invalidated_at is null and consumed_at is null;
alter table app.activation_challenges enable row level security;
alter table app.activation_challenges force row level security;
revoke all on app.activation_challenges from public, anon, authenticated, service_role;
grant select, insert, update on app.activation_challenges to sysap_api;
create policy select_activation_challenges on app.activation_challenges for select to sysap_api using (true);
create policy insert_activation_challenges on app.activation_challenges for insert to sysap_api with check (true);
create policy update_activation_challenges on app.activation_challenges for update to sysap_api using (true);

create policy select_athlete_profile_for_activation on app.athlete_profiles for select to sysap_api using (status in ('pending_activation', 'activated') and current_setting('sysap.activation_flow', true) = 'true');
create policy select_activation_invitation_for_activation on app.activation_invitations for select to sysap_api using (status in ('pending', 'consumed') and current_setting('sysap.activation_flow', true) = 'true');

create index athlete_profiles_enrollment_activation_idx on app.athlete_profiles(enrollment_number) where status = 'pending_activation';

grant insert on table app.profiles to sysap_api;
create policy insert_profile_for_activation on app.profiles for insert to sysap_api with check (current_setting('sysap.activation_flow', true) = 'true');

grant insert on table app.organization_memberships to sysap_api;
create policy insert_membership_for_activation on app.organization_memberships for insert to sysap_api with check (current_setting('sysap.activation_flow', true) = 'true');

grant update on table app.athlete_profiles to sysap_api;
create policy update_athlete_profile_for_activation on app.athlete_profiles for update to sysap_api using (current_setting('sysap.activation_flow', true) = 'true') with check (true);

grant update on table app.activation_invitations to sysap_api;
create policy update_activation_invitation_for_activation on app.activation_invitations for update to sysap_api using (current_setting('sysap.activation_flow', true) = 'true') with check (true);

commit;
