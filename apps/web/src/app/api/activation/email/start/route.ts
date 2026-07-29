import { NextRequest, NextResponse } from "next/server";
import { callActivation, safeActivationError } from "@/lib/api/activation";
import { validateOrigin } from "@/lib/auth/csrf";
import { ACTIVATION_SMS_PROOF_COOKIE, getActivationProofCookie } from "@/lib/auth/cookies";

export async function POST(request: NextRequest) {
  if (!validateOrigin(request)) return NextResponse.json({ error: "forbidden" }, { status: 403 });
  try {
    const body = await request.json() as { enrollment_number?: unknown };
    const proof = await getActivationProofCookie(ACTIVATION_SMS_PROOF_COOKIE);
    if (typeof body.enrollment_number !== "string" || !/^\d{10}$/.test(body.enrollment_number) || !proof) return NextResponse.json({ error: "activation_failed" }, { status: 401 });
    const response = await callActivation("/v1/activation/email/start", { enrollment_number: body.enrollment_number, sms_proof: proof });
    if (!response.ok && response.status !== 202) return safeActivationError(response.status);
    return NextResponse.json({ status: "accepted" }, { status: 202, headers: { "Cache-Control": "no-store" } });
  } catch {
    return NextResponse.json({ error: "service_unavailable" }, { status: 503 });
  }
}
