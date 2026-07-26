import path from "node:path";

import { runCommandCapture } from "./run-command.mjs";

export function evaluateAuditReport(report) {
  if (report === null || typeof report !== "object" || Array.isArray(report)) {
    throw new Error("relatorio de dependencias invalido");
  }
  const advisories = report.advisories;
  if (advisories === null || typeof advisories !== "object" || Array.isArray(advisories)) {
    throw new Error("relatorio de dependencias sem advisories");
  }

  const entries = Object.values(advisories);
  if (entries.length > 0) {
    throw new Error("vulnerabilidade nao autorizada encontrada");
  }
  return { advisoryCount: 0 };
}

async function main() {
  const root = path.resolve(import.meta.dirname, "..");
  const result = await runCommandCapture("pnpm", ["audit", "--json"], {
    cwd: root,
    env: process.env,
    allowFailure: true,
    outputLimit: 8 * 1024 * 1024,
  });

  let report;
  try {
    report = JSON.parse(result.stdout);
  } catch {
    throw new Error("pnpm audit nao produziu JSON valido");
  }
  evaluateAuditReport(report);
  process.stdout.write("Dependencias: PASS; nenhuma vulnerabilidade encontrada.\n");
}

if (process.argv[1] === new URL(import.meta.url).pathname) {
  main().catch(() => {
    process.stderr.write("Dependencias: FAIL; auditoria recusou o estado atual.\n");
    process.exitCode = 1;
  });
}
