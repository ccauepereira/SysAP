import assert from "node:assert/strict";
import test from "node:test";
import { parseSmokeTestEnv } from "./email-smoke-test.mjs";

test("parseSmokeTestEnv extrai valores com aspas duplas, aspas simples, ou sem aspas", () => {
  const envContent = `
# comentario
SYSAP_EMAIL_PROVIDER=brevo_api
SYSAP_BREVO_API_KEY=key_123
SYSAP_EMAIL_FROM="from@example.com"
export SYSAP_EMAIL_FROM_NAME='Sender Name'
SYSAP_EMAIL_SMOKE_TEST_TO=to@example.com
  `;
  const env = parseSmokeTestEnv(envContent);
  assert.equal(env.SYSAP_EMAIL_PROVIDER, "brevo_api");
  assert.equal(env.SYSAP_BREVO_API_KEY, "key_123");
  assert.equal(env.SYSAP_EMAIL_FROM, "from@example.com");
  assert.equal(env.SYSAP_EMAIL_FROM_NAME, "Sender Name");
  assert.equal(env.SYSAP_EMAIL_SMOKE_TEST_TO, "to@example.com");
});

test("parseSmokeTestEnv lida corretamente com espacos extras", () => {
  const envContent = `
  SYSAP_EMAIL_PROVIDER =   brevo_api  
  `;
  const env = parseSmokeTestEnv(envContent);
  assert.equal(env.SYSAP_EMAIL_PROVIDER, "brevo_api");
});
