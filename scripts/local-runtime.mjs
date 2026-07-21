import { spawn } from "node:child_process";
import { createRequire } from "node:module";
import net from "node:net";
import { chmod, lstat, mkdir, rm } from "node:fs/promises";
import path from "node:path";

import { readDatabasePort } from "./local-database-url.mjs";
import { runCommand, runCommandCapture } from "./run-command.mjs";
import {
  canTerminateOwnedChild,
  parseDatabaseStatus,
  readLocalEnvironment,
  safeChildEnvironment,
  shouldStopDatabase,
  validateLoopbackHTTPURL,
} from "./runtime-policy.mjs";

export const repositoryRoot = path.resolve(import.meta.dirname, "..");
export const apiDirectory = path.join(repositoryRoot, "apps", "api");
export const webDirectory = path.join(repositoryRoot, "apps", "web");
export const runtimeDirectory = path.join(repositoryRoot, ".sysap-runtime");
export const apiURL = validateLoopbackHTTPURL("http://127.0.0.1:8080", 8080);
export const webURL = validateLoopbackHTTPURL("http://127.0.0.1:3000", 3000);

const require = createRequire(import.meta.url);
const nextBinary = require.resolve("next/dist/bin/next", { paths: [webDirectory] });

export async function prepareRuntimeDirectory() {
  await mkdir(runtimeDirectory, { recursive: true, mode: 0o700 });
  const metadata = await lstat(runtimeDirectory);
  if (!metadata.isDirectory() || metadata.isSymbolicLink()) {
    throw new Error("diretorio local de runtime nao e seguro");
  }
  await chmod(runtimeDirectory, 0o700);
}

export async function assertPortAvailable(host, port) {
  await new Promise((resolve, reject) => {
    const server = net.createServer();
    server.unref();
    server.once("error", () => reject(new Error(`porta local ${port} esta ocupada`)));
    server.listen({ host, port, exclusive: true }, () => {
      server.close((error) => (error === undefined ? resolve() : reject(new Error("falha ao liberar teste de porta"))));
    });
  });
}

export class LocalDatabaseSession {
  constructor() {
    this.wasRunning = false;
    this.startCompleted = false;
  }

  async start({ requireInactive = false } = {}) {
    const status = await runCommandCapture("pnpm", ["db:status"], {
      cwd: repositoryRoot,
      env: process.env,
    });
    this.wasRunning = parseDatabaseStatus(status.stdout);
    const containersBeforeStart = await localNetworkContainerIDs();
    if (requireInactive && this.wasRunning) {
      throw new Error("a integracao exige que o Supabase local esteja parado antes do inicio");
    }
    if (!this.wasRunning && containersBeforeStart.length > 0) {
      throw new Error("a rede local contem uma stack parcial preexistente");
    }
    if (!this.wasRunning) {
      try {
        await runCommand("pnpm", ["db:start"], { cwd: repositoryRoot, env: process.env });
        this.startCompleted = true;
      } catch {
        const containersAfterFailure = await localNetworkContainerIDs();
        if (containersBeforeStart.length === 0 && containersAfterFailure.length > 0) {
          this.startCompleted = true;
          await this.stopIfOwned();
        }
        throw new Error("o banco local nao iniciou completamente");
      }
    }
  }

  async resetAndLint() {
    await runCommand("pnpm", ["db:reset"], { cwd: repositoryRoot, env: process.env });
    for (let attempt = 0; attempt < 10; attempt += 1) {
      const lint = await runCommandCapture("pnpm", ["db:lint"], {
        cwd: repositoryRoot,
        env: process.env,
        allowFailure: true,
      });
      if (lint.code === 0) {
        process.stdout.write("Banco local: lint concluido.\n");
        return;
      }
      await new Promise((resolve) => setTimeout(resolve, 500));
    }
    throw new Error("lint do banco local nao estabilizou apos o reset");
  }

  async environment() {
    await runCommand("pnpm", ["db:env"], { cwd: repositoryRoot, env: process.env });
    const port = readDatabasePort(path.join(repositoryRoot, "infra", "supabase", "config.toml"));
    return readLocalEnvironment(path.join(repositoryRoot, ".env.local"), port);
  }

  async stopIfOwned() {
    if (!shouldStopDatabase(this.wasRunning, this.startCompleted)) {
      return;
    }
    await runCommand("pnpm", ["db:stop"], { cwd: repositoryRoot, env: process.env });
    this.startCompleted = false;
  }
}

async function localNetworkContainerIDs() {
  const result = await runCommandCapture(
    "docker",
    [
      "network",
      "inspect",
      "--format",
      "{{range $id, $_ := .Containers}}{{$id}} {{end}}",
      "sysap-loopback",
    ],
    { cwd: repositoryRoot, env: process.env, allowFailure: true },
  );
  if (result.code !== 0) {
    return [];
  }
  const identifiers = result.stdout.trim() === "" ? [] : result.stdout.trim().split(/\s+/);
  if (identifiers.some((identifier) => !/^[0-9a-f]{64}$/.test(identifier))) {
    throw new Error("Docker retornou identificador de container invalido");
  }
  return identifiers;
}

export class OwnedProcesses {
  constructor() {
    this.children = new Set();
    this.completions = new Map();
  }

  start(command, args, options) {
    const child = spawn(command, args, {
      cwd: options.cwd,
      env: options.env,
      shell: false,
      detached: false,
      stdio: "inherit",
    });
    this.children.add(child);
    const completion = new Promise((resolve, reject) => {
      child.once("error", () => reject(new Error(`nao foi possivel iniciar ${options.label}`)));
      child.once("close", (code, signal) => resolve({ child, code: code ?? 1, signal, label: options.label }));
    });
    this.completions.set(child, completion);
    return child;
  }

