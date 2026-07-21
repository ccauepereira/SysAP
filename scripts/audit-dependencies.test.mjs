import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import path from "node:path";
import test from "node:test";

import { evaluateAuditReport } from "./audit-dependencies.mjs";

async function fixture(name) {
  const contents = await readFile(path.join(import.meta.dirname, "fixtures", name), "utf8");
  return JSON.parse(contents);
}

test("accepts an audit without advisories and reports the obsolete exception", async () => {
  assert.deepEqual(evaluateAuditReport(await fixture("pnpm-audit-clean.json")), {
    acceptedPostCSSRisk: false,
    advisoryCount: 0,
  });
});

test("accepts only the exact documented PostCSS path and version", async () => {
  assert.deepEqual(evaluateAuditReport(await fixture("pnpm-audit-accepted.json")), {
    acceptedPostCSSRisk: true,
    advisoryCount: 1,
  });
});

test("rejects a new high vulnerability", async () => {
  const report = await fixture("pnpm-audit-rejected.json");
  assert.throws(() => evaluateAuditReport(report));
});

test("rejects the known advisory on a different path, version, or package", async () => {
  const report = await fixture("pnpm-audit-accepted.json");
  const advisory = report.advisories["1000001"];

  advisory.findings[0].paths = ["other>postcss"];
  assert.throws(() => evaluateAuditReport(report));
  advisory.findings[0].paths = ["apps__web>next>postcss"];
  advisory.findings[0].version = "8.4.30";
  assert.throws(() => evaluateAuditReport(report));
  advisory.findings[0].version = "8.4.31";
  advisory.module_name = "other-package";
  assert.throws(() => evaluateAuditReport(report));
});

test("rejects any additional advisory, including lower severity", async () => {
  const report = await fixture("pnpm-audit-accepted.json");
  report.advisories["1000003"] = {
    github_advisory_id: "GHSA-1111-1111-1111",
    module_name: "fixture-package",
    severity: "low",
    findings: [{ version: "1.0.0", paths: ["fixture-package"] }],
  };
  assert.throws(() => evaluateAuditReport(report));
});
