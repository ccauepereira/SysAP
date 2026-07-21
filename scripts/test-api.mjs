import { mkdir } from "node:fs/promises";
import path from "node:path";

import { runCommand } from "./run-command.mjs";

const root = path.resolve(import.meta.dirname, "..");
const goCache = path.join(root, ".sysap-runtime", "go-cache");
await mkdir(goCache, { recursive: true, mode: 0o700 });
await runCommand("go", ["test", "-race", "./..."], {
  cwd: path.join(root, "apps", "api"),
  env: { ...process.env, GOCACHE: goCache, GOFLAGS: "-buildvcs=false" },
});