  completion(child) {
    return this.completions.get(child);
  }

  async terminate(child) {
    if (!canTerminateOwnedChild(child, this.children)) {
      throw new Error("recusa ao encerrar processo nao pertencente ao comando atual");
    }
    if (child.exitCode !== null || child.signalCode !== null) {
      this.children.delete(child);
      return;
    }
    child.kill("SIGTERM");
    const closed = await Promise.race([
      this.completion(child).then(() => true),
      new Promise((resolve) => setTimeout(() => resolve(false), 5_000)),
    ]);
    if (!closed && canTerminateOwnedChild(child, this.children)) {
      child.kill("SIGKILL");
      await this.completion(child);
    }
    this.children.delete(child);
  }

  async terminateAll() {
    const children = [...this.children];
    const results = await Promise.allSettled(children.map((child) => this.terminate(child)));
    if (results.some((result) => result.status === "rejected")) {
      throw new Error("nao foi possivel encerrar todos os processos locais criados");
    }
  }
}

export async function buildAPI(outputName = "sysap-api") {
  await prepareRuntimeDirectory();
  const outputPath = path.join(runtimeDirectory, outputName);
  if (!outputPath.startsWith(`${runtimeDirectory}${path.sep}`)) {
    throw new Error("caminho do binario temporario invalido");
  }
  await rm(outputPath, { force: true });
  const goCache = path.join(runtimeDirectory, "go-cache");
  await mkdir(goCache, { recursive: true, mode: 0o700 });
  await runCommand("go", ["build", "-o", outputPath, "./cmd/api"], {
    cwd: apiDirectory,
    env: safeChildEnvironment({ GOCACHE: goCache, GOFLAGS: "-buildvcs=false" }),
  });
  await chmod(outputPath, 0o700);
  return outputPath;
}

export async function buildWeb() {
  await runCommand("pnpm", ["--filter", "@sysap/web", "build"], {
    cwd: repositoryRoot,
    env: safeChildEnvironment({ NODE_ENV: "production", SYSAP_API_BASE_URL: apiURL }),
  });
}

export function startAPI(processes, binaryPath, databaseURL, environment = "development") {
  return processes.start(binaryPath, [], {
    cwd: apiDirectory,
    label: "API",
    env: safeChildEnvironment({
      SYSAP_ENV: environment,
      SYSAP_HTTP_ADDR: "127.0.0.1:8080",
      SYSAP_DATABASE_URL: databaseURL,
      SYSAP_SHUTDOWN_TIMEOUT: "10s",
      SYSAP_DATABASE_PING_TIMEOUT: "2s",
    }),
  });
}

export function startWeb(processes, mode = "dev") {
  const args = [nextBinary, mode, "--hostname", "127.0.0.1", "--port", "3000"];
  return processes.start(process.execPath, args, {
    cwd: webDirectory,
    label: "Web",
    env: safeChildEnvironment({
      NODE_ENV: mode === "start" ? "production" : "development",
      SYSAP_API_BASE_URL: apiURL,
    }),
  });
}

export async function waitForAPI(pathname, expectedStatus, timeoutMilliseconds = 30_000) {
  const endpoint = new URL(pathname, apiURL);
  return waitFor(async () => {
    const response = await fetch(endpoint, {
      cache: "no-store",
      redirect: "error",
      signal: AbortSignal.timeout(1_000),
    });
    if (response.status !== expectedStatus || response.redirected) {
      return null;
    }
    const text = await readLimitedText(response, 64 * 1024);
    return { response, text };
  }, timeoutMilliseconds);
}

export async function waitForWeb(expectedText, timeoutMilliseconds = 45_000) {
  return waitFor(async () => {
    const response = await fetch(webURL, {
      cache: "no-store",
      redirect: "error",
      signal: AbortSignal.timeout(2_500),
    });
    if (response.status !== 200 || response.redirected) {
      return null;
    }
    const text = await readLimitedText(response, 2 * 1024 * 1024);
    return text.includes(expectedText) ? { response, text } : null;
  }, timeoutMilliseconds);
}

async function waitFor(attempt, timeoutMilliseconds) {
  const deadline = Date.now() + timeoutMilliseconds;
  while (Date.now() < deadline) {
    try {
      const result = await attempt();
      if (result !== null) {
        return result;
      }
    } catch {
      // A inicializacao e as transicoes de degradacao sao eventualmente consistentes.
    }
    await new Promise((resolve) => setTimeout(resolve, 250));
  }
  throw new Error("servico local nao atingiu o estado esperado no prazo");
}

async function readLimitedText(response, maximumBytes) {
  if (response.body === null) {
    throw new Error("resposta local sem corpo");
  }
  const reader = response.body.getReader();
  const chunks = [];
  let received = 0;
  try {
    while (true) {
      const { done, value } = await reader.read();
      if (done) {
        break;
      }
      received += value.byteLength;
      if (received > maximumBytes) {
        await reader.cancel();
        throw new Error("resposta local excedeu o limite seguro");
      }
      chunks.push(value);
    }
  } finally {
    reader.releaseLock();
  }
  const body = new Uint8Array(received);
  let offset = 0;
  for (const chunk of chunks) {
    body.set(chunk, offset);
    offset += chunk.byteLength;
  }
  return new TextDecoder().decode(body);
}
