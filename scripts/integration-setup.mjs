import { repositoryRoot } from "./local-runtime.mjs";
import { runCommand } from "./run-command.mjs";

export async function initializeIntegrationDatabase(databaseSession, executeCommand = runCommand) {
  await databaseSession.start({ requireInactive: true });
  await databaseSession.resetAndLint();
  await executeCommand("pnpm", ["db:env"], { cwd: repositoryRoot, env: process.env });
  return databaseSession.environment();
}
