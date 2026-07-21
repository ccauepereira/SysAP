import path from "node:path";

import { runCommand } from "./run-command.mjs";

const root = path.resolve(import.meta.dirname, "..");
await runCommand(
  "pnpm",
  ["exec", "redocly", "lint", "contracts/openapi/openapi.yaml", "--config", "redocly.yaml"],
  {
    cwd: root,
    env: {
      ...process.env,
      REDOCLY_TELEMETRY: "off",
      REDOCLY_SUPPRESS_UPDATE_NOTICE: "true",
      NO_UPDATE_NOTIFIER: "1",
    },
  },
);
