import path from "node:path";

import { runCommandCapture } from "./run-command.mjs";

// Excecao temporaria e exata. Nao e allowlist: casa com um unico advisory, uma
// unica versao, um unico caminho de desenvolvimento e deixa de valer em
// expiresAt. Registro e evidencias: docs/security/accepted-risks.md.
export const temporaryException = Object.freeze({
  githubAdvisoryID: "GHSA-vfj7-8cjw-p6xm",
  moduleName: "braces",
  severity: "high",
  version: "3.0.3",
  path: "apps__web>eslint-config-next>@next/eslint-plugin-next>fast-glob>micromatch>braces",
  justification:
    "sem versao corrigida publicada; caminho somente de lint (dev), fora da arvore de producao e do build; " +
    "braces nao e chamado com a configuracao atual (sem settings.next.rootDir)",
  owner: "Caue (mantenedor do SysAP)",
  // Fim de 13/11/2026 em America/Fortaleza (UTC-03:00), expresso em UTC.
  expiresAt: "2026-11-14T03:00:00Z",
  record: "docs/security/accepted-risks.md",
});

export function evaluateAuditReport(report, now = new Date()) {
  if (report === null || typeof report !== "object" || Array.isArray(report)) {
    throw new Error("relatorio de dependencias invalido");
  }
  const advisories = report.advisories;
  if (advisories === null || typeof advisories !== "object" || Array.isArray(advisories)) {
    throw new Error("relatorio de dependencias sem advisories");
  }

  const entries = Object.values(advisories);
  if (entries.length === 0) {
    return { advisoryCount: 0, acceptedException: null };
  }
  if (entries.length !== 1 || !matchesTemporaryException(entries[0])) {
    throw new Error("vulnerabilidade nao autorizada encontrada");
  }
  // Data invalida resulta em NaN e falha fechada como expirada.
  if (!(now.getTime() < Date.parse(temporaryException.expiresAt))) {
    throw new Error(`excecao temporaria ${temporaryException.githubAdvisoryID} expirada`);
  }
  return { advisoryCount: 1, acceptedException: temporaryException };
}

function matchesTemporaryException(advisory) {
  if (advisory === null || typeof advisory !== "object") {
    return false;
  }
  if (
    advisory.github_advisory_id !== temporaryException.githubAdvisoryID ||
    advisory.module_name !== temporaryException.moduleName ||
    advisory.severity !== temporaryException.severity ||
    !Array.isArray(advisory.findings) ||
    advisory.findings.length !== 1
  ) {
    return false;
  }

  const finding = advisory.findings[0];
  return (
    finding !== null &&
    typeof finding === "object" &&
    finding.version === temporaryException.version &&
    finding.dev === true &&
    Array.isArray(finding.paths) &&
    finding.paths.length === 1 &&
    finding.paths[0] === temporaryException.path
  );
}

function reportAcceptedException(exception) {
  const lines = [
    "Dependencias: PASS COM EXCECAO TEMPORARIA; existe 1 vulnerabilidade conhecida aceita explicitamente.",
    `  advisory: ${exception.githubAdvisoryID} (severidade ${exception.severity})`,
    `  pacote: ${exception.moduleName}@${exception.version}`,
    `  caminho: ${exception.path}`,
    `  justificativa: ${exception.justification}`,
    `  responsavel: ${exception.owner}`,
    `  expira em: ${exception.expiresAt}`,
    `  registro: ${exception.record}`,
  ];
  process.stdout.write(`${lines.join("\n")}\n`);
  if (process.env.GITHUB_ACTIONS === "true") {
    process.stdout.write(
      `::warning title=Excecao temporaria de dependencia::${exception.githubAdvisoryID} em ` +
        `${exception.moduleName}@${exception.version} aceito ate ${exception.expiresAt}; ` +
        `responsavel ${exception.owner}; ver ${exception.record}\n`,
    );
  }
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
  if (decision.acceptedException !== null) {
    reportAcceptedException(decision.acceptedException);
    return;
  }
  process.stdout.write(
    "Dependencias: PASS; nenhuma vulnerabilidade encontrada. " +
      `A excecao temporaria ${temporaryException.githubAdvisoryID} nao e mais necessaria; remova-a.\n`,
  );
}

if (process.argv[1] === new URL(import.meta.url).pathname) {
  main().catch((error) => {
    // As mensagens deste fluxo sao fixas e nao contem dados do relatorio.
    process.stderr.write(`Dependencias: FAIL; auditoria recusou o estado atual: ${error.message}.\n`);
    process.exitCode = 1;
  });
}
