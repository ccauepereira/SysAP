import { NextRequest, NextResponse } from "next/server";
import { validateOrigin } from "@/lib/auth/csrf";

export async function POST(request: NextRequest) {
  if (!validateOrigin(request)) return NextResponse.json({ error: "forbidden" }, { status: 403 });
  try {
    const body = await request.json() as { enrollment_number?: unknown };
    if (typeof body.enrollment_number !== "string" || !/^\d{10}$/.test(body.enrollment_number)) {
      return NextResponse.json({ error: "validation_failed" }, { status: 422 });
    }
    const apiUrl = process.env.SYSAP_API_BASE_URL || "http://127.0.0.1:8080";
    const response = await fetch(`${apiUrl}/v1/auth/password-recovery/start`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ enrollment_number: body.enrollment_number }),
    });
    if (!response.ok && response.status !== 202) return NextResponse.json({ error: "failed" }, { status: response.status });
    return NextResponse.json({ status: "accepted" }, { status: 202, headers: { "Cache-Control": "no-store" } });
  } catch {
    return NextResponse.json({ error: "service_unavailable" }, { status: 503 });
  }
}
