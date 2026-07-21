import path from "node:path";

import { runNodeScript } from "./run-command.mjs";

const root = path.resolve(import.meta.dirname, "..");
for (const script of ["check-api.mjs", "check-web.mjs", "openapi-lint.mjs", "check-scripts.mjs"]) {
  await runNodeScript(root, script);
}
