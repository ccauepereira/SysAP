import "server-only";
import { cookies } from "next/headers";

const isSecure = process.env.SYSAP_COOKIE_SECURE === "true" || process.env.NODE_ENV === "production";
const cookiePrefix = isSecure ? "__Host-" : "";

export const ACCESS_TOKEN_COOKIE = `${cookiePrefix}sysap-access`;
export const REFRESH_TOKEN_COOKIE = `${cookiePrefix}sysap-refresh`;

export async function setAuthCookies(accessToken: string, refreshToken: string) {
  const cookieStore = await cookies();

  // Access token cookie
  cookieStore.set(ACCESS_TOKEN_COOKIE, accessToken, {
    httpOnly: true,
    secure: isSecure,
    sameSite: "lax",
    path: "/",
  });

  // Refresh token cookie
  cookieStore.set(REFRESH_TOKEN_COOKIE, refreshToken, {
    httpOnly: true,
    secure: isSecure,
    sameSite: "lax",
    path: "/api/auth",
  });
}

export async function clearAuthCookies() {
  const cookieStore = await cookies();

  cookieStore.delete({
    name: ACCESS_TOKEN_COOKIE,
    httpOnly: true,
    secure: isSecure,
    sameSite: "lax",
    path: "/",
  });

  cookieStore.delete({
    name: REFRESH_TOKEN_COOKIE,
    httpOnly: true,
    secure: isSecure,
    sameSite: "lax",
    path: "/api/auth",
  });
}

export async function getAccessToken(): Promise<string | undefined> {
  const cookieStore = await cookies();
  return cookieStore.get(ACCESS_TOKEN_COOKIE)?.value;
}

export async function getRefreshToken(): Promise<string | undefined> {
  const cookieStore = await cookies();
  return cookieStore.get(REFRESH_TOKEN_COOKIE)?.value;
}
