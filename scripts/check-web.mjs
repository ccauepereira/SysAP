import path from "node:path";

import { runCommand } from "./run-command.mjs";

const root = path.resolve(import.meta.dirname, "..");
for (const script of ["lint", "typecheck", "test", "build"]) {
  await runCommand("pnpm", ["--filter", "@sysap/web", script], {
    cwd: root,
    env: process.env,
  });
}
