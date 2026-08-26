import { NextRequest, NextResponse } from "next/server";
import { callActivation, safeActivationError } from "@/lib/api/activation";
import { validateOrigin } from "@/lib/auth/csrf";
import { ACTIVATION_FINAL_PROOF_COOKIE, clearActivationProofCookies, getActivationProofCookie } from "@/lib/auth/cookies";

export async function POST(request: NextRequest) {
  if (!validateOrigin(request)) return NextResponse.json({ error: "forbidden" }, { status: 403 });
  try {
    const body = await request.json() as { password?: unknown; password_confirmation?: unknown };
    const proof = await getActivationProofCookie(ACTIVATION_FINAL_PROOF_COOKIE);
    if (typeof body.password !== "string" || typeof body.password_confirmation !== "string" || body.password !== body.password_confirmation || body.password.length < 15 || !proof) return NextResponse.json({ error: "validation_failed" }, { status: 422 });
    const response = await callActivation("/v1/activation/complete", { activation_proof: proof, password: body.password });
    if (!response.ok) return safeActivationError(response.status);
    await clearActivationProofCookies();
    return new NextResponse(null, { status: 204, headers: { "Cache-Control": "no-store" } });
  } catch {
    return NextResponse.json({ error: "service_unavailable" }, { status: 503 });
  }
}
