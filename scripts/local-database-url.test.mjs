import assert from "node:assert/strict";
import test from "node:test";

import {
  renderEnvironment,
  validateLocalDatabaseURL,
} from "./local-database-url.mjs";

const databasePort = 54322;

function fixtureURL(overrides = {}) {
  const value = new URL(`${overrides.protocol ?? "postgresql:"}//localhost`);
  value.username = overrides.username ?? "fixture-user";
  value.password = overrides.password ?? "fixture-password";
  value.hostname = overrides.hostname ?? "127.0.0.1";
  value.port = overrides.port ?? String(databasePort);
  value.pathname = overrides.pathname ?? "/postgres";
  return value;
}

test("accepts a complete local PostgreSQL URL", () => {
  const input = fixtureURL().href;
  assert.equal(validateLocalDatabaseURL(input, databasePort), input);
  assert.equal(
    renderEnvironment(input),
    `SYSAP_DATABASE_URL='${input}'\nSYSAP_TEST_DATABASE_URL='${input}'\n`,
  );
});

test("rejects an invalid protocol", () => {
  assert.throws(() => validateLocalDatabaseURL(fixtureURL({ protocol: "http:" }).href, databasePort));
});

test("rejects a remote hostname", () => {
  assert.throws(() => validateLocalDatabaseURL(fixtureURL({ hostname: "database.example.invalid" }).href, databasePort));
});

test("rejects an incorrect port", () => {
  assert.throws(() => validateLocalDatabaseURL(fixtureURL({ port: "54323" }).href, databasePort));
});

test("rejects a missing username", () => {
  assert.throws(() => validateLocalDatabaseURL(fixtureURL({ username: "" }).href, databasePort));
});

test("rejects a missing password", () => {
  assert.throws(() => validateLocalDatabaseURL(fixtureURL({ password: "" }).href, databasePort));
});

test("rejects a newline", () => {
  assert.throws(() => validateLocalDatabaseURL(`${fixtureURL().href}\n`, databasePort));
});

test("rejects a carriage return", () => {
  assert.throws(() => validateLocalDatabaseURL(`${fixtureURL().href}\r`, databasePort));
});

test("rejects other control characters", () => {
  assert.throws(() => validateLocalDatabaseURL(`${fixtureURL().href}\u0001`, databasePort));
});

test("rejects command substitution", () => {
  assert.throws(() => validateLocalDatabaseURL(`${fixtureURL().href}$(fixture-command)`, databasePort));
});

test("rejects a URL fragment", () => {
  const input = fixtureURL();
  input.hash = "fragment";
  assert.throws(() => validateLocalDatabaseURL(input.href, databasePort));
});

test("rejects an unexpected database name", () => {
  assert.throws(() => validateLocalDatabaseURL(fixtureURL({ pathname: "/other" }).href, databasePort));
});

test("rejects an attempt to add a third variable", () => {
  assert.throws(() => validateLocalDatabaseURL(`${fixtureURL().href}\nSYSAP_EXTRA=value`, databasePort));
});
