import { spawn } from "node:child_process";
import {
  copyFile,
  lstat,
  mkdir,
  readFile,
  rm,
} from "node:fs/promises";
import path from "node:path";

import { runCommandCapture } from "./run-command.mjs";

export const gitleaksImage =
  "ghcr.io/gitleaks/gitleaks@sha256:c00b6bd0aeb3071cbcb79009cb16a60dd9e0a7c60e2be9ab65d25e6bc8abbb7f";

export function parseGitleaksReport(contents) {
  let report;
  try {
    report = JSON.parse(contents);
  } catch {
    throw new Error("relatorio Gitleaks invalido");
  }
  if (!Array.isArray(report)) {
    throw new Error("relatorio Gitleaks deve ser uma lista");
  }
  return report.map((finding) => {
    if (
      finding === null ||
      typeof finding !== "object" ||
      typeof finding.RuleID !== "string" ||
      typeof finding.File !== "string"
    ) {
      throw new Error("achado Gitleaks invalido");
    }
    return { rule: finding.RuleID, file: finding.File };
  });
}

export function safeFindingDescription(finding) {
  const rule = finding.rule.replace(/[^a-zA-Z0-9._-]/g, "?").slice(0, 80);
  const normalized = finding.file.replaceAll("\\", "/").replace(/[^a-zA-Z0-9._/-]/g, "?");
  const file = normalized
    .split("/")
    .filter((part) => part !== "" && part !== "." && part !== "..")
    .join("/")
    .slice(0, 300);
  return `${rule}: ${file}`;
}

async function copyCurrentRepository(root, destination) {
  const listed = await runCommandCapture(
    "git",
    ["ls-files", "--cached", "--others", "--exclude-standard", "-z"],
    { cwd: root, env: process.env },
  );
  const files = listed.stdout.split("\0").filter(Boolean);
  for (const relativePath of files) {
    if (path.isAbsolute(relativePath) || relativePath.split("/").includes("..")) {
      throw new Error("Git retornou caminho inseguro");
    }
    const source = path.resolve(root, relativePath);
    if (!source.startsWith(`${root}${path.sep}`)) {
      throw new Error("Git retornou caminho fora do repositorio");
    }
    const metadata = await lstat(source);
    if (!metadata.isFile() || metadata.isSymbolicLink()) {
      throw new Error("snapshot de segredos aceita somente arquivos regulares");
    }
    const target = path.resolve(destination, relativePath);
    if (!target.startsWith(`${destination}${path.sep}`)) {
      throw new Error("destino inseguro no snapshot de segredos");
    }
    await mkdir(path.dirname(target), { recursive: true, mode: 0o700 });
    await copyFile(source, target);
  }
}

async function runGitleaks(source, mode, reportPath, root) {
  const uid = process.getuid?.();
  const gid = process.getgid?.();
  if (!Number.isSafeInteger(uid) || !Number.isSafeInteger(gid)) {
    throw new Error("UID/GID local indisponivel para o scanner");
  }
  const outputDirectory = path.dirname(reportPath);
  const containerSource = mode === "git" ? "/repository" : "/snapshot";
  const args = [
    "run",
    "--rm",
    "--network",
    "none",
    "--user",
    `${uid}:${gid}`,
    "--volume",
    `${source}:${containerSource}:ro`,
    "--volume",
    `${outputDirectory}:/output`,
    gitleaksImage,
    mode,
    "--no-banner",
    "--no-color",
    "--redact",
    "--report-format",
    "json",
    "--report-path",
    `/output/${path.basename(reportPath)}`,
    containerSource,
  ];

  const result = await new Promise((resolve, reject) => {
    const child = spawn("docker", args, {
      cwd: root,
      env: process.env,
      shell: false,
      stdio: ["ignore", "pipe", "pipe"],
    });
    let outputBytes = 0;
    const count = (chunk) => {
      outputBytes += chunk.length;
      if (outputBytes > 1024 * 1024) {
        child.kill("SIGTERM");
      }
    };
    child.stdout.on("data", count);
    child.stderr.on("data", count);
    child.once("error", () => reject(new Error("nao foi possivel iniciar Docker para Gitleaks")));
    child.once("close", (code) => resolve({ code: code ?? 1, outputBytes }));
  });

  let reportContents;
  try {
    reportContents = await readFile(reportPath, "utf8");
  } catch {
    throw new Error("Gitleaks nao gerou relatorio redigido");
  }
  const findings = parseGitleaksReport(reportContents);
  if (result.outputBytes > 1024 * 1024) {
    throw new Error("Gitleaks excedeu o limite de saida");
  }
  if (findings.length > 0 || result.code !== 0) {
    const details = findings.map(safeFindingDescription).join(", ");
    throw new Error(details === "" ? "Gitleaks falhou sem achado classificavel" : details);
  }
}

async function main() {
  const root = path.resolve(import.meta.dirname, "..");
  const runtimeRoot = path.join(root, ".sysap-runtime", "secret-scan");
  if (!runtimeRoot.startsWith(`${root}${path.sep}.sysap-runtime${path.sep}`)) {
    throw new Error("diretorio temporario de seguranca invalido");
  }
  await rm(runtimeRoot, { recursive: true, force: true });
  await mkdir(runtimeRoot, { recursive: true, mode: 0o700 });

  try {
    const snapshot = path.join(runtimeRoot, "current");
    const reports = path.join(runtimeRoot, "reports");
    await mkdir(snapshot, { recursive: true, mode: 0o700 });
    await mkdir(reports, { recursive: true, mode: 0o700 });
    await copyCurrentRepository(root, snapshot);
    await runGitleaks(snapshot, "dir", path.join(reports, "current.json"), root);
    await runGitleaks(root, "git", path.join(reports, "history.json"), root);
    process.stdout.write("Segredos: PASS; estado atual e historico verificados com saida redigida.\n");
  } finally {
    await rm(runtimeRoot, { recursive: true, force: true });
  }
}

if (process.argv[1] === new URL(import.meta.url).pathname) {
  main().catch((error) => {
    process.stderr.write(`Segredos: FAIL; ${error.message}\n`);
    process.exitCode = 1;
  });
}
