import { LocalDatabaseSession, OwnedProcesses, startAPI } from "./scripts/local-runtime.mjs";
import { readSupabaseEnvironment } from "./scripts/local-runtime.mjs";
async function run() {
  const db = new LocalDatabaseSession();
  const baseEnvironment = await db.environment();
  const supabaseEnv = await readSupabaseEnvironment();
  const environment = {
    ...baseEnvironment,
    SYSAP_SUPABASE_AUTH_URL: `${supabaseEnv.apiURL}/auth/v1`,
    SYSAP_SUPABASE_SERVICE_ROLE_KEY: supabaseEnv.serviceRoleKey,
    SYSAP_AUTH_JWT_ISSUER: `${supabaseEnv.apiURL}/auth/v1`,
    SYSAP_AUTH_JWT_AUDIENCE: "authenticated",
    SYSAP_AUTH_JWKS_URL: `${supabaseEnv.apiURL}/auth/v1/.well-known/jwks.json`,
  };
  console.log(environment);
}
run();
