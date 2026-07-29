import { NextRequest, NextResponse } from "next/server";

const accessCookieNames = ["sysap-access", "__Host-sysap-access"] as const;
const protectedPathPattern = /^\/(?:atletas|treinos|alertas)(?:\/|$)/;

export function proxy(request: NextRequest) {
  const isProtectedPath = request.nextUrl.pathname === "/" || protectedPathPattern.test(request.nextUrl.pathname);
  if (!isProtectedPath) return NextResponse.next();

  const accessCookie = accessCookieNames
    .map((name) => request.cookies.get(name)?.value)
    .find((value): value is string => value !== undefined);
  const hasPlausibleAccessCookie = accessCookie !== undefined &&
    /^[A-Za-z0-9_-]+\.[A-Za-z0-9_-]+\.[A-Za-z0-9_-]+$/.test(accessCookie);

  if (!hasPlausibleAccessCookie) {
    return NextResponse.redirect(new URL("/login", request.url));
  }

  return NextResponse.next();
}

export const config = {
  matcher: ["/", "/atletas/:path*", "/treinos/:path*", "/alertas/:path*"],
};

export default proxy;
