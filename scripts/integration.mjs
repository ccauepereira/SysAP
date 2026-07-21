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

function assertNoConnectionDetails(text, databaseURL) {
  const parsed = new URL(databaseURL);
  const forbidden = [databaseURL, parsed.host, parsed.username, parsed.password].filter(Boolean);
  if (forbidden.some((value) => text.includes(value))) {
    throw new Error("resposta local expos detalhes de conexao");
  }
}

async function main() {
  await assertPortAvailable("127.0.0.1", 8080);
  await assertPortAvailable("127.0.0.1", 3000);
  await database.start({ requireInactive: true });
  await database.resetAndLint();
  const environment = await database.environment();

  const goCache = path.join(runtimeDirectory, "go-cache");
  await runCommand("go", ["test", "-race", "-v", "./..."], {
    cwd: apiDirectory,
    env: safeChildEnvironment({
      GOCACHE: goCache,
      GOFLAGS: "-buildvcs=false",
      SYSAP_TEST_DATABASE_URL: environment.SYSAP_TEST_DATABASE_URL,
    }),
  });

  const binary = await buildAPI("sysap-api-integration");
  await buildWeb();
  const api = startAPI(processes, binary, environment.SYSAP_DATABASE_URL, "test");
  const web = startWeb(processes, "start");

  const health = await waitForAPI("/healthz", 200);
  assertExactAPIContract(health, '{"status":"ok","service":"sysap-api"}\n');
  const ready = await waitForAPI("/readyz", 200);
  assertExactAPIContract(
    ready,
    '{"status":"ready","service":"sysap-api","checks":{"database":"up"}}\n',
  );
  const webReady = await waitForWeb("API online · banco pronto");
  assertNoConnectionDetails(webReady.text, environment.SYSAP_DATABASE_URL);

  await database.stopIfOwned();
  const unavailable = await waitForAPI("/readyz", 503);
  const unavailableBody = JSON.parse(unavailable.text);
  if (
    unavailableBody?.error?.code !== "service_not_ready" ||
    unavailableBody?.error?.message !== "service is not ready" ||
    unavailableBody?.error?.request_id !== unavailable.response.headers.get("x-request-id")
  ) {
    throw new Error("readiness degradada divergiu do contrato seguro");
  }
  assertNoConnectionDetails(unavailable.text, environment.SYSAP_DATABASE_URL);
  const webWithoutDatabase = await waitForWeb("API online · banco indisponível");
  assertNoConnectionDetails(webWithoutDatabase.text, environment.SYSAP_DATABASE_URL);

  await processes.terminate(api);
  const webWithoutAPI = await waitForWeb("API indisponível");
  assertNoConnectionDetails(webWithoutAPI.text, environment.SYSAP_DATABASE_URL);
  await processes.terminate(web);

  process.stdout.write("Integracao: PASS; Web, API e PostgreSQL validados, inclusive degradacao segura.\n");
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
