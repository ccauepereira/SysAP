import { NextRequest, NextResponse } from "next/server";
import { callActivation, safeActivationError } from "@/lib/api/activation";
import { validateOrigin } from "@/lib/auth/csrf";
import { ACTIVATION_SMS_PROOF_COOKIE, setActivationProofCookie } from "@/lib/auth/cookies";

export async function POST(request: NextRequest) {
  if (!validateOrigin(request)) return NextResponse.json({ error: "forbidden" }, { status: 403 });
  try {
    const body = await request.json() as { enrollment_number?: unknown; code?: unknown };
    if (typeof body.enrollment_number !== "string" || !/^\d{10}$/.test(body.enrollment_number) || typeof body.code !== "string" || !/^\d{6}$/.test(body.code)) {
      return NextResponse.json({ error: "validation_failed" }, { status: 422 });
    }
    const response = await callActivation("/v1/activation/verify-sms", { enrollment_number: body.enrollment_number, code: body.code });
    if (!response.ok) return safeActivationError(response.status);
    const result = await response.json() as { activation_proof?: unknown };
    if (typeof result.activation_proof !== "string" || result.activation_proof.length < 32) return NextResponse.json({ error: "activation_failed" }, { status: 401 });
    await setActivationProofCookie(ACTIVATION_SMS_PROOF_COOKIE, result.activation_proof);
    return NextResponse.json({ status: "sms_verified" }, { headers: { "Cache-Control": "no-store" } });
  } catch {
    return NextResponse.json({ error: "service_unavailable" }, { status: 503 });
  }
}
