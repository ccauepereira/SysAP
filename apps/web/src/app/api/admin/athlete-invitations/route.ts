import { NextRequest, NextResponse } from "next/server";
import { getAccessToken } from "@/lib/auth/cookies";
import { getSession } from "@/lib/auth/session";
import { validateOrigin } from "@/lib/auth/csrf";

function normalizePhoneE164(raw: string): string {
  const trimmed = raw.trim();
  if (!trimmed) return "";
  const hasPlus = trimmed.startsWith("+");
  if (/[^0-9\s\-().+]/.test(trimmed)) return trimmed;
  const digits = trimmed.replace(/\D/g, "");
  if (digits.length < 8 || digits.length > 15) return trimmed;
  if (hasPlus) return `+${digits}`;
  if (digits.length === 10 || digits.length === 11) return `+55${digits}`;
  if ((digits.length === 12 || digits.length === 13) && digits.startsWith("55")) return `+${digits}`;
  return trimmed;
}

export async function POST(request: NextRequest) {
  if (!validateOrigin(request)) return NextResponse.json({ error: "forbidden" }, { status: 403 });
  const session = await getSession();
  if (session.status !== "authenticated" || session.user.role !== "owner") return NextResponse.json({ error: "access_denied" }, { status: 403 });
  const token = await getAccessToken();
  if (!token) return NextResponse.json({ error: "authentication_required" }, { status: 401 });
  try {
    const body = await request.json() as Record<string, unknown>;
    const idempotencyKey = request.headers.get("Idempotency-Key");
    if (!idempotencyKey || !/^[0-9a-f]{8}-[0-9a-f]{4}-[1-8][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$/i.test(idempotencyKey)) return NextResponse.json({ error: "validation_failed" }, { status: 400 });
    const required = ["full_name", "phone_e164", "email", "birth_date", "locality", "team_id", "football_position", "enrollment_delivery_channel"];
    if (required.some((key) => typeof body[key] !== "string" || String(body[key]).trim() === "")) return NextResponse.json({ error: "validation_failed" }, { status: 422 });
    if (body.enrollment_delivery_channel !== "email" && body.enrollment_delivery_channel !== "manual") return NextResponse.json({ error: "validation_failed" }, { status: 422 });
    const normalizedBody = { ...body, phone_e164: normalizePhoneE164(String(body.phone_e164)) };
    const apiUrl = process.env.SYSAP_API_BASE_URL || "http://127.0.0.1:8080";
    const response = await fetch(`${apiUrl}/v1/organizations/${session.user.organizationId}/athlete-invitations`, { method: "POST", headers: { Authorization: `Bearer ${token}`, "Content-Type": "application/json", "X-Organization-ID": session.user.organizationId, "Idempotency-Key": idempotencyKey }, body: JSON.stringify(normalizedBody), cache: "no-store" });
    if (!response.ok) {
      let apiCode: string | undefined;
      try {
        const errPayload = await response.json() as { error?: { code?: string } };
        apiCode = errPayload.error?.code;
      } catch {
        /* ignore */
      }
      return NextResponse.json({ error: apiCode || (response.status === 503 ? "service_unavailable" : response.status === 403 ? "access_denied" : "invitation_failed") }, { status: response.status });
    }
    const result = await response.json() as { invitation_id?: string; profile_id?: string; enrollment_number?: string; status?: string; expires_at?: string; delivery_channel?: string; delivery_status?: string };
    return NextResponse.json({
      invitation_id: result.invitation_id,
      profile_id: result.profile_id,
      ...(typeof result.enrollment_number === "string" && /^\d{10}$/.test(result.enrollment_number) ? { enrollment_number: result.enrollment_number } : {}),
      status: result.status,
      expires_at: result.expires_at,
      delivery_channel: result.delivery_channel,
      delivery_status: result.delivery_status,
    }, { status: 201, headers: { "Cache-Control": "no-store" } });
  } catch {
    return NextResponse.json({ error: "service_unavailable" }, { status: 503 });
  }
}
