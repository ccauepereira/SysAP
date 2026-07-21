import path from "node:path";

import {
  OwnedProcesses,
  apiURL,
  assertPortAvailable,
  buildAPI,
  repositoryRoot,
  startAPI,
  waitForAPI,
} from "./local-runtime.mjs";
import { readDatabasePort } from "./local-database-url.mjs";
import { readLocalEnvironment, sanitizeMessage } from "./runtime-policy.mjs";

const processes = new OwnedProcesses();
let releaseStop;
const stopRequested = new Promise((resolve) => {
  releaseStop = resolve;
});
for (const signal of ["SIGINT", "SIGTERM"]) {
  process.once(signal, () => releaseStop({ kind: "signal" }));
}

try {
  await assertPortAvailable("127.0.0.1", 8080);
  const databasePort = readDatabasePort(path.join(repositoryRoot, "infra", "supabase", "config.toml"));
  const environment = readLocalEnvironment(path.join(repositoryRoot, ".env.local"), databasePort);
  const binary = await buildAPI("sysap-api-manual");
  const api = startAPI(processes, binary, environment.SYSAP_DATABASE_URL);
  await waitForAPI("/healthz", 200);
  process.stdout.write(`API SysAP: ${apiURL}; pressione Ctrl+C para encerrar.\n`);
  const outcome = await Promise.race([
    stopRequested,
    processes.completion(api).then((result) => ({ kind: "exit", ...result })),
  ]);
  if (outcome.kind === "exit") {
    throw new Error("API encerrou inesperadamente");
  }
} catch (error) {
  process.stderr.write(`API SysAP: falha segura: ${sanitizeMessage(error.message)}.\n`);
  process.exitCode = 1;
} finally {
  try {
    await processes.terminateAll();
  } catch {
    process.stderr.write("API SysAP: limpeza incompleta.\n");
    process.exitCode = 1;
  }
}
