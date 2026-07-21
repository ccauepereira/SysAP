import assert from "node:assert/strict";
import test from "node:test";

import {
  validatePackagePolicy,
  validateRepositoryPaths,
  validateWorkflowText,
} from "./repository-policy.mjs";

function validPackages() {
  const scripts = Object.fromEntries(
    [
      "dev", "check", "check:api", "check:web", "openapi:lint", "test", "test:api",
      "test:web", "test:integration", "security:dependencies", "security:secrets",
      "db:start", "db:stop", "db:status", "db:reset", "db:lint", "db:env",
    ].map((name) => [name, "fixture"]),
  );
  return {
    root: {
      packageManager: "pnpm@11.15.1",
      scripts,
      devDependencies: { "@redocly/cli": "2.39.0", supabase: "2.109.1" },
    },
    web: { engines: { node: "24.18.0" }, dependencies: { next: "16.2.10" } },
  };
}

test("accepts exact package and runtime pins", () => {
  const packages = validPackages();
  assert.deepEqual(
    validatePackagePolicy(packages.root, packages.web, "nodeVersion: 24.18.0", "module fixture\n\ngo 1.26.5\n"),
    [],
  );
});

test("rejects ranges, latest, exotic sources and runtime drift", () => {
  const packages = validPackages();
  packages.root.packageManager = "pnpm@11.0.0";
  packages.web.dependencies.next = "^16.2.10";
  packages.web.devDependencies = { fixture: "git:https://example.invalid/fixture" };
  const errors = validatePackagePolicy(packages.root, packages.web, "nodeVersion: 24.0.0", "go 1.25.0\n");
  assert.equal(errors.length >= 4, true);
});

test("accepts only approved Actions with full SHAs and tag comments", () => {
  const workflow = `
on:
  push:
  pull_request:
  workflow_dispatch:
permissions:
  contents: read
jobs:
  fixture:
    runs-on: ubuntu-24.04
    steps:
      - uses: actions/checkout@11bd71901bbe5b1630ceea73d27597364c9af683 # v4.2.2
        with:
          fetch-depth: 0
          persist-credentials: false
`;
  assert.deepEqual(validateWorkflowText(workflow), []);
});

test("rejects mutable Actions, dangerous triggers, writes and secret contexts", () => {
  const workflow = `
pull_request_target:
permissions: write-all
runs-on: self-hosted
- uses: actions/checkout@v4
- run: echo \${{ secrets.FIXTURE }}
`;
  assert.equal(validateWorkflowText(workflow).length >= 5, true);
});

test("rejects generated, environment and key material paths", () => {
  const errors = validateRepositoryPaths([
    "apps/web/.next/build.json",
    ".env.local",
    "coverage/report.json",
    "fixture.key",
  ]);
  assert.equal(errors.length, 4);
  assert.deepEqual(validateRepositoryPaths([".env.example", "apps/web/.env.example"]), []);
});
