import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import path from "node:path";
import test from "node:test";

import { evaluateAuditReport, temporaryException } from "./audit-dependencies.mjs";

const beforeExpiry = new Date("2026-10-14T03:00:00Z");
const atExpiry = new Date(temporaryException.expiresAt);

async function fixture(name) {
  const contents = await readFile(path.join(import.meta.dirname, "fixtures", name), "utf8");
  return JSON.parse(contents);
}

async function bracesFinding(mutate) {
  const report = await fixture("pnpm-audit-braces-exception.json");
  mutate(report.advisories["1240992"]);
  return report;
}

test("accepts an audit without advisories (clean report)", async () => {
  assert.deepEqual(evaluateAuditReport(await fixture("pnpm-audit-clean.json"), beforeExpiry), {
    advisoryCount: 0,
    acceptedException: null,
  });
});

test("rejects any advisory (unexpected advisory)", async () => {
  const report = await fixture("pnpm-audit-accepted.json");
  assert.throws(() => evaluateAuditReport(report, beforeExpiry), /vulnerabilidade nao autorizada encontrada/);
});

test("rejects a high severity advisory", async () => {
  const report = await fixture("pnpm-audit-rejected.json");
  assert.throws(() => evaluateAuditReport(report, beforeExpiry), /vulnerabilidade nao autorizada encontrada/);
});

test("accepts only the exact temporary braces exception before expiry", async () => {
  const decision = evaluateAuditReport(await fixture("pnpm-audit-braces-exception.json"), beforeExpiry);
  assert.equal(decision.advisoryCount, 1);
  assert.equal(decision.acceptedException, temporaryException);
  assert.ok(decision.acceptedException.owner.length > 0);
  assert.ok(decision.acceptedException.justification.length > 0);
});

test("rejects the braces exception at or after its expiry", async () => {
  const report = await fixture("pnpm-audit-braces-exception.json");
  assert.throws(() => evaluateAuditReport(report, atExpiry), /excecao temporaria GHSA-vfj7-8cjw-p6xm expirada/);
  const later = new Date(atExpiry.getTime() + 24 * 60 * 60 * 1000);
  assert.throws(() => evaluateAuditReport(report, later), /excecao temporaria GHSA-vfj7-8cjw-p6xm expirada/);
});

test("rejects braces findings outside the exact exception", async () => {
  const variants = {
    "production path": (advisory) => {
      advisory.findings[0].dev = false;
    },
    "missing dev flag": (advisory) => {
      delete advisory.findings[0].dev;
    },
    "other path": (advisory) => {
      advisory.findings[0].paths = ["apps__web>next>micromatch>braces"];
    },
    "additional path": (advisory) => {
      advisory.findings[0].paths.push("apps__web>other>micromatch>braces");
    },
    "other version": (advisory) => {
      advisory.findings[0].version = "3.0.2";
    },
    "additional finding": (advisory) => {
      advisory.findings.push({ ...advisory.findings[0], version: "2.3.2" });
    },
    "escalated severity": (advisory) => {
      advisory.severity = "critical";
    },
    "other advisory id": (advisory) => {
      advisory.github_advisory_id = "GHSA-0000-0000-0000";
    },
  };
  for (const [name, mutate] of Object.entries(variants)) {
    const report = await bracesFinding(mutate);
    assert.throws(() => evaluateAuditReport(report, beforeExpiry), /vulnerabilidade nao autorizada encontrada/, name);
  }
});

test("rejects the braces exception combined with any other advisory", async () => {
  const report = await fixture("pnpm-audit-braces-exception.json");
  const other = await fixture("pnpm-audit-rejected.json");
  Object.assign(report.advisories, other.advisories);
  assert.throws(() => evaluateAuditReport(report, beforeExpiry), /vulnerabilidade nao autorizada encontrada/);
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
