-- In-person (manual) password recovery channel.
--
-- An administrator verifies the athlete's identity outside the system and asks
-- the API to issue a short-lived recovery code. Only an HMAC digest of the code,
-- bound to the target profile and to the recovery purpose, is persisted. The
-- password itself stays exclusively in Supabase Auth; completion reuses the
-- existing proof flow and app.complete_password_recovery (session revocation).
begin;

alter table app.password_recovery_challenges
    add column channel text not null default 'provider',
    add column code_digest bytea,
    add column attempt_count integer not null default 0,
    add column issued_by_profile_id uuid references app.profiles(id) on delete restrict,
    add column issued_at timestamptz;

alter table app.password_recovery_challenges
    add constraint password_recovery_challenges_channel_check
        check (channel in ('provider', 'manual')),
    add constraint password_recovery_challenges_code_digest_check
        check (code_digest is null or octet_length(code_digest) = 32),
    add constraint password_recovery_challenges_attempt_count_check
        check (attempt_count between 0 and 5),
    add constraint password_recovery_challenges_channel_state_check check (
        (channel = 'provider' and code_digest is null and issued_by_profile_id is null and issued_at is null)
        or (channel = 'manual' and code_digest is not null and issued_by_profile_id is not null and issued_at is not null)
    );

comment on column app.password_recovery_challenges.code_digest is
    'HMAC-SHA256 of the manual recovery code bound to profile and purpose. The code itself is never stored.';

grant update (channel, code_digest, attempt_count, issued_by_profile_id, issued_at)
    on table app.password_recovery_challenges to sysap_api;

-- Rollback: forward-only. A compensating migration must first wait for (or
-- invalidate) every open manual challenge, delete rows with channel = 'manual',
-- and then drop the added constraints and columns. Dropping code_digest
-- irreversibly invalidates any manual recovery code already handed out.
commit;
