import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import path from "node:path";
import test from "node:test";

import { evaluateAuditReport } from "./audit-dependencies.mjs";

async function fixture(name) {
  const contents = await readFile(path.join(import.meta.dirname, "fixtures", name), "utf8");
  return JSON.parse(contents);
}

test("accepts an audit without advisories (clean report)", async () => {
  assert.deepEqual(evaluateAuditReport(await fixture("pnpm-audit-clean.json")), {
    advisoryCount: 0,
  });
});

test("rejects any advisory (unexpected advisory)", async () => {
  const report = await fixture("pnpm-audit-accepted.json");
  assert.throws(() => evaluateAuditReport(report), /vulnerabilidade nao autorizada encontrada/);
});

test("rejects a high severity advisory", async () => {
  const report = await fixture("pnpm-audit-rejected.json");
  assert.throws(() => evaluateAuditReport(report), /vulnerabilidade nao autorizada encontrada/);
});

test("rejects invalid report structure (null, non-object, array)", () => {
  assert.throws(() => evaluateAuditReport(null), /relatorio de dependencias invalido/);
  assert.throws(() => evaluateAuditReport("not-json"), /relatorio de dependencias invalido/);
  assert.throws(() => evaluateAuditReport([]), /relatorio de dependencias invalido/);
});

test("rejects report missing advisories field or unexpected format", () => {
  assert.throws(() => evaluateAuditReport({}), /relatorio de dependencias sem advisories/);
  assert.throws(() => evaluateAuditReport({ advisories: null }), /relatorio de dependencias sem advisories/);
  assert.throws(() => evaluateAuditReport({ advisories: "invalid" }), /relatorio de dependencias sem advisories/);
});
