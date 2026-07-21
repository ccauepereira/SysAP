import path from "node:path";

import { runCommandCapture } from "./run-command.mjs";

const acceptedAdvisory = Object.freeze({
  githubAdvisoryID: "GHSA-qx2v-qp2m-jg93",
  moduleName: "postcss",
  severity: "moderate",
  version: "8.4.31",
  path: "apps__web>next>postcss",
});

export function evaluateAuditReport(report) {
  if (report === null || typeof report !== "object" || Array.isArray(report)) {
    throw new Error("relatorio de dependencias invalido");
  }
  const advisories = report.advisories;
  if (advisories === null || typeof advisories !== "object" || Array.isArray(advisories)) {
    throw new Error("relatorio de dependencias sem advisories");
  }

  const entries = Object.values(advisories);
  if (entries.length === 0) {
    return { acceptedPostCSSRisk: false, advisoryCount: 0 };
  }
  if (entries.length !== 1 || !isAcceptedPostCSSAdvisory(entries[0])) {
    throw new Error("vulnerabilidade nao autorizada encontrada");
  }
  return { acceptedPostCSSRisk: true, advisoryCount: 1 };
}

function isAcceptedPostCSSAdvisory(advisory) {
  if (advisory === null || typeof advisory !== "object") {
    return false;
  }
  if (
    advisory.github_advisory_id !== acceptedAdvisory.githubAdvisoryID ||
    advisory.module_name !== acceptedAdvisory.moduleName ||
    advisory.severity !== acceptedAdvisory.severity ||
    !Array.isArray(advisory.findings) ||
    advisory.findings.length !== 1
  ) {
    return false;
  }

  const finding = advisory.findings[0];
  return (
    finding !== null &&
    typeof finding === "object" &&
    finding.version === acceptedAdvisory.version &&
    Array.isArray(finding.paths) &&
    finding.paths.length === 1 &&
    finding.paths[0] === acceptedAdvisory.path
  );
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
  const decision = evaluateAuditReport(report);
  if (decision.acceptedPostCSSRisk) {
    process.stdout.write("Dependencias: PASS; risco moderado PostCSS documentado e isolado.\n");
  } else {
    process.stdout.write("Dependencias: PASS; a excecao PostCSS nao e mais necessaria.\n");
  }
}

if (process.argv[1] === new URL(import.meta.url).pathname) {
  main().catch(() => {
    process.stderr.write("Dependencias: FAIL; auditoria recusou o estado atual.\n");
    process.exitCode = 1;
  });
}
