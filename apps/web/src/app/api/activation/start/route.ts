import { NextRequest, NextResponse } from "next/server";
import { callActivation, safeActivationError } from "@/lib/api/activation";
import { validateOrigin } from "@/lib/auth/csrf";

export async function POST(request: NextRequest) {
  if (!validateOrigin(request)) return NextResponse.json({ error: "forbidden" }, { status: 403 });
  try {
    const body = await request.json() as { enrollment_number?: unknown };
    if (typeof body.enrollment_number !== "string" || !/^\d{10}$/.test(body.enrollment_number)) {
      return NextResponse.json({ error: "validation_failed" }, { status: 422 });
    }
    const response = await callActivation("/v1/activation/start", { enrollment_number: body.enrollment_number });
    if (!response.ok && response.status !== 202) return safeActivationError(response.status);
    return NextResponse.json({ status: "accepted" }, { status: 202, headers: { "Cache-Control": "no-store" } });
  } catch {
    return NextResponse.json({ error: "service_unavailable" }, { status: 503 });
  }
}
