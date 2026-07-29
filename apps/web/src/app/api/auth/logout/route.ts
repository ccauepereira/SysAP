import { NextRequest, NextResponse } from "next/server";
import { getAccessToken, clearAuthCookies } from "@/lib/auth/cookies";
import { validateOrigin } from "@/lib/auth/csrf";

export async function POST(request: NextRequest) {
  if (!validateOrigin(request)) {
    return NextResponse.json({ error: "forbidden" }, { status: 403 });
  }

  try {
    const accessToken = await getAccessToken();

    // Even if we don't have a token, we should clear the cookies on the client side
    // to ensure idempotency.
    if (accessToken) {
      const apiUrl = process.env.SYSAP_API_BASE_URL || "http://127.0.0.1:8080";
      
      // We don't await/care about the result to be completely resilient on the client side
      // However, we wait for the fetch to avoid dropping the connection prematurely.
      // We ignore errors to ensure logout always succeeds from the BFF perspective.
      await fetch(`${apiUrl}/v1/auth/logout`, {
        method: "POST",
        headers: {
          "Authorization": `Bearer ${accessToken}`,
        },
        cache: "no-store",
      }).catch(() => {});
    }

    await clearAuthCookies();

    return new NextResponse(null, { status: 204 });
  } catch {
    await clearAuthCookies();
    return new NextResponse(null, { status: 204 });
  }
}
