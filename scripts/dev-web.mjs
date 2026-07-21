import {
  OwnedProcesses,
  assertPortAvailable,
  startWeb,
  waitForWeb,
  webURL,
} from "./local-runtime.mjs";
import { sanitizeMessage } from "./runtime-policy.mjs";

const processes = new OwnedProcesses();
let releaseStop;
const stopRequested = new Promise((resolve) => {
  releaseStop = resolve;
});
for (const signal of ["SIGINT", "SIGTERM"]) {
  process.once(signal, () => releaseStop({ kind: "signal" }));
}

try {
  await assertPortAvailable("127.0.0.1", 3000);
  const web = startWeb(processes, "dev");
  await waitForWeb("Dados demonstrativos");
  process.stdout.write(`Web SysAP: ${webURL}; pressione Ctrl+C para encerrar.\n`);
  const outcome = await Promise.race([
    stopRequested,
    processes.completion(web).then((result) => ({ kind: "exit", ...result })),
  ]);
  if (outcome.kind === "exit") {
    throw new Error("Web encerrou inesperadamente");
  }
} catch (error) {
  process.stderr.write(`Web SysAP: falha segura: ${sanitizeMessage(error.message)}.\n`);
  process.exitCode = 1;
} finally {
  try {
    await processes.terminateAll();
  } catch {
    process.stderr.write("Web SysAP: limpeza incompleta.\n");
    process.exitCode = 1;
  }
}
