import assert from "node:assert/strict";
import { createServer } from "node:http";
import test from "node:test";

import {
  initializeIntegrationDatabase,
  integrationAPIEnvironment,
} from "./integration-setup.mjs";
import { assertNoSensitiveRuntimeDetails } from "./integration-contracts.mjs";
import { OwnedProcesses, repositoryRoot, waitForLocalHTTP } from "./local-runtime.mjs";

async function withHTTPServer(handler, callback) {
  const server = createServer(handler);
  await new Promise((resolve, reject) => {
    server.once("error", reject);
    server.listen(0, "127.0.0.1", resolve);
  });
  const address = server.address();
  assert.ok(address && typeof address === "object");

  try {
    await callback(new URL(`http://127.0.0.1:${address.port}`));
  } finally {
    await new Promise((resolve, reject) => {
      server.close((error) => (error ? reject(error) : resolve()));
      server.closeAllConnections();
    });
  }
}

test("generates the local database environment after reset and before reading it", async () => {
  const calls = [];
  const session = {
    async start(options) {
      calls.push(["start", options]);
    },
    async resetAndLint() {
      calls.push(["resetAndLint"]);
    },
    async environment() {
      calls.push(["environment"]);
      return { SYSAP_DATABASE_URL: "local", SYSAP_TEST_DATABASE_URL: "local-test" };
    },
  };
  const executeCommand = async (...args) => {
    calls.push(["command", ...args]);
  };

  const environment = await initializeIntegrationDatabase(session, executeCommand);

  assert.deepEqual(calls.map(([name]) => name), ["start", "resetAndLint", "command", "environment"]);
  assert.deepEqual(calls[0], ["start", { requireInactive: true }]);
  assert.deepEqual(calls[2], [
    "command",
    "pnpm",
    ["db:env"],
    { cwd: repositoryRoot, env: process.env },
  ]);
  assert.deepEqual(environment, { SYSAP_DATABASE_URL: "local", SYSAP_TEST_DATABASE_URL: "local-test" });
});

test("does not read the database environment when local environment generation fails", async () => {
  const calls = [];
  const session = {
    async start() {
      calls.push("start");
    },
    async resetAndLint() {
      calls.push("resetAndLint");
    },
    async environment() {
      calls.push("environment");
      throw new Error("environment must not be read");
    },
  };
  const executeCommand = async () => {
    calls.push("db:env");
    throw new Error("local environment generation failed");
  };

  await assert.rejects(
    initializeIntegrationDatabase(session, executeCommand),
    /local environment generation failed/,
  );
  assert.deepEqual(calls, ["start", "resetAndLint", "db:env"]);
});

test("passes the validated API database URL as an environment object", () => {
  const source = {
    SYSAP_DATABASE_URL: "local-database-url",
    SYSAP_TEST_DATABASE_URL: "local-test-database-url",
  };

  assert.deepEqual(integrationAPIEnvironment(source), {
    SYSAP_DATABASE_URL: "local-database-url",
  });
});

test("waits for a local HTTP path with the expected status and returns its bounded body", async () => {
  await withHTTPServer((request, response) => {
    assert.equal(request.url, "/login");
    response.writeHead(200, { "content-type": "text/html; charset=utf-8" });
    response.end("SysAP login");
  }, async (baseURL) => {
    const result = await waitForLocalHTTP(
      new URL("/login", baseURL),
      200,
      1_000,
      1_024,
      500,
    );

    assert.equal(result.response.status, 200);
    assert.equal(result.text, "SysAP login");
  });
});

test("does not accept an unexpected HTTP status as availability", async () => {
  await withHTTPServer((_request, response) => {
    response.writeHead(503, { "content-type": "application/json" });
    response.end('{"error":"unavailable"}');
  }, async (baseURL) => {
    await assert.rejects(
      waitForLocalHTTP(new URL("/login", baseURL), 200, 30, 1_024, 500),
      /servico local nao atingiu o estado esperado no prazo/,
    );
  });
});

test("times out when a local HTTP endpoint never responds", async () => {
  await withHTTPServer(() => {}, async (baseURL) => {
    await assert.rejects(
      waitForLocalHTTP(new URL("/readyz", baseURL), 200, 80, 1_024, 20),
      /servico local nao atingiu o estado esperado no prazo/,
    );
  });
});

test("rejects a response body larger than the configured read limit", async () => {
  await withHTTPServer((_request, response) => {
    response.writeHead(200, { "content-type": "text/plain" });
    response.end("x".repeat(1_024));
  }, async (baseURL) => {
    await assert.rejects(
      waitForLocalHTTP(new URL("/login", baseURL), 200, 1_000, 32, 500),
      /resposta local excedeu o limite seguro/,
    );
  });
});

test("rejects database credentials, JWTs, and credential assignments without echoing them", () => {
  const databaseURL = "postgresql://fixture-user:fixture-password@127.0.0.1:54322/postgres";
  const leakedValues = [
    databaseURL,
    "token=fixture-token-value",
    "eyJhbGciOiJub25lIn0.eyJzdWIiOiJmaXh0dXJlIn0.signature",
  ];

  assertNoSensitiveRuntimeDetails("login page without runtime details", databaseURL);
  for (const body of leakedValues) {
    assert.throws(
      () => assertNoSensitiveRuntimeDetails(body, databaseURL),
      (error) => error.message === "resposta local expos detalhes sensiveis" &&
        leakedValues.every((value) => !error.message.includes(value)),
    );
  }
});

test("terminates only processes owned by the integration runtime", async () => {
  const processes = new OwnedProcesses();
  const child = processes.start(process.execPath, ["-e", "setInterval(() => {}, 1000)"], {
    cwd: repositoryRoot,
    label: "teste descartavel",
    env: process.env,
  });

  await processes.terminateAll();

  assert.ok(child.exitCode !== null || child.signalCode !== null);
  assert.equal(processes.children.size, 0);
});
