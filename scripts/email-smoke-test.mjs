import { spawn } from "node:child_process";
import { fileURLToPath } from "node:url";
import path from "node:path";
import { chmod, rm } from "node:fs/promises";
import { readFileSync } from "node:fs";

import {
  apiDirectory,
  prepareRuntimeDirectory,
  repositoryRoot,
  runtimeDirectory,
} from "./local-runtime.mjs";
import { safeChildEnvironment } from "./runtime-policy.mjs";

export function parseSmokeTestEnv(envContent) {
  const lines = envContent.split("\n");
  const localEnv = {};
  for (const line of lines) {
    let trimmed = line.trim();
    if (trimmed.startsWith("export ")) {
      trimmed = trimmed.substring(7).trim();
    }
    if (!trimmed || trimmed.startsWith('#')) continue;
    const match = /^([A-Za-z_]+)\s*=\s*(.*)$/.exec(trimmed);
    if (match) {
      let value = match[2].trim();
      if ((value.startsWith('"') && value.endsWith('"')) || (value.startsWith("'") && value.endsWith("'"))) {
        value = value.slice(1, -1);
      }
      localEnv[match[1]] = value;
    }
  }

  return localEnv;
}

async function main() {
  try {
    const envPath = path.join(repositoryRoot, ".env.local");
    let envContent = "";
    try {
      envContent = readFileSync(envPath, "utf8");
    } catch (e) {
      throw new Error("Não foi possível ler .env.local");
    }

    const localEnv = parseSmokeTestEnv(envContent);

    await prepareRuntimeDirectory();
    const outputPath = path.join(runtimeDirectory, "sysap-email-smoke-test");
    await rm(outputPath, { force: true });
    
    const goCache = path.join(runtimeDirectory, "go-cache");
    const goBuild = spawn("go", ["build", "-o", outputPath, "./cmd/email-smoke-test"], {
      cwd: apiDirectory,
      env: safeChildEnvironment({ GOCACHE: goCache, GOFLAGS: "-buildvcs=false" }),
      stdio: "inherit"
    });

    const buildCode = await new Promise((resolve) => {
      goBuild.on("close", resolve);
    });

    if (buildCode !== 0) {
      throw new Error("falha ao compilar comando email-smoke-test");
    }

    await chmod(outputPath, 0o700);

    const childEnv = safeChildEnvironment({
      SYSAP_EMAIL_PROVIDER: localEnv.SYSAP_EMAIL_PROVIDER || "",
      SYSAP_BREVO_API_KEY: localEnv.SYSAP_BREVO_API_KEY || "",
      SYSAP_EMAIL_FROM: localEnv.SYSAP_EMAIL_FROM || "",
      SYSAP_EMAIL_FROM_NAME: localEnv.SYSAP_EMAIL_FROM_NAME || "",
      SYSAP_EMAIL_SMOKE_TEST_TO: localEnv.SYSAP_EMAIL_SMOKE_TEST_TO || "",
    });

    const child = spawn(outputPath, [], {
      cwd: apiDirectory,
      env: childEnv,
      stdio: "inherit"
    });

    const code = await new Promise((resolve) => {
      child.on("close", resolve);
    });

    process.exitCode = code;
  } catch (error) {
    process.stderr.write(`Email Smoke Test: falha segura.\n`);
    process.exitCode = 1;
  }
}

if (process.argv[1] === fileURLToPath(import.meta.url)) {
  await main();
}
