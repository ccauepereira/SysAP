import path from "node:path";

import { runCommand, runNodeScript } from "./run-command.mjs";

const root = path.resolve(import.meta.dirname, "..");
await runNodeScript(root, "test-api.mjs");
await runCommand("pnpm", ["--filter", "@sysap/web", "test"], { cwd: root, env: process.env });
await runNodeScript(root, "check-scripts.mjs");
