import { NextRequest, NextResponse } from "next/server";
import { validateOrigin } from "@/lib/auth/csrf";
import { cookies } from "next/headers";

export async function POST(request: NextRequest) {
  if (!validateOrigin(request)) return NextResponse.json({ error: "forbidden" }, { status: 403 });
  try {
    const body = await request.json() as { enrollment_number?: unknown; code?: unknown };
    if (typeof body.enrollment_number !== "string" || !/^\d{10}$/.test(body.enrollment_number) || typeof body.code !== "string" || !/^\d{6}$/.test(body.code)) {
      return NextResponse.json({ error: "validation_failed" }, { status: 422 });
    }
    
    const apiUrl = process.env.SYSAP_API_BASE_URL || "http://127.0.0.1:8080";
    const response = await fetch(`${apiUrl}/v1/auth/password-recovery/verify`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ enrollment_number: body.enrollment_number, code: body.code }),
    });
    
    if (!response.ok) return NextResponse.json({ error: "failed" }, { status: response.status });
    const result = await response.json() as { recovery_proof?: string };
    
    if (!result.recovery_proof) return NextResponse.json({ error: "failed" }, { status: 401 });
    
    const isSecure = process.env.SYSAP_COOKIE_SECURE === "true" || process.env.NODE_ENV === "production";
    const cookieStore = await cookies();
    cookieStore.set("sysap-recovery-proof", result.recovery_proof, {
      httpOnly: true,
      secure: isSecure,
      sameSite: "lax",
      path: "/api/auth/password-recovery",
      maxAge: 600,
    });
    
    return NextResponse.json({ status: "verified" }, { headers: { "Cache-Control": "no-store" } });
  } catch {
    return NextResponse.json({ error: "service_unavailable" }, { status: 503 });
  }
}
