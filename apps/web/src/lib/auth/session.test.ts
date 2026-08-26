import { beforeEach, describe, expect, it, vi } from "vitest";
import { clearAuthCookies, getAccessToken } from "./cookies";
import { getSession, requireSession } from "./session";

const { redirect } = vi.hoisted(() => ({
  redirect: vi.fn((target: string): never => {
    throw new Error(`NEXT_REDIRECT:${target}`);
  }),
}));

vi.mock("./cookies", () => ({
  clearAuthCookies: vi.fn(),
  getAccessToken: vi.fn(),
}));

vi.mock("next/navigation", () => ({ redirect }));

beforeEach(() => {
  vi.restoreAllMocks();
  vi.mocked(getAccessToken).mockResolvedValue(undefined);
  vi.mocked(clearAuthCookies).mockResolvedValue(undefined);
  redirect.mockClear();
  vi.stubGlobal("fetch", vi.fn());
});

describe("server session guard", () => {
  it("treats an absent cookie as unauthenticated", async () => {
    await expect(getSession()).resolves.toEqual({ status: "unauthenticated", reason: "no_session" });
  });

  it.each([
    [401, "expired"],
    [403, "revoked"],
  ] as const)("maps BFF status %s to %s", async (status, reason) => {
    vi.mocked(getAccessToken).mockResolvedValue("invalid-cookie");
    vi.mocked(fetch).mockResolvedValueOnce(new Response(null, { status }));

    await expect(getSession()).resolves.toEqual({ status: "unauthenticated", reason });
  });

  it("treats an active membership as an authenticated session", async () => {
    vi.mocked(getAccessToken).mockResolvedValue("valid-cookie");
    vi.mocked(fetch).mockResolvedValueOnce(new Response(JSON.stringify({
      profile: { id: "profile", display_name: "Example" },
      memberships: [{ organization_id: "organization", role: "athlete", status: "active" }],
    }), { status: 200 }));

    await expect(getSession()).resolves.toEqual({
      status: "authenticated",
      user: { id: "profile", name: "Example", role: "athlete", organizationId: "organization" },
    });
  });

  it("rejects an explicitly suspended profile", async () => {
    vi.mocked(getAccessToken).mockResolvedValue("suspended-cookie");
    vi.mocked(fetch).mockResolvedValueOnce(new Response(JSON.stringify({
      profile: { id: "profile", display_name: "Example", status: "suspended" },
      memberships: [{ organization_id: "organization", role: "athlete", status: "active" }],
    }), { status: 200 }));

    await expect(getSession()).resolves.toEqual({ status: "unauthenticated", reason: "revoked" });
  });

  it("does not release the dashboard when the API is unavailable", async () => {
    vi.mocked(getAccessToken).mockResolvedValue("unavailable-cookie");
    vi.mocked(fetch).mockRejectedValueOnce(new Error("connection refused"));

    await expect(getSession()).resolves.toEqual({ status: "error", reason: "api_unavailable" });
    await expect(requireSession()).rejects.toThrow("NEXT_REDIRECT:/login");
    expect(redirect).toHaveBeenCalledWith("/login");
  });

  it("clears expired and revoked cookies before redirecting", async () => {
    vi.mocked(getAccessToken).mockResolvedValue("expired-cookie");
    vi.mocked(fetch).mockResolvedValueOnce(new Response(null, { status: 401 }));

    await expect(requireSession()).rejects.toThrow("NEXT_REDIRECT:/login");
    expect(clearAuthCookies).toHaveBeenCalledOnce();
  });
});
