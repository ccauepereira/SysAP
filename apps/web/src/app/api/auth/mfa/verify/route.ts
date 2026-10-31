import { NextRequest, NextResponse } from "next/server";
import { getAccessToken, setAuthCookies } from "@/lib/auth/cookies";
import { validateOrigin } from "@/lib/auth/csrf";

export async function POST(request: NextRequest) {
  if (!validateOrigin(request)) {
    return NextResponse.json({ error: "forbidden" }, { status: 403 });
  }

  const token = await getAccessToken();
  if (!token) {
    return NextResponse.json({ error: "authentication_required" }, { status: 401 });
  }

  try {
    const body = (await request.json()) as { factor_id?: unknown; code?: unknown };
    const factorId = typeof body.factor_id === "string" ? body.factor_id.trim() : "";
    const code = typeof body.code === "string" ? body.code.trim() : "";

    if (!factorId || !/^\d{6}$/.test(code)) {
      return NextResponse.json({ error: "validation_failed" }, { status: 422 });
    }

    const apiUrl = process.env.SYSAP_API_BASE_URL || "http://127.0.0.1:8080";

    const challengeResponse = await fetch(`${apiUrl}/v1/auth/mfa/challenge`, {
      method: "POST",
      headers: {
        Authorization: `Bearer ${token}`,
        "Content-Type": "application/json",
      },
      body: JSON.stringify({ factor_id: factorId }),
      cache: "no-store",
    });

    if (!challengeResponse.ok) {
      return NextResponse.json(
        { error: "mfa_challenge_failed" },
        { status: challengeResponse.status === 401 ? 401 : 503 },
      );
    }

    const challengeData = (await challengeResponse.json()) as {
      challenge_id?: string;
    };

    if (!challengeData.challenge_id) {
      return NextResponse.json({ error: "internal_error" }, { status: 500 });
    }

    const verifyResponse = await fetch(`${apiUrl}/v1/auth/mfa/verify`, {
      method: "POST",
      headers: {
        Authorization: `Bearer ${token}`,
        "Content-Type": "application/json",
      },
      body: JSON.stringify({
        factor_id: factorId,
        challenge_id: challengeData.challenge_id,
        code,
      }),
      cache: "no-store",
    });

    if (!verifyResponse.ok) {
      return NextResponse.json(
        { error: verifyResponse.status === 401 ? "invalid_code" : "mfa_verify_failed" },
        { status: verifyResponse.status },
      );
    }

    const verifyData = (await verifyResponse.json()) as {
      aal?: string;
      access_token?: string;
      refresh_token?: string;
    };

    if (!verifyData.access_token || !verifyData.refresh_token) {
      return NextResponse.json({ error: "internal_error" }, { status: 500 });
    }

    await setAuthCookies(verifyData.access_token, verifyData.refresh_token);

    return NextResponse.json(
      {
        status: "verified",
        aal: verifyData.aal ?? "aal2",
      },
      { status: 200, headers: { "Cache-Control": "no-store" } },
    );
  } catch {
    return NextResponse.json({ error: "service_unavailable" }, { status: 503 });
  }
}
