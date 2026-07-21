import { readFileSync } from "node:fs";
import { fileURLToPath } from "node:url";
import { resolve } from "node:path";

const allowedHostnames = new Set(["localhost", "127.0.0.1", "[::1]"]);
const allowedProtocols = new Set(["postgres:", "postgresql:"]);
const expectedDatabasePath = "/postgres";
const unsafeShellCharacters = /[$`'"\\]/;
const controlCharacters = /[\u0000-\u001f\u007f]/;

export function readDatabasePort(configPath) {
  const contents = readFileSync(configPath, "utf8");
  let inDatabaseSection = false;

  for (const line of contents.split("\n")) {
    const trimmed = line.trim();
    if (trimmed.startsWith("[") && trimmed.endsWith("]")) {
      inDatabaseSection = trimmed === "[db]";
      continue;
    }

    if (inDatabaseSection) {
      const match = /^port\s*=\s*([0-9]+)\s*$/.exec(trimmed);
      if (match) {
        return Number.parseInt(match[1], 10);
      }
    }
  }

  throw new Error("database port is not configured");
}

export function validateLocalDatabaseURL(input, expectedPort) {
  if (typeof input !== "string" || input.length === 0) {
    throw new Error("database URL is empty");
  }

  let candidate = input;
  if (candidate.startsWith('"') || candidate.endsWith('"')) {
    if (!(candidate.startsWith('"') && candidate.endsWith('"'))) {
      throw new Error("database URL quoting is invalid");
    }
    candidate = JSON.parse(candidate);
  }

  assertSafeText(candidate);
  if (/\s/.test(candidate)) {
    throw new Error("database URL contains whitespace");
  }

  let parsed;
  try {
    parsed = new URL(candidate);
  } catch {
    throw new Error("database URL is invalid");
  }

  if (!allowedProtocols.has(parsed.protocol)) {
    throw new Error("database URL protocol is invalid");
  }
  if (!allowedHostnames.has(parsed.hostname)) {
    throw new Error("database URL host is not loopback");
  }
  if (parsed.port !== String(expectedPort)) {
    throw new Error("database URL port is invalid");
  }
  if (parsed.username.length === 0) {
    throw new Error("database URL username is missing");
  }
  if (parsed.password.length === 0) {
    throw new Error("database URL password is missing");
  }
  if (parsed.hash !== "") {
    throw new Error("database URL fragment is not allowed");
  }

  const decodedUsername = decodeComponent(parsed.username);
  const decodedPassword = decodeComponent(parsed.password);
  const decodedPath = decodeComponent(parsed.pathname);
  const decodedSearch = decodeComponent(parsed.search);
  assertSafeText(decodedUsername);
  assertSafeText(decodedPassword);
  assertSafeText(decodedPath);
  assertSafeText(decodedSearch);

  if (decodedPath !== expectedDatabasePath) {
    throw new Error("database URL database name is invalid");
  }

  return parsed.href;
}

export function renderEnvironment(databaseURL) {
  assertSafeText(databaseURL);
  return [
    `SYSAP_DATABASE_URL='${databaseURL}'`,
    `SYSAP_TEST_DATABASE_URL='${databaseURL}'`,
    "",
  ].join("\n");
}

function decodeComponent(value) {
  try {
    return decodeURIComponent(value);
  } catch {
    throw new Error("database URL encoding is invalid");
  }
}

function assertSafeText(value) {
  if (controlCharacters.test(value)) {
    throw new Error("database URL contains a control character");
  }
  if (unsafeShellCharacters.test(value)) {
    throw new Error("database URL contains a shell metacharacter");
  }
}

async function readStandardInput() {
  const chunks = [];
  for await (const chunk of process.stdin) {
    chunks.push(chunk);
  }
  return Buffer.concat(chunks).toString("utf8");
}

async function main() {
  try {
    const configPath = process.argv[2];
    if (!configPath) {
      throw new Error("configuration path is missing");
    }
    const expectedPort = readDatabasePort(configPath);
    const input = await readStandardInput();
    const databaseURL = validateLocalDatabaseURL(input, expectedPort);
    process.stdout.write(renderEnvironment(databaseURL));
  } catch {
    process.stderr.write("Local database URL validation failed.\n");
    process.exitCode = 1;
  }
}

const invokedPath = process.argv[1] ? resolve(process.argv[1]) : "";
if (invokedPath === fileURLToPath(import.meta.url)) {
  await main();
}
