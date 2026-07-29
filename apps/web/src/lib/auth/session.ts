import "server-only";
import { getAccessToken, clearAuthCookies } from "./cookies";
import { redirect } from "next/navigation";

export type SessionState =
  | { status: "authenticated"; user: { id: string; name: string; role: string; organizationId: string } }
  | { status: "unauthenticated"; reason: "no_session" | "expired" | "revoked" }
  | { status: "error"; reason: "api_unavailable" };

interface ApiIdentityResponse {
  profile: {
    id: string;
    display_name: string;
    status?: string;
  };
  memberships: Array<{
    organization_id: string;
    role: string;
    status: string;
  }>;
}

export async function getSession(): Promise<SessionState> {
  const token = await getAccessToken();

  if (!token) {
    return { status: "unauthenticated", reason: "no_session" };
  }

  const apiUrl = process.env.SYSAP_API_BASE_URL || "http://127.0.0.1:8080";

  try {
    const response = await fetch(`${apiUrl}/v1/me`, {
      method: "GET",
      headers: {
        Authorization: `Bearer ${token}`,
      },
      cache: "no-store",
    });

    if (response.ok) {
      const data = (await response.json()) as ApiIdentityResponse;
      if (data.profile.status !== undefined && data.profile.status !== "active") {
        return { status: "unauthenticated", reason: "revoked" };
      }
      const activeMembership = data.memberships.find((m) => m.status === "active");

      if (!activeMembership) {
        return { status: "unauthenticated", reason: "revoked" };
      }

      return {
        status: "authenticated",
        user: {
          id: data.profile.id,
          name: data.profile.display_name,
          role: activeMembership.role,
          organizationId: activeMembership.organization_id,
        },
      };
    }

    if (response.status === 401) {
      return { status: "unauthenticated", reason: "expired" };
    }

    if (response.status === 403) {
      return { status: "unauthenticated", reason: "revoked" };
    }

    // Treat other errors (5xx, 429) as API unavailable or generic error
    return { status: "error", reason: "api_unavailable" };
  } catch {
    // Network errors (e.g. ECONNREFUSED)
    return { status: "error", reason: "api_unavailable" };
  }
}

export async function requireSession(redirectTo = "/login") {
  const session = await getSession();

  if (session.status !== "authenticated") {
    // Optional: if it's expired/revoked, we could clear the cookies here
    // But typically clearing cookies happens when a request actually fails or in middleware.
    // We'll clear them just in case they are invalid.
    if (session.status === "unauthenticated" && (session.reason === "expired" || session.reason === "revoked")) {
      await clearAuthCookies();
    }
    
    // Redirect to login page
    redirect(redirectTo);
  }

  return session.user;
}
