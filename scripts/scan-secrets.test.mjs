import assert from "node:assert/strict";
import test from "node:test";

import { parseGitleaksReport, safeFindingDescription } from "./scan-secrets.mjs";

test("accepts an empty redacted Gitleaks report", () => {
  assert.deepEqual(parseGitleaksReport("[]\n"), []);
});

test("a new synthetic finding remains blocking without carrying secret content", () => {
  const findings = parseGitleaksReport(
    JSON.stringify([{ RuleID: "generic-api-key", File: "scripts/fixtures/new-fixture.txt" }]),
  );
  assert.equal(findings.length, 1);
  assert.equal(
    safeFindingDescription(findings[0]),
    "generic-api-key: scripts/fixtures/new-fixture.txt",
  );
});

test("sanitizes untrusted rule and path metadata", () => {
  assert.equal(
    safeFindingDescription({ rule: "rule\nvalue", file: "../../fixture\nfile" }),
    "rule?value: fixture?file",
  );
});

test("rejects malformed reports", () => {
  assert.throws(() => parseGitleaksReport("{}"));
  assert.throws(() => parseGitleaksReport("not-json"));
  assert.throws(() => parseGitleaksReport(JSON.stringify([{ RuleID: "fixture" }])));
});
