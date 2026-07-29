import { NextRequest, NextResponse } from "next/server";
import { getRefreshToken, setAuthCookies, clearAuthCookies } from "@/lib/auth/cookies";
import { validateOrigin } from "@/lib/auth/csrf";

export async function POST(request: NextRequest) {
  if (!validateOrigin(request)) {
    return NextResponse.json({ error: "forbidden" }, { status: 403 });
  }

  try {
    const refreshToken = await getRefreshToken();

    if (!refreshToken) {
      return NextResponse.json({ error: "unauthenticated" }, { status: 401 });
    }

    const apiUrl = process.env.SYSAP_API_BASE_URL || "http://127.0.0.1:8080";
    
    const response = await fetch(`${apiUrl}/v1/auth/refresh`, {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
      },
      body: JSON.stringify({ refresh_token: refreshToken }),
      cache: "no-store",
    });

    if (!response.ok) {
      await clearAuthCookies();
      return NextResponse.json({ error: "refresh_failed" }, { status: 401 });
    }

    const data = await response.json();

    if (!data.access_token || !data.refresh_token) {
      await clearAuthCookies();
      return NextResponse.json({ error: "internal_error" }, { status: 500 });
    }

    await setAuthCookies(data.access_token, data.refresh_token);

    return NextResponse.json(
      { status: "success" },
      { 
        status: 200,
        headers: { "Cache-Control": "no-store" } 
      }
    );

  } catch {
    return NextResponse.json({ error: "service_unavailable" }, { status: 503 });
  }
}
