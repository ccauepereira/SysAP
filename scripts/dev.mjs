import {
  LocalDatabaseSession,
  OwnedProcesses,
  apiURL,
  assertPortAvailable,
  buildAPI,
  startAPI,
  startWeb,
  waitForAPI,
  waitForWeb,
  webURL,
  readSupabaseEnvironment,
} from "./local-runtime.mjs";
import { sanitizeMessage } from "./runtime-policy.mjs";

const database = new LocalDatabaseSession();
const processes = new OwnedProcesses();
let requestedSignal = null;
let releaseStop;
const stopRequested = new Promise((resolve) => {
  releaseStop = resolve;
});

for (const signal of ["SIGINT", "SIGTERM"]) {
  process.once(signal, () => {
    requestedSignal = signal;
    releaseStop({ kind: "signal", signal });
  });
}

async function main() {
  await assertPortAvailable("127.0.0.1", 8080);
  await assertPortAvailable("127.0.0.1", 3000);
  await database.start();
  const baseEnvironment = await database.environment();
  const supabaseEnv = await readSupabaseEnvironment();
  const environment = {
    ...baseEnvironment,
    SYSAP_SUPABASE_AUTH_URL: `${supabaseEnv.apiURL}/auth/v1`,
    SYSAP_SUPABASE_SERVICE_ROLE_KEY: supabaseEnv.serviceRoleKey,
    SYSAP_AUTH_JWT_ISSUER: `${supabaseEnv.apiURL}/auth/v1`,
    SYSAP_AUTH_JWT_AUDIENCE: "authenticated",
    SYSAP_AUTH_JWKS_URL: `${supabaseEnv.apiURL}/auth/v1/.well-known/jwks.json`,
  };
  const binary = await buildAPI("sysap-api-dev");
  const api = startAPI(processes, binary, environment);
  const web = startWeb(processes, "dev");
  await waitForAPI("/healthz", 200);
  await waitForWeb("/login", 200);

  process.stdout.write(`SysAP local: API em ${apiURL}\n`);
  process.stdout.write(`SysAP local: Web em ${webURL}\n`);
  process.stdout.write("SysAP local: pressione Ctrl+C para encerrar.\n");

  const outcome = await Promise.race([
    stopRequested,
    processes.completion(api).then((result) => ({ kind: "exit", ...result })),
    processes.completion(web).then((result) => ({ kind: "exit", ...result })),
  ]);
  if (outcome.kind === "exit") {
    throw new Error(`${outcome.label} encerrou inesperadamente`);
  }
}

try {
  await main();
} catch (error) {
  process.stderr.write(`SysAP local: falha segura: ${sanitizeMessage(error.message)}.\n`);
  process.exitCode = 1;
} finally {
  try {
    await processes.terminateAll();
    await database.stopIfOwned();
  } catch {
    process.stderr.write("SysAP local: a limpeza nao foi concluida integralmente.\n");
    process.exitCode = 1;
  }
  if (requestedSignal !== null && process.exitCode === undefined) {
    process.stdout.write(`SysAP local: encerrado apos ${requestedSignal}.\n`);
  }
}
