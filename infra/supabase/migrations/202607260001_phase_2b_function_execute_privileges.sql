begin;

revoke all on function app.current_tenant_id()
from public, anon, authenticated, service_role;

grant execute on function app.current_tenant_id()
to sysap_api;

revoke all on function app.contains_forbidden_keys(jsonb, integer)
from public, anon, authenticated, service_role;

grant execute on function app.contains_forbidden_keys(jsonb, integer)
to sysap_api;

commit;
