import { NextRequest, NextResponse } from "next/server";
import { getAccessToken } from "@/lib/auth/cookies";
import { getSession } from "@/lib/auth/session";
import { validateOrigin } from "@/lib/auth/csrf";

const uuidPattern = /^[0-9a-f]{8}-[0-9a-f]{4}-[1-8][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$/i;

export async function POST(request: NextRequest, context: { params: Promise<{ invitationId: string }> }) {
  if (!validateOrigin(request)) return NextResponse.json({ error: "forbidden" }, { status: 403 });
  const session = await getSession();
  if (session.status !== "authenticated" || session.user.role !== "owner") return NextResponse.json({ error: "access_denied" }, { status: 403 });
  const token = await getAccessToken();
  if (!token) return NextResponse.json({ error: "authentication_required" }, { status: 401 });
  const { invitationId } = await context.params;
  const idempotencyKey = request.headers.get("Idempotency-Key");
  if (!uuidPattern.test(invitationId) || !idempotencyKey || !uuidPattern.test(idempotencyKey)) return NextResponse.json({ error: "validation_failed" }, { status: 400 });
  try {
    const apiUrl = process.env.SYSAP_API_BASE_URL || "http://127.0.0.1:8080";
    const response = await fetch(`${apiUrl}/v1/athlete-invitations/${invitationId}/resend`, {
      method: "POST",
      headers: { Authorization: `Bearer ${token}`, "X-Organization-ID": session.user.organizationId, "Idempotency-Key": idempotencyKey },
      cache: "no-store",
    });
    if (!response.ok) return NextResponse.json({ error: response.status === 429 ? "delivery_rate_limited" : response.status === 409 ? "delivery_retry_not_allowed" : response.status === 403 ? "access_denied" : "delivery_unavailable" }, { status: response.status });
    const item = await response.json() as Record<string, unknown>;
    return NextResponse.json({ invitation_id: item.invitation_id, profile_id: item.profile_id, status: item.status, expires_at: item.expires_at, delivery_channel: item.delivery_channel, delivery_status: item.delivery_status }, { status: 202, headers: { "Cache-Control": "no-store" } });
  } catch {
    return NextResponse.json({ error: "delivery_unavailable" }, { status: 503 });
  }
}
