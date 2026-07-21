import { mkdir } from "node:fs/promises";
import path from "node:path";

import { runCommand, runCommandCapture } from "./run-command.mjs";

const root = path.resolve(import.meta.dirname, "..");
const apiDirectory = path.join(root, "apps", "api");
const goCache = path.join(root, ".sysap-runtime", "go-cache");
await mkdir(goCache, { recursive: true, mode: 0o700 });
const environment = { ...process.env, GOCACHE: goCache, GOFLAGS: "-buildvcs=false" };

const formatting = await runCommandCapture("gofmt", ["-l", "."], {
  cwd: apiDirectory,
  env: environment,
});
if (formatting.stdout.trim() !== "") {
  process.stderr.write("API: ha arquivos Go fora do formato esperado.\n");
  process.exit(1);
}

await runCommand("go", ["mod", "verify"], { cwd: apiDirectory, env: environment });
await runCommand("go", ["vet", "./..."], { cwd: apiDirectory, env: environment });
await runCommand("go", ["test", "-race", "./..."], { cwd: apiDirectory, env: environment });
await runCommand("go", ["build", "./..."], { cwd: apiDirectory, env: environment });
