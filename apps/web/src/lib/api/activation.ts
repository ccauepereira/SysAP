import "server-only";

export async function callActivation(path: string, body: unknown): Promise<Response> {
  const apiUrl = process.env.SYSAP_API_BASE_URL || "http://127.0.0.1:8080";
  return fetch(`${apiUrl}${path}`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(body),
    cache: "no-store",
  });
}

export function safeActivationError(status: number): Response {
  if (status === 503) return Response.json({ error: "service_unavailable" }, { status: 503 });
  if (status === 422) return Response.json({ error: "validation_failed" }, { status: 422 });
  return Response.json({ error: "activation_failed" }, { status: status >= 400 && status < 500 ? status : 401 });
}
