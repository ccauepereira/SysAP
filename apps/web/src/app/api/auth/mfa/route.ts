import { NextResponse } from "next/server";
import { getAccessToken } from "@/lib/auth/cookies";

function extractSecretFromURI(uri: string): string | undefined {
  try {
    const parsed = new URL(uri);
    return parsed.searchParams.get("secret") ?? undefined;
  } catch {
    return undefined;
  }
}

export async function GET() {
  const token = await getAccessToken();
  if (!token) {
    return NextResponse.json({ error: "authentication_required" }, { status: 401 });
  }

  const apiUrl = process.env.SYSAP_API_BASE_URL || "http://127.0.0.1:8080";

  try {
    const statusResponse = await fetch(`${apiUrl}/v1/auth/mfa`, {
      method: "GET",
      headers: {
        Authorization: `Bearer ${token}`,
      },
      cache: "no-store",
    });

    if (!statusResponse.ok) {
      return NextResponse.json(
        { error: "mfa_unavailable" },
        { status: statusResponse.status === 401 ? 401 : 503 },
      );
    }

    const statusData = (await statusResponse.json()) as {
      mfa_enabled?: boolean;
      factor_id?: string;
    };

    if (statusData.mfa_enabled && statusData.factor_id) {
      return NextResponse.json(
        {
          status: "challenge_ready",
          mfa_enabled: true,
          factor_id: statusData.factor_id,
        },
        { status: 200, headers: { "Cache-Control": "no-store" } },
      );
    }

    const enrollResponse = await fetch(`${apiUrl}/v1/auth/mfa/enroll`, {
      method: "POST",
      headers: {
        Authorization: `Bearer ${token}`,
        "Content-Type": "application/json",
      },
      cache: "no-store",
    });

    if (!enrollResponse.ok) {
      return NextResponse.json(
        { error: "mfa_enroll_failed" },
        { status: enrollResponse.status === 401 ? 401 : 503 },
      );
    }

    const enrollData = (await enrollResponse.json()) as {
      factor_id?: string;
      provisioning_uri?: string;
    };

    if (!enrollData.factor_id || !enrollData.provisioning_uri) {
      return NextResponse.json({ error: "internal_error" }, { status: 500 });
    }

    return NextResponse.json(
      {
        status: "enroll_required",
        mfa_enabled: false,
        factor_id: enrollData.factor_id,
        provisioning_uri: enrollData.provisioning_uri,
        secret: extractSecretFromURI(enrollData.provisioning_uri),
      },
      { status: 200, headers: { "Cache-Control": "no-store" } },
    );
  } catch {
    return NextResponse.json({ error: "service_unavailable" }, { status: 503 });
  }
}
