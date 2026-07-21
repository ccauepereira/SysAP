import { readFile } from "node:fs/promises";
import path from "node:path";

import { runCommandCapture } from "./run-command.mjs";

const requiredScripts = [
  "dev",
  "check",
  "check:api",
  "check:web",
  "openapi:lint",
  "test",
  "test:api",
  "test:web",
  "test:integration",
  "security:dependencies",
  "security:secrets",
  "db:start",
  "db:stop",
  "db:status",
  "db:reset",
  "db:lint",
  "db:env",
];

const approvedActions = new Map([
  ["actions/checkout", { sha: "11bd71901bbe5b1630ceea73d27597364c9af683", tag: "v4.2.2" }],
  ["actions/setup-node", { sha: "49933ea5288caeca8642d1e84afbd3f7d6820020", tag: "v4.4.0" }],
  ["actions/setup-go", { sha: "d35c59abb061a4a6fb18e82ac0862c26744d6ab5", tag: "v5.5.0" }],
  ["pnpm/action-setup", { sha: "7088e561eb65bb68695d245aa206f005ef30921d", tag: "v4.1.0" }],
]);

export function validatePackagePolicy(rootPackage, webPackage, workspaceText, goModuleText) {
  const errors = [];
  if (rootPackage.packageManager !== "pnpm@11.15.1") {
    errors.push("packageManager deve ser pnpm@11.15.1");
  }
  for (const script of requiredScripts) {
    if (typeof rootPackage.scripts?.[script] !== "string") {
      errors.push(`script raiz ausente: ${script}`);
    }
  }
  if (
    rootPackage.devDependencies?.supabase !== "2.109.1" ||
    rootPackage.devDependencies?.["@redocly/cli"] !== "2.39.0"
  ) {
    errors.push("ferramentas raiz devem usar as versoes aprovadas");
  }
  if (webPackage.engines?.node !== "24.18.0" || !workspaceText.includes("nodeVersion: 24.18.0")) {
    errors.push("Node deve estar fixado em 24.18.0");
  }
  if (!goModuleText.split("\n").includes("go 1.26.5")) {
    errors.push("Go deve estar fixado em 1.26.5");
  }

  for (const manifest of [rootPackage, webPackage]) {
    for (const group of ["dependencies", "devDependencies", "optionalDependencies"]) {
      for (const [name, version] of Object.entries(manifest[group] ?? {})) {
        if (
          typeof version !== "string" ||
          version === "latest" ||
          /^[~^]/.test(version) ||
          /^(?:git|https?|file|link):/i.test(version)
        ) {
          errors.push(`dependencia direta nao fixada: ${name}`);
        }
      }
    }
  }
  return errors;
}

export function validateWorkflowText(workflow) {
  const errors = [];
  for (const forbidden of [
    [/(^|\n)\s*pull_request_target\s*:/, "pull_request_target proibido"],
    [/permissions\s*:\s*write-all/i, "permissoes de escrita proibidas"],
    [/\b(?:contents|actions|checks|deployments|id-token|packages|pull-requests|statuses)\s*:\s*write\b/i, "permissao de escrita proibida"],
    [/\$\{\{\s*secrets\./, "segredos do GitHub proibidos"],
    [/\b(?:supabase\s+(?:login|link)|db\s+(?:push|pull)|deploy)\b/i, "operacao remota ou deploy proibido"],
    [/@latest\b|:\s*latest\b/i, "latest proibido"],
    [/runs-on:\s*self-hosted/i, "runner self-hosted proibido"],
  ]) {
    if (forbidden[0].test(workflow)) {
      errors.push(forbidden[1]);
    }
  }
  for (const required of [
    "push:",
    "pull_request:",
    "workflow_dispatch:",
    "permissions:\n  contents: read",
    "runs-on: ubuntu-24.04",
    "persist-credentials: false",
    "fetch-depth: 0",
  ]) {
    if (!workflow.includes(required)) {
      errors.push(`configuracao obrigatoria ausente: ${required}`);
    }
  }

  const usesLines = workflow.split("\n").filter((line) => line.includes("uses:"));
  for (const line of usesLines) {
    const match = /uses:\s*([^@\s]+)@([0-9a-f]{40})\s*#\s*(v[^\s]+)/.exec(line);
    if (match === null) {
      errors.push("Action sem SHA completo e comentario de tag");
      continue;
    }
    const approved = approvedActions.get(match[1]);
    if (approved === undefined || approved.sha !== match[2] || approved.tag !== match[3]) {
      errors.push(`Action nao aprovada: ${match[1]}`);
    }
  }
  if (usesLines.length === 0) {
    errors.push("workflow sem Actions fixadas");
  }
  return errors;
}

export function validateRepositoryPaths(paths) {
  const forbidden = paths.filter((file) => {
    const baseName = path.posix.basename(file);
    const isRealEnvironment = baseName === ".env" || (baseName.startsWith(".env.") && !baseName.endsWith(".example"));
    return (
      /(^|\/)(?:node_modules|\.next|coverage)(\/|$)/i.test(file) ||
      isRealEnvironment ||
      /\.(?:log|pem|key|dump|backup|bak|map)$/i.test(file)
    );
  });
  return forbidden.map((file) => `arquivo proibido versionavel: ${file}`);
}

async function main() {
  const root = path.resolve(import.meta.dirname, "..");
  const [rootPackage, webPackage, workspaceText, goModuleText, workflow, files] = await Promise.all([
    readFile(path.join(root, "package.json"), "utf8").then(JSON.parse),
    readFile(path.join(root, "apps", "web", "package.json"), "utf8").then(JSON.parse),
    readFile(path.join(root, "pnpm-workspace.yaml"), "utf8"),
    readFile(path.join(root, "apps", "api", "go.mod"), "utf8"),
    readFile(path.join(root, ".github", "workflows", "ci.yml"), "utf8"),
    runCommandCapture("git", ["ls-files", "--cached", "--others", "--exclude-standard", "-z"], {
      cwd: root,
      env: process.env,
    }).then((result) => result.stdout.split("\0").filter(Boolean)),
  ]);
  const errors = [
    ...validatePackagePolicy(rootPackage, webPackage, workspaceText, goModuleText),
    ...validateWorkflowText(workflow),
    ...validateRepositoryPaths(files),
  ];
  if (errors.length > 0) {
    throw new Error(errors.join("; "));
  }
  process.stdout.write("Politica do repositorio: PASS.\n");
}

if (process.argv[1] === new URL(import.meta.url).pathname) {
  main().catch(() => {
    process.stderr.write("Politica do repositorio: FAIL.\n");
    process.exitCode = 1;
  });
}
