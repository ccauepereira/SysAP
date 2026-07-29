import { NextRequest, NextResponse } from "next/server";
import { setAuthCookies } from "@/lib/auth/cookies";
import { validateOrigin } from "@/lib/auth/csrf";

export async function POST(request: NextRequest) {
  if (!validateOrigin(request)) {
    return NextResponse.json({ error: "forbidden" }, { status: 403 });
  }

  try {
    const body = await request.json();
    const { enrollment_number, password } = body;

    if (!enrollment_number || !password) {
      return NextResponse.json({ error: "bad_request" }, { status: 400 });
    }

    const apiUrl = process.env.SYSAP_API_BASE_URL || "http://127.0.0.1:8080";
    
    // Call the Go API
    const response = await fetch(`${apiUrl}/v1/auth/login`, {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
      },
      body: JSON.stringify({ enrollment_number, password }),
      cache: "no-store",
    });

    if (!response.ok) {
      // Don't forward raw response. Just return generic error.
      if (response.status === 401) {
        return NextResponse.json({ error: "invalid_credentials" }, { status: 401 });
      }
      return NextResponse.json({ error: "auth_failed" }, { status: response.status });
    }

    const data = await response.json();

    // Verify expected payload from API
    if (!data.access_token || !data.refresh_token) {
      return NextResponse.json({ error: "internal_error" }, { status: 500 });
    }

    // Set secure cookies
    await setAuthCookies(data.access_token, data.refresh_token);

    // Return safe data (no tokens)
    return NextResponse.json(
      {
        session_id: data.session_id,
        profile_id: data.profile_id,
        organization_id: data.organization_id,
        role: data.role,
        aal: data.aal,
      },
      { 
        status: 200,
        headers: { "Cache-Control": "no-store" } 
      }
    );

  } catch {
    // Handle timeout/network errors securely
    return NextResponse.json({ error: "service_unavailable" }, { status: 503 });
  }
}
