import { NextRequest, NextResponse } from "next/server";
import { getAccessToken, clearAuthCookies } from "@/lib/auth/cookies";
import { validateOrigin } from "@/lib/auth/csrf";

export async function POST(request: NextRequest) {
  if (!validateOrigin(request)) {
    return NextResponse.json({ error: "forbidden" }, { status: 403 });
  }

  try {
    const accessToken = await getAccessToken();

    if (accessToken) {
      const apiUrl = process.env.SYSAP_API_BASE_URL || "http://127.0.0.1:8080";
      
      await fetch(`${apiUrl}/v1/auth/logout-all`, {
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
