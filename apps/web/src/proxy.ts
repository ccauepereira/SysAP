import { NextRequest, NextResponse } from "next/server";

const accessCookieNames = ["sysap-access", "__Host-sysap-access"] as const;
const protectedPathPattern = /^\/(?:admin|atleta|atletas|treinos|financeiro|alertas)(?:\/|$)/;

export function proxy(request: NextRequest) {
  if (/^\/demonstracao(?:\/|$)/.test(request.nextUrl.pathname)) {
    const localHost = /^(localhost|127\.0\.0\.1)(:\d+)?$/.test((request.headers.get("host") ?? "").toLowerCase());
    if (process.env.NODE_ENV !== "development" || process.env.SYSAP_LOCAL_DEMO !== "1" || !localHost) {
      return new NextResponse(null, { status: 404 });
    }
  }
  if (process.env.NODE_ENV === "development" && process.env.SYSAP_LOCAL_DEMO === "1" && request.nextUrl.pathname.startsWith("/api/")) {
    return new NextResponse(null, { status: 404 });
  }
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
  matcher: ["/", "/admin/:path*", "/atleta/:path*", "/atletas/:path*", "/treinos/:path*", "/financeiro/:path*", "/alertas/:path*", "/api/:path*", "/demonstracao/:path*"],
};

export default proxy;
