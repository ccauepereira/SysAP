begin;

alter table app.profiles
    add column suspended_at timestamptz;

comment on column app.profiles.suspended_at is
    'When present, denies every subsequent protected API request for the profile. The access token and session registry remain non-secret.';

-- Forward-only migration. Clearing a mistaken suspension is a data correction;
-- reverting this schema change requires a compensating migration after proving
-- no protected route or audit process still depends on the column.
commit;
