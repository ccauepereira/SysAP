import { spawn } from "node:child_process";
import path from "node:path";
import { chmod, rm } from "node:fs/promises";

import {
  apiDirectory,
  buildAPI,
  prepareRuntimeDirectory,
  repositoryRoot,
  runtimeDirectory,
  readSupabaseEnvironment,
} from "./local-runtime.mjs";
import { readDatabasePort } from "./local-database-url.mjs";
import { readLocalEnvironment, safeChildEnvironment, sanitizeMessage } from "./runtime-policy.mjs";
import { runCommandCapture } from "./run-command.mjs";

async function main() {
  try {
    // 1. Get database URL from .env.local
    const databasePort = readDatabasePort(path.join(repositoryRoot, "infra", "supabase", "config.toml"));
    const localEnv = readLocalEnvironment(path.join(repositoryRoot, ".env.local"), databasePort);

    const supabaseEnv = await readSupabaseEnvironment();

    // 3. Compile the bootstrap-owner command
    await prepareRuntimeDirectory();
    const outputPath = path.join(runtimeDirectory, "sysap-bootstrap-owner");
    await rm(outputPath, { force: true });
    
    const goCache = path.join(runtimeDirectory, "go-cache");
    const goBuild = spawn("go", ["build", "-o", outputPath, "./cmd/bootstrap-owner"], {
      cwd: apiDirectory,
      env: safeChildEnvironment({ GOCACHE: goCache, GOFLAGS: "-buildvcs=false" }),
      stdio: "inherit"
    });

    const buildCode = await new Promise((resolve) => {
      goBuild.on("close", resolve);
    });

    if (buildCode !== 0) {
      throw new Error("failed to build bootstrap-owner command");
    }

    await chmod(outputPath, 0o700);

    // 4. Run the compiled command in interactive mode
    const child = spawn(outputPath, [], {
      cwd: apiDirectory,
      env: safeChildEnvironment({
        SYSAP_ENV: "development",
        SYSAP_DATABASE_URL: localEnv.SYSAP_DATABASE_URL,
        SYSAP_SUPABASE_AUTH_URL: `${supabaseEnv.apiURL}/auth/v1`,
        SYSAP_SUPABASE_SERVICE_ROLE_KEY: supabaseEnv.serviceRoleKey,
      }),
      stdio: "inherit"
    });

    const code = await new Promise((resolve) => {
      child.on("close", resolve);
    });

    process.exitCode = code;

  } catch (error) {
    process.stderr.write(`Owner Bootstrap: falha segura: ${sanitizeMessage(error.message)}.\n`);
    process.exitCode = 1;
  }
}

await main();
