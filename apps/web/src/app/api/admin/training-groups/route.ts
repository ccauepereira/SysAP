import { NextRequest, NextResponse } from "next/server";
import { getAccessToken } from "@/lib/auth/cookies";
import { validateOrigin } from "@/lib/auth/csrf";
import { getSession } from "@/lib/auth/session";

export async function GET() {
  const session = await getSession();
  if (session.status !== "authenticated" || session.user.role !== "owner") {
    return NextResponse.json({ error: "access_denied" }, { status: 403 });
  }
  const token = await getAccessToken();
  if (!token) return NextResponse.json({ error: "authentication_required" }, { status: 401 });
  try {
    const apiUrl = process.env.SYSAP_API_BASE_URL || "http://127.0.0.1:8080";
    const response = await fetch(`${apiUrl}/v1/organizations/${encodeURIComponent(session.user.organizationId)}/training-groups`, {
      method: "GET",
      headers: {
        Authorization: `Bearer ${token}`,
        "X-Organization-ID": session.user.organizationId,
      },
      cache: "no-store",
    });
    if (!response.ok) return NextResponse.json({ error: "failed" }, { status: response.status });
    const data = await response.json();
    return NextResponse.json(data);
  } catch {
    return NextResponse.json({ error: "service_unavailable" }, { status: 503 });
  }
}

export async function POST(request: NextRequest) {
  if (!validateOrigin(request)) return NextResponse.json({ error: "forbidden" }, { status: 403 });
  const session = await getSession();
  if (session.status !== "authenticated" || session.user.role !== "owner") return NextResponse.json({ error: "access_denied" }, { status: 403 });
  const token = await getAccessToken();
  if (!token) return NextResponse.json({ error: "authentication_required" }, { status: 401 });
  let body: unknown;
  try { body = await request.json(); } catch { return NextResponse.json({ error: "invalid_request" }, { status: 400 }); }
  if (typeof body !== "object" || body === null || !("name" in body) || typeof body.name !== "string") {
    return NextResponse.json({ error: "invalid_request" }, { status: 400 });
  }
  try {
    const apiUrl = process.env.SYSAP_API_BASE_URL || "http://127.0.0.1:8080";
    const response = await fetch(`${apiUrl}/v1/organizations/${session.user.organizationId}/training-groups`, {
      method: "POST", headers: { Authorization: `Bearer ${token}`, "Content-Type": "application/json", "X-Organization-ID": session.user.organizationId },
      body: JSON.stringify({ name: body.name }), cache: "no-store",
    });
    if (!response.ok) return NextResponse.json({ error: "group_creation_failed" }, { status: response.status });
    if (response.status !== 201) return NextResponse.json({ error: "invalid_response" }, { status: 502 });
    const result: unknown = await response.json();
    if (typeof result !== "object" || result === null || !("id" in result) || !("name" in result) || typeof result.id !== "string" || !/^[0-9a-f]{8}-[0-9a-f]{4}-[1-8][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$/i.test(result.id) || typeof result.name !== "string") {
      return NextResponse.json({ error: "invalid_response" }, { status: 502 });
    }
    return NextResponse.json({ id: result.id, name: result.name }, { status: 201, headers: { "Cache-Control": "no-store" } });
  } catch {
    return NextResponse.json({ error: "service_unavailable" }, { status: 503 });
  }
}
