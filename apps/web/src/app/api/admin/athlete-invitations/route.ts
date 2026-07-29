import { NextRequest, NextResponse } from "next/server";
import { getAccessToken } from "@/lib/auth/cookies";
import { getSession } from "@/lib/auth/session";
import { validateOrigin } from "@/lib/auth/csrf";

export async function POST(request: NextRequest) {
  if (!validateOrigin(request)) return NextResponse.json({ error: "forbidden" }, { status: 403 });
  const session = await getSession();
  if (session.status !== "authenticated" || session.user.role !== "owner") return NextResponse.json({ error: "access_denied" }, { status: 403 });
  const token = await getAccessToken();
  if (!token) return NextResponse.json({ error: "authentication_required" }, { status: 401 });
  try {
    const body = await request.json() as Record<string, unknown>;
    const required = ["full_name", "phone_e164", "email", "birth_date", "locality", "team_id", "football_position", "enrollment_delivery_channel"];
    if (required.some((key) => typeof body[key] !== "string" || String(body[key]).trim() === "")) return NextResponse.json({ error: "validation_failed" }, { status: 422 });
    if (!/^(sms|email)$/.test(String(body.enrollment_delivery_channel))) return NextResponse.json({ error: "validation_failed" }, { status: 422 });
    const apiUrl = process.env.SYSAP_API_BASE_URL || "http://127.0.0.1:8080";
    const response = await fetch(`${apiUrl}/v1/organizations/${session.user.organizationId}/athlete-invitations`, { method: "POST", headers: { Authorization: `Bearer ${token}`, "Content-Type": "application/json", "X-Organization-ID": session.user.organizationId, "Idempotency-Key": crypto.randomUUID() }, body: JSON.stringify(body), cache: "no-store" });
    if (!response.ok) return NextResponse.json({ error: response.status === 503 ? "service_unavailable" : response.status === 403 ? "access_denied" : "invitation_failed" }, { status: response.status });
    const result = await response.json() as { invitation_id?: string; profile_id?: string; status?: string; expires_at?: string; delivery_channel?: string };
    return NextResponse.json({ invitation_id: result.invitation_id, profile_id: result.profile_id, status: result.status, expires_at: result.expires_at, delivery_channel: result.delivery_channel }, { status: 201, headers: { "Cache-Control": "no-store" } });
  } catch {
    return NextResponse.json({ error: "service_unavailable" }, { status: 503 });
  }
}
