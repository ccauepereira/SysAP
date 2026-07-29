import "server-only";
import { cookies } from "next/headers";

const isSecure = process.env.SYSAP_COOKIE_SECURE === "true" || process.env.NODE_ENV === "production";
const cookiePrefix = isSecure ? "__Host-" : "";

export const ACCESS_TOKEN_COOKIE = `${cookiePrefix}sysap-access`;
export const REFRESH_TOKEN_COOKIE = `${cookiePrefix}sysap-refresh`;
export const ACTIVATION_SMS_PROOF_COOKIE = `${cookiePrefix}sysap-activation-sms-proof`;
export const ACTIVATION_FINAL_PROOF_COOKIE = `${cookiePrefix}sysap-activation-final-proof`;

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

export async function setActivationProofCookie(name: string, value: string) {
  const cookieStore = await cookies();
  cookieStore.set(name, value, {
    httpOnly: true,
    secure: isSecure,
    sameSite: "lax",
    path: "/api/activation",
    maxAge: 600,
  });
}

export async function getActivationProofCookie(name: string): Promise<string | undefined> {
  const cookieStore = await cookies();
  return cookieStore.get(name)?.value;
}

export async function clearActivationProofCookies() {
  const cookieStore = await cookies();
  for (const name of [ACTIVATION_SMS_PROOF_COOKIE, ACTIVATION_FINAL_PROOF_COOKIE]) {
    cookieStore.delete({ name, httpOnly: true, secure: isSecure, sameSite: "lax", path: "/api/activation" });
  }
}
