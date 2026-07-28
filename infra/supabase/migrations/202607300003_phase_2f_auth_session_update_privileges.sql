-- Forward-only correction for 2F.1. Session lifecycle only changes the two
-- revocation fields; table-wide UPDATE would permit unrelated state changes.
begin;

revoke update on table app.auth_sessions from sysap_api;
grant update (revoked_at, revocation_reason) on table app.auth_sessions to sysap_api;

-- Later migrations must use a compensating migration if a new session field
-- needs to be mutable. This migration deliberately grants no DELETE,
-- TRUNCATE, REFERENCES, or TRIGGER privilege.
commit;
