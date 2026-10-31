import path from "node:path";

import {
  LocalDatabaseSession,
  OwnedProcesses,
  apiDirectory,
  assertPortAvailable,
  buildAPI,
  buildWeb,
  runtimeDirectory,
  startAPI,
  startWeb,
  waitForAPI,
  waitForWeb,
} from "./local-runtime.mjs";
import {
  initializeIntegrationDatabase,
  integrationAPIEnvironment,
} from "./integration-setup.mjs";
import { assertNoSensitiveRuntimeDetails } from "./integration-contracts.mjs";
import { runCommand } from "./run-command.mjs";
import { safeChildEnvironment, sanitizeMessage } from "./runtime-policy.mjs";

const database = new LocalDatabaseSession();
const processes = new OwnedProcesses();

function assertExactAPIContract(result, expectedBody) {
  if (result.text !== expectedBody) {
    throw new Error("API local divergiu do contrato HTTP exato");
  }
  if (
    result.response.headers.get("content-type") !== "application/json; charset=utf-8" ||
    result.response.headers.get("cache-control") !== "no-store" ||
    !/^[0-9a-f]{32,}$/.test(result.response.headers.get("x-request-id") ?? "")
  ) {
    throw new Error("API local divergiu dos headers HTTP obrigatorios");
  }
}

async function main() {
  await assertPortAvailable("127.0.0.1", 8080);
  await assertPortAvailable("127.0.0.1", 3000);
  const environment = await initializeIntegrationDatabase(database);

  const goCache = path.join(runtimeDirectory, "go-cache");
  // API integration packages use one local PostgreSQL instance and create
  // independent fixtures. Run packages serially so fixture cleanup in one
  // package cannot overlap another package's assertions.
  await runCommand("go", ["test", "-race", "-p", "1", "-v", "./..."], {
    cwd: apiDirectory,
    env: safeChildEnvironment({
      GOCACHE: goCache,
      GOFLAGS: "-buildvcs=false",
      SYSAP_TEST_DATABASE_URL: environment.SYSAP_TEST_DATABASE_URL,
    }),
  });

  const binary = await buildAPI("sysap-api-integration");
  await buildWeb();
  const api = startAPI(processes, binary, integrationAPIEnvironment(environment));
  const web = startWeb(processes, "start");

  const health = await waitForAPI("/healthz", 200);
  assertExactAPIContract(health, '{"status":"ok","service":"sysap-api"}\n');
  assertNoSensitiveRuntimeDetails(health.text, environment.SYSAP_DATABASE_URL);
  const ready = await waitForAPI("/readyz", 200);
  assertExactAPIContract(
    ready,
    '{"status":"ready","service":"sysap-api","checks":{"database":"up"}}\n',
  );
  assertNoSensitiveRuntimeDetails(ready.text, environment.SYSAP_DATABASE_URL);
  const webReady = await waitForWeb("/login", 200);
  assertNoSensitiveRuntimeDetails(webReady.text, environment.SYSAP_DATABASE_URL);

  await database.stopIfOwned();
  const unavailable = await waitForAPI("/readyz", 503);
  assertExactUnavailableAPIContract(unavailable);
  assertNoSensitiveRuntimeDetails(unavailable.text, environment.SYSAP_DATABASE_URL);
  const webWithoutDatabase = await waitForWeb("/login", 200);
  assertNoSensitiveRuntimeDetails(webWithoutDatabase.text, environment.SYSAP_DATABASE_URL);

  await processes.terminate(api);
  const webWithoutAPI = await waitForWeb("/login", 200);
  assertNoSensitiveRuntimeDetails(webWithoutAPI.text, environment.SYSAP_DATABASE_URL);
  await processes.terminate(web);

  process.stdout.write(
    "Integracao: PASS; GET /login respondeu 200 com API e banco disponíveis, com banco indisponível e com API parada; API /healthz 200 e /readyz 200→503 validados. A interface não possui rota pública com contrato observável de degradação do backend; essa cobertura permanece pendente.\n",
  );
}

function assertExactUnavailableAPIContract(result) {
  const body = JSON.parse(result.text);
  const requestID = result.response.headers.get("x-request-id") ?? "";
  if (
    result.response.headers.get("content-type") !== "application/json; charset=utf-8" ||
    result.response.headers.get("cache-control") !== "no-store" ||
    !/^[0-9a-f]{32,}$/.test(requestID) ||
    body?.error?.code !== "service_not_ready" ||
    body?.error?.message !== "service is not ready" ||
    body?.error?.request_id !== requestID
  ) {
    throw new Error("readiness degradada divergiu do contrato seguro");
  }
}

try {
  await main();
} catch (error) {
  process.stderr.write(`Integracao: FAIL; ${sanitizeMessage(error.message)}.\n`);
  process.exitCode = 1;
} finally {
  try {
    await processes.terminateAll();
    await database.stopIfOwned();
  } catch {
    process.stderr.write("Integracao: FAIL; limpeza local incompleta.\n");
    process.exitCode = 1;
  }
}
