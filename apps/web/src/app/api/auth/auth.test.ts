import { describe, it, expect, vi, beforeEach } from "vitest";
import { POST as LoginPOST } from "./login/route";
import { POST as RefreshPOST } from "./refresh/route";
import { POST as LogoutPOST } from "./logout/route";
import { POST as LogoutAllPOST } from "./logout-all/route";
import { GET as SessionGET } from "./session/route";
import { NextRequest } from "next/server";
import * as cookiesModule from "@/lib/auth/cookies";

// Mock the cookies module since Next.js cookies() API is hard to mock outside of a real server
vi.mock("@/lib/auth/cookies", async (importOriginal) => {
  const actual = await importOriginal<typeof import("@/lib/auth/cookies")>();
  return {
    ...actual,
    setAuthCookies: vi.fn(),
    clearAuthCookies: vi.fn(),
    getAccessToken: vi.fn(),
    getRefreshToken: vi.fn(),
  };
});

describe("BFF Auth Routes", () => {
  const mockOrigin = "http://localhost:3000";

  beforeEach(() => {
    vi.restoreAllMocks();
    process.env.SYSAP_WEB_ORIGIN = mockOrigin;
    process.env.SYSAP_API_BASE_URL = "http://api.example.com";
    process.env.SYSAP_COOKIE_SECURE = "false";
    global.fetch = vi.fn();
  });

  function createRequest(url: string, method: string, body?: unknown, origin: string | null = mockOrigin) {
    const headers = new Headers();
    if (origin) {
      headers.set("origin", origin);
    }
    if (body) {
      headers.set("content-type", "application/json");
    }
    return new NextRequest(new URL(url, "http://localhost:3000"), {
      method,
      headers,
      body: body ? JSON.stringify(body) : null,
    });
  }

  describe("POST /api/auth/login", () => {
    it("fails securely on missing or invalid Origin", async () => {
      const reqNoOrigin = createRequest("/api/auth/login", "POST", { enrollment_number: "123", password: "pwd" }, null);
      const res1 = await LoginPOST(reqNoOrigin);
      expect(res1.status).toBe(403);
      expect(await res1.json()).toEqual({ error: "forbidden" });

      const reqBadOrigin = createRequest("/api/auth/login", "POST", { enrollment_number: "123", password: "pwd" }, "http://attacker.com");
      const res2 = await LoginPOST(reqBadOrigin);
      expect(res2.status).toBe(403);
    });

    it("succeeds and sets cookies, stripping tokens from response", async () => {
      vi.mocked(global.fetch).mockResolvedValueOnce(new Response(JSON.stringify({
        session_id: "sess-1",
        profile_id: "prof-1",
        organization_id: "org-1",
        role: "athlete",
        aal: "aal1",
        access_token: "access-token-123",
        refresh_token: "refresh-token-123",
        expires_in: 3600
      }), { status: 200 }));

      const req = createRequest("/api/auth/login", "POST", { enrollment_number: "2026000001", password: "pwd" });
      const res = await LoginPOST(req);
      
      expect(res.status).toBe(200);
      expect(res.headers.get("Cache-Control")).toBe("no-store");
      
      const json = await res.json();
      expect(json).toEqual({
        session_id: "sess-1",
        profile_id: "prof-1",
        organization_id: "org-1",
        role: "athlete",
        aal: "aal1"
      });
      expect(json.access_token).toBeUndefined();

      expect(cookiesModule.setAuthCookies).toHaveBeenCalledWith("access-token-123", "refresh-token-123");
    });

    it("handles login failure securely without leaking details", async () => {
      vi.mocked(global.fetch).mockResolvedValueOnce(new Response(JSON.stringify({
        error: { code: "invalid_credentials", message: "invalid credentials", request_id: "req-1" }
      }), { status: 401 }));

      const req = createRequest("/api/auth/login", "POST", { enrollment_number: "2026000001", password: "pwd" });
      const res = await LoginPOST(req);
      
      expect(res.status).toBe(401);
      const json = await res.json();
      expect(json).toEqual({ error: "invalid_credentials" });
      expect(json.request_id).toBeUndefined();
    });

    it("handles API 500 error securely", async () => {
      vi.mocked(global.fetch).mockResolvedValueOnce(new Response("Internal Server Error", { status: 500 }));

      const req = createRequest("/api/auth/login", "POST", { enrollment_number: "2026000001", password: "pwd" });
      const res = await LoginPOST(req);
      
      expect(res.status).toBe(500);
      expect(await res.json()).toEqual({ error: "auth_failed" });
    });
  });

  describe("POST /api/auth/refresh", () => {
    it("fails without refresh token", async () => {
      vi.mocked(cookiesModule.getRefreshToken).mockResolvedValue(undefined);
      const req = createRequest("/api/auth/refresh", "POST", {});
      const res = await RefreshPOST(req);
      expect(res.status).toBe(401);
    });

    it("rotates cookies and does not leak tokens", async () => {
      vi.mocked(cookiesModule.getRefreshToken).mockResolvedValue("old-refresh-token");
      vi.mocked(global.fetch).mockResolvedValueOnce(new Response(JSON.stringify({
        access_token: "new-access",
        refresh_token: "new-refresh",
        token_type: "Bearer",
        expires_in: 3600
      }), { status: 200 }));

      const req = createRequest("/api/auth/refresh", "POST", {});
      const res = await RefreshPOST(req);
      
      expect(res.status).toBe(200);
      expect(await res.json()).toEqual({ status: "success" });
      expect(cookiesModule.setAuthCookies).toHaveBeenCalledWith("new-access", "new-refresh");
    });

    it("clears cookies on failure", async () => {
      vi.mocked(cookiesModule.getRefreshToken).mockResolvedValue("old-refresh-token");
      vi.mocked(global.fetch).mockResolvedValueOnce(new Response("Unauthorized", { status: 401 }));

      const req = createRequest("/api/auth/refresh", "POST", {});
      const res = await RefreshPOST(req);
      
      expect(res.status).toBe(401);
      expect(cookiesModule.clearAuthCookies).toHaveBeenCalled();
    });
  });

  describe("POST /api/auth/logout", () => {
    it("calls API and clears cookies", async () => {
      vi.mocked(cookiesModule.getAccessToken).mockResolvedValue("my-token");
      vi.mocked(global.fetch).mockResolvedValueOnce(new Response(null, { status: 204 }));

      const req = createRequest("/api/auth/logout", "POST", {});
      const res = await LogoutPOST(req);

      expect(res.status).toBe(204);
      expect(global.fetch).toHaveBeenCalledWith(
        "http://api.example.com/v1/auth/logout",
        expect.objectContaining({ headers: { Authorization: "Bearer my-token" }})
      );
      expect(cookiesModule.clearAuthCookies).toHaveBeenCalled();
    });

    it("clears cookies even if API fails", async () => {
      vi.mocked(cookiesModule.getAccessToken).mockResolvedValue("my-token");
      vi.mocked(global.fetch).mockRejectedValueOnce(new Error("Network Error"));

      const req = createRequest("/api/auth/logout", "POST", {});
      const res = await LogoutPOST(req);

      expect(res.status).toBe(204);
      expect(cookiesModule.clearAuthCookies).toHaveBeenCalled();
    });
  });

  describe("POST /api/auth/logout-all", () => {
    it("calls API and clears cookies", async () => {
      vi.mocked(cookiesModule.getAccessToken).mockResolvedValue("my-token");
      vi.mocked(global.fetch).mockResolvedValueOnce(new Response(null, { status: 204 }));

      const req = createRequest("/api/auth/logout-all", "POST", {});
      const res = await LogoutAllPOST(req);

      expect(res.status).toBe(204);
      expect(global.fetch).toHaveBeenCalledWith(
        "http://api.example.com/v1/auth/logout-all",
        expect.objectContaining({ headers: { Authorization: "Bearer my-token" }})
      );
      expect(cookiesModule.clearAuthCookies).toHaveBeenCalled();
    });
  });

  describe("GET /api/auth/session", () => {
    it("returns safe unauthenticated state if no cookie", async () => {
      vi.mocked(cookiesModule.getAccessToken).mockResolvedValue(undefined);
      
      // createRequest is not needed since GET doesn't use the request object
      const res = await SessionGET();
      
      expect(res.status).toBe(200);
      expect(await res.json()).toEqual({ authenticated: false, reason: "no_session" });
    });

    it("fetches session and returns safe public data", async () => {
      vi.mocked(cookiesModule.getAccessToken).mockResolvedValue("access-token-123");
      vi.mocked(global.fetch).mockResolvedValueOnce(new Response(JSON.stringify({
        profile: { id: "p1", display_name: "Artur" },
        memberships: [{ organization_id: "o1", role: "owner", status: "active" }]
      }), { status: 200 }));

      // createRequest is not needed since GET doesn't use the request object
      const res = await SessionGET();
      
      expect(res.status).toBe(200);
      expect(await res.json()).toEqual({
        authenticated: true,
        user: { id: "p1", name: "Artur", role: "owner", organizationId: "o1" }
      });
    });

    it("handles 401 as expired", async () => {
      vi.mocked(cookiesModule.getAccessToken).mockResolvedValue("access-token-123");
      vi.mocked(global.fetch).mockResolvedValueOnce(new Response(null, { status: 401 }));

      // createRequest is not needed since GET doesn't use the request object
      const res = await SessionGET();
      
      expect(res.status).toBe(200);
      expect(await res.json()).toEqual({ authenticated: false, reason: "expired" });
    });
  });
});
