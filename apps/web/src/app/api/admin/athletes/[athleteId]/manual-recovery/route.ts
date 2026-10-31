import { NextRequest, NextResponse } from "next/server";
import { getAccessToken } from "@/lib/auth/cookies";
import { validateOrigin } from "@/lib/auth/csrf";
import { getSession } from "@/lib/auth/session";

export async function POST(request: NextRequest, context: { params: Promise<{ athleteId: string }> }) {
  if (!validateOrigin(request)) return NextResponse.json({ error: "forbidden" }, { status: 403 });
  const session = await getSession();
  if (session.status !== "authenticated" || session.user.role !== "owner") return NextResponse.json({ error: "access_denied" }, { status: 403 });
  const token = await getAccessToken();
  if (!token) return NextResponse.json({ error: "authentication_required" }, { status: 401 });
  const { athleteId } = await context.params;
  if (!/^[0-9a-f]{8}-[0-9a-f]{4}-[1-8][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$/i.test(athleteId)) {
    return NextResponse.json({ error: "invalid_request" }, { status: 400 });
  }
  try {
    const apiUrl = process.env.SYSAP_API_BASE_URL || "http://127.0.0.1:8080";
    const response = await fetch(`${apiUrl}/v1/organizations/${encodeURIComponent(session.user.organizationId)}/athletes/${encodeURIComponent(athleteId)}/manual-recovery`, {
      method: "POST",
      headers: { Authorization: `Bearer ${token}`, "X-Organization-ID": session.user.organizationId },
      cache: "no-store",
    });
    if (!response.ok) return NextResponse.json({ error: "issuance_failed" }, { status: response.status });
    const result: unknown = await response.json();
    if (
      typeof result !== "object" ||
      result === null ||
      !("enrollment_number" in result) ||
      typeof result.enrollment_number !== "string" ||
      !/^\d{10}$/.test(result.enrollment_number) ||
      !("code" in result) ||
      typeof result.code !== "string" ||
      !/^\d{6}$/.test(result.code) ||
      !("expires_at" in result) ||
      typeof result.expires_at !== "string" ||
      Number.isNaN(Date.parse(result.expires_at))
    ) {
      return NextResponse.json({ error: "invalid_response" }, { status: 502 });
    }
    return NextResponse.json(
      { enrollment_number: result.enrollment_number, code: result.code, expires_at: result.expires_at },
      { status: 201, headers: { "Cache-Control": "no-store" } },
    );
  } catch {
    return NextResponse.json({ error: "issuance_unknown" }, { status: 503 });
  }
}
