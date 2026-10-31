import assert from "node:assert/strict";
import test from "node:test";

import {
  initializeIntegrationDatabase,
  integrationAPIEnvironment,
} from "./integration-setup.mjs";
import { repositoryRoot } from "./local-runtime.mjs";

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
