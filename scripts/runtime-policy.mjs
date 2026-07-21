import { lstatSync, readFileSync } from "node:fs";

import { validateLocalDatabaseURL } from "./local-database-url.mjs";

const allowedLoopbackHosts = new Set(["127.0.0.1", "localhost", "[::1]"]);
const environmentLine = /^(SYSAP_DATABASE_URL|SYSAP_TEST_DATABASE_URL)='([^']*)'$/;
const controlCharacters = /[\u0000-\u001f\u007f]/;
const sensitiveAssignment = /\b(password|passwd|secret|token|api[_-]?key|service_role)\s*[:=]\s*[^\s,;]+/gi;
const URLWithUserInfo = /\b(?:postgres|postgresql|https?):\/\/[^\s/@:]+:[^\s/@]+@[^\s]+/gi;
const jwtShape = /\beyJ[A-Za-z0-9_-]{8,}\.[A-Za-z0-9_-]{8,}\.[A-Za-z0-9_-]{8,}\b/g;

export function parseLocalEnvironment(contents, expectedDatabasePort) {
  if (typeof contents !== "string" || controlCharacters.test(contents.replaceAll("\n", ""))) {
    throw new Error("arquivo de ambiente local invalido");
  }

  const lines = contents.endsWith("\n")
    ? contents.slice(0, -1).split("\n")
    : contents.split("\n");
  if (lines.length !== 2) {
    throw new Error("arquivo de ambiente local deve conter duas variaveis");
  }

  const values = new Map();
  for (const line of lines) {
    const match = environmentLine.exec(line);
    if (match === null || values.has(match[1])) {
      throw new Error("arquivo de ambiente local possui formato nao autorizado");
    }
    values.set(match[1], validateLocalDatabaseURL(match[2], expectedDatabasePort));
  }

  const databaseURL = values.get("SYSAP_DATABASE_URL");
  const testDatabaseURL = values.get("SYSAP_TEST_DATABASE_URL");
  if (databaseURL === undefined || testDatabaseURL === undefined || databaseURL !== testDatabaseURL) {
    throw new Error("arquivo de ambiente local possui valores inconsistentes");
  }

  return { SYSAP_DATABASE_URL: databaseURL, SYSAP_TEST_DATABASE_URL: testDatabaseURL };
}

export function readLocalEnvironment(environmentPath, expectedDatabasePort) {
  const metadata = lstatSync(environmentPath);
  if (metadata.isSymbolicLink() || !metadata.isFile()) {
    throw new Error("arquivo de ambiente local deve ser um arquivo regular");
  }
  return parseLocalEnvironment(readFileSync(environmentPath, "utf8"), expectedDatabasePort);
}

export function validateLoopbackHTTPURL(value, expectedPort) {
  let parsed;
  try {
    parsed = new URL(value);
  } catch {
    throw new Error("URL HTTP local invalida");
  }
  if (parsed.protocol !== "http:" || !allowedLoopbackHosts.has(parsed.hostname)) {
    throw new Error("URL HTTP deve usar loopback sem TLS local");
  }
  if (parsed.username || parsed.password || parsed.hash || parsed.search) {
    throw new Error("URL HTTP local contem componentes proibidos");
  }
  if (parsed.port !== String(expectedPort) || parsed.pathname !== "/") {
    throw new Error("URL HTTP local possui porta ou caminho invalido");
  }
  return parsed.origin;
}

export function parseDatabaseStatus(stdout) {
  if (stdout === "Supabase local: running.\n") {
    return true;
  }
  if (stdout === "Supabase local: not running.\n") {
    return false;
  }
  throw new Error("estado do banco local nao reconhecido");
}

export function shouldStopDatabase(wasRunning, startCompleted) {
  return wasRunning === false && startCompleted === true;
}

export function canTerminateOwnedChild(child, ownedChildren) {
  return (
    child !== null &&
    typeof child === "object" &&
    Number.isSafeInteger(child.pid) &&
    child.pid > 1 &&
    ownedChildren.has(child)
  );
}

export function hasForbiddenEnvironmentName(name) {
  const normalized = name.toUpperCase();
  return (
    normalized.startsWith("NEXT_PUBLIC_") ||
    normalized === "SUPABASE_SERVICE_ROLE_KEY" ||
    normalized === "SUPABASE_SECRET_KEY" ||
    normalized === "SUPABASE_ACCESS_TOKEN"
  );
}

export function safeChildEnvironment(additional = {}) {
  const allowedNames = [
    "PATH",
    "HOME",
    "USER",
    "LOGNAME",
    "TMPDIR",
    "LANG",
    "LC_ALL",
    "TERM",
    "CI",
    "NO_COLOR",
    "FORCE_COLOR",
  ];
  const environment = {};
  for (const name of allowedNames) {
    if (process.env[name] !== undefined) {
      environment[name] = process.env[name];
    }
  }
  for (const [name, value] of Object.entries(additional)) {
    if (hasForbiddenEnvironmentName(name) || typeof value !== "string" || controlCharacters.test(value)) {
      throw new Error("variavel de ambiente de processo nao autorizada");
    }
    environment[name] = value;
  }
  return environment;
}

export function sanitizeMessage(message, sensitiveValues = []) {
  let sanitized = String(message);
  for (const value of sensitiveValues) {
    if (typeof value === "string" && value.length > 0) {
      sanitized = sanitized.replaceAll(value, "[redigido]");
    }
  }
  sanitized = sanitized
    .replace(URLWithUserInfo, "[URL redigida]")
    .replace(jwtShape, "[token redigido]")
    .replace(sensitiveAssignment, "$1=[redigido]");
  return sanitized;
}
