import { readdir } from "node:fs/promises";
import path from "node:path";

import { runCommand } from "./run-command.mjs";

const root = path.resolve(import.meta.dirname, "..");
await runCommand("bash", ["-n", "scripts/supabase-local.sh"], { cwd: root, env: process.env });
await runCommand(process.execPath, ["scripts/repository-policy.mjs"], { cwd: root, env: process.env });

const testFiles = (await readdir(path.join(root, "scripts")))
  .filter((name) => name.endsWith(".test.mjs"))
  .sort()
  .map((name) => path.join("scripts", name));
await runCommand(process.execPath, ["--test", ...testFiles], { cwd: root, env: process.env });
