begin;

-- The Phase 2B tenant profile policy reads organization_memberships. The
-- Phase 2C self-membership policy must read profiles to map the authenticated
-- Auth subject. Keeping both SELECT policies creates a PostgreSQL RLS cycle.
-- Profile reads in this subphase are intentionally limited to the current
-- subject; tenant-wide profile projections remain outside this delivery.
drop policy select_tenant on app.profiles;

-- Forward-only migration. Restoring tenant-wide profile reads requires a new,
-- action-specific policy design that does not reintroduce an RLS recursion.
commit;
