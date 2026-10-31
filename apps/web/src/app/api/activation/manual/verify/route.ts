import { NextRequest, NextResponse } from "next/server";
import { callActivation, safeActivationError } from "@/lib/api/activation";
import { validateOrigin } from "@/lib/auth/csrf";
import { ACTIVATION_FINAL_PROOF_COOKIE, clearActivationProofCookies, setActivationProofCookie } from "@/lib/auth/cookies";

export async function POST(request: NextRequest) {
  if (!validateOrigin(request)) return NextResponse.json({ error: "forbidden" }, { status: 403 });
  try {
    const body: unknown = await request.json();
    if (typeof body !== "object" || body === null || !("enrollment_number" in body) || typeof body.enrollment_number !== "string" || !/^\d{10}$/.test(body.enrollment_number) || !("code" in body) || typeof body.code !== "string" || !/^\d{6}$/.test(body.code)) return NextResponse.json({ error: "activation_failed" }, { status: 401 });
    const response = await callActivation("/v1/activation/manual/verify", { enrollment_number: body.enrollment_number, code: body.code });
    if (!response.ok) return safeActivationError(response.status);
    const result: unknown = await response.json();
    if (typeof result !== "object" || result === null || !("activation_proof" in result) || typeof result.activation_proof !== "string" || !/^[0-9a-f]{48}$/.test(result.activation_proof)) return NextResponse.json({ error: "activation_failed" }, { status: 401 });
    await clearActivationProofCookies();
    await setActivationProofCookie(ACTIVATION_FINAL_PROOF_COOKIE, result.activation_proof);
    return NextResponse.json({ status: "verified" }, { headers: { "Cache-Control": "no-store" } });
  } catch { return NextResponse.json({ error: "service_unavailable" }, { status: 503 }); }
}
