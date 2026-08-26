import { NextRequest, NextResponse } from "next/server";
import { validateOrigin } from "@/lib/auth/csrf";
import { cookies } from "next/headers";

export async function POST(request: NextRequest) {
  if (!validateOrigin(request)) return NextResponse.json({ error: "forbidden" }, { status: 403 });
  try {
    const body = await request.json() as { password?: unknown; password_confirmation?: unknown };
    
    if (typeof body.password !== "string" || body.password.length < 15 || body.password !== body.password_confirmation) {
      return NextResponse.json({ error: "validation_failed" }, { status: 422 });
    }
    
    const cookieStore = await cookies();
    const proof = cookieStore.get("sysap-recovery-proof")?.value;
    if (!proof) return NextResponse.json({ error: "unauthorized" }, { status: 401 });
    
    const apiUrl = process.env.SYSAP_API_BASE_URL || "http://127.0.0.1:8080";
    const response = await fetch(`${apiUrl}/v1/auth/password-recovery/complete`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ password: body.password, password_confirmation: body.password_confirmation, recovery_proof: proof }),
    });
    
    if (!response.ok) return NextResponse.json({ error: "failed" }, { status: response.status });
    
    cookieStore.delete({
      name: "sysap-recovery-proof",
      path: "/api/auth/password-recovery",
    });
    
    return NextResponse.json({ status: "completed" }, { headers: { "Cache-Control": "no-store" } });
  } catch {
    return NextResponse.json({ error: "service_unavailable" }, { status: 503 });
  }
}
