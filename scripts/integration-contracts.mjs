export function assertNoSensitiveRuntimeDetails(text, databaseURL) {
  const parsed = new URL(databaseURL);
  const forbiddenValues = [databaseURL, parsed.host, parsed.username, parsed.password].filter(Boolean);
  const containsCredentialAssignment =
    /\b(?:password|passwd|token|secret|service[_-]?role[_-]?key|api[_-]?key)\s*[:=]\s*["']?[^\s"'<>{}]{4,}/i.test(text);
  const containsJWT = /\beyJ[A-Za-z0-9_-]*\.[A-Za-z0-9_-]+\.[A-Za-z0-9_-]+\b/.test(text);

  if (
    forbiddenValues.some((value) => text.includes(value)) ||
    containsCredentialAssignment ||
    containsJWT
  ) {
    throw new Error("resposta local expos detalhes sensiveis");
  }
}
