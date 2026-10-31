import { NextRequest, NextResponse } from "next/server";
import { getAccessToken } from "@/lib/auth/cookies";
import { getSession } from "@/lib/auth/session";

const uuidPattern = /^[0-9a-f]{8}-[0-9a-f]{4}-[1-8][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$/i;

function safeStatus(value: unknown) {
  const item = value as Record<string, unknown>;
  return {
    invitation_id: item.invitation_id,
    profile_id: item.profile_id,
    status: item.status,
    expires_at: item.expires_at,
    delivery_channel: item.delivery_channel,
    delivery_status: item.delivery_status,
  };
}

export async function GET(_request: NextRequest, context: { params: Promise<{ invitationId: string }> }) {
  const token = await getAccessToken();
  if (!token) return NextResponse.json({ error: "authentication_required" }, { status: 401 });
  const session = await getSession();
  if (session.status === "unauthenticated") return NextResponse.json({ error: "authentication_required" }, { status: 401 });
  if (session.status === "error") return NextResponse.json({ error: "invitation_status_unavailable" }, { status: 503 });
  if (session.user.role !== "owner") return NextResponse.json({ error: "access_denied" }, { status: 403 });
  const { invitationId } = await context.params;
  if (!uuidPattern.test(invitationId)) return NextResponse.json({ error: "resource_not_found" }, { status: 404 });
  try {
    const apiUrl = process.env.SYSAP_API_BASE_URL || "http://127.0.0.1:8080";
    const response = await fetch(`${apiUrl}/v1/athlete-invitations/${invitationId}`, {
      headers: { Authorization: `Bearer ${token}`, "X-Organization-ID": session.user.organizationId },
      cache: "no-store",
    });
    if (!response.ok) return NextResponse.json({ error: response.status === 404 ? "resource_not_found" : "invitation_status_unavailable" }, { status: response.status });
    return NextResponse.json(safeStatus(await response.json()), { headers: { "Cache-Control": "no-store" } });
  } catch {
    return NextResponse.json({ error: "invitation_status_unavailable" }, { status: 503 });
  }
}
