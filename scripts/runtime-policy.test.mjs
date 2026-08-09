import assert from "node:assert/strict";
import test from "node:test";

import {
  canTerminateOwnedChild,
  hasForbiddenEnvironmentName,
  parseDatabaseStatus,
  parseLocalEnvironment,
  parseSupabaseEnv,
  sanitizeMessage,
  shouldStopDatabase,
  validateLoopbackHTTPURL,
} from "./runtime-policy.mjs";

const databasePort = 54322;

function fixtureDatabaseURL() {
  const value = new URL("postgresql://127.0.0.1");
  value.username = "fixture-user";
  value.password = "fixture-password";
  value.port = String(databasePort);
  value.pathname = "/postgres";
  return value.href;
}

test("parses SYSAP_ variables to an allowlisted environment", () => {
  const value = fixtureDatabaseURL();
  const input = `
SYSAP_ENV=development
# Comment
SYSAP_DATABASE_URL='${value}'
SYSAP_TEST_DATABASE_URL='${value}'
SYSAP_AUTH_JWT_ISSUER="test-issuer"
`;
  assert.deepEqual(
    parseLocalEnvironment(input, databasePort),
    {
      SYSAP_ENV: "development",
      SYSAP_DATABASE_URL: value,
      SYSAP_TEST_DATABASE_URL: value,
      SYSAP_AUTH_JWT_ISSUER: "test-issuer",
    },
  );
});

test("rejects extra, duplicated, inconsistent, and remote environment values", () => {
  const value = fixtureDatabaseURL();
  assert.throws(() => parseLocalEnvironment(`SYSAP_DATABASE_URL='${value}'\n`, databasePort));
  const other = new URL(value);
  other.hostname = "database.example.invalid";
  assert.throws(() =>
    parseLocalEnvironment(
      `SYSAP_DATABASE_URL='${other.href}'\nSYSAP_TEST_DATABASE_URL='${other.href}'\n`,
      databasePort,
    ),
  );
  assert.throws(() =>
    parseLocalEnvironment(
      `SYSAP_DATABASE_URL='${value}'\nSYSAP_TEST_DATABASE_URL='${value}'\nINVALID_VAR=123\n`,
      databasePort,
    ),
  );
});

test("accepts only the expected loopback HTTP endpoint", () => {
  assert.equal(validateLoopbackHTTPURL("http://127.0.0.1:8080", 8080), "http://127.0.0.1:8080");
  assert.throws(() => validateLoopbackHTTPURL("http://0.0.0.0:8080", 8080));
  assert.throws(() => validateLoopbackHTTPURL("https://127.0.0.1:8080", 8080));
  assert.throws(() => validateLoopbackHTTPURL("http://127.0.0.1:8081", 8080));
});

test("recognizes only fixed database status messages", () => {
  assert.equal(parseDatabaseStatus("Supabase local: running.\n"), true);
  assert.equal(parseDatabaseStatus("Supabase local: not running.\n"), false);
  assert.throws(() => parseDatabaseStatus("running with details"));
});

test("stops only a database started by the current command", () => {
  assert.equal(shouldStopDatabase(false, true), true);
  assert.equal(shouldStopDatabase(true, true), false);
  assert.equal(shouldStopDatabase(false, false), false);
});

test("signals only the exact owned child object with a validated PID", () => {
  const child = { pid: 1234 };
  const other = { pid: 1234 };
  const owned = new Set([child]);
  assert.equal(canTerminateOwnedChild(child, owned), true);
  assert.equal(canTerminateOwnedChild(other, owned), false);
  assert.equal(canTerminateOwnedChild({ pid: 1 }, new Set()), false);
});

test("detects environment names that must never reach runtime children", () => {
  assert.equal(hasForbiddenEnvironmentName("NEXT_PUBLIC_DATABASE_URL"), true);
  assert.equal(hasForbiddenEnvironmentName("SUPABASE_SERVICE_ROLE_KEY"), true);
  assert.equal(hasForbiddenEnvironmentName("SYSAP_DATABASE_URL"), false);
});

test("sanitizes URLs with userinfo, token shapes, assignments and explicit values", () => {
  const explicit = "fixture-sensitive-value";
  const input = [
    "postgresql://fixture-user:fixture-password@127.0.0.1:54322/postgres",
    "token=fixture-token",
    "eyJabcdefgh.ijklmnop.qrstuvwx",
    explicit,
  ].join(" ");
  const result = sanitizeMessage(input, [explicit]);
  assert.equal(result.includes("fixture-password"), false);
  assert.equal(result.includes("fixture-token"), false);
  assert.equal(result.includes("eyJabcdefgh"), false);
  assert.equal(result.includes(explicit), false);
});

test("parseSupabaseEnv extracts API_URL and SERVICE_ROLE_KEY ignoring ANSI codes", () => {
  const stdout = `\x1B[32mAPI_URL="http://127.0.0.1:54321"\x1B[0m\n\x1B[32mSERVICE_ROLE_KEY="test-key-123"\x1B[0m\nANOTHER_VAR="foo"\n`;
  const env = parseSupabaseEnv(stdout);
  assert.equal(env.apiURL, "http://127.0.0.1:54321");
  assert.equal(env.serviceRoleKey, "test-key-123");
});

test("parseSupabaseEnv throws if credentials are missing", () => {
  const stdout = `API_URL="http://127.0.0.1:54321"\n`;
  assert.throws(() => parseSupabaseEnv(stdout));
});
