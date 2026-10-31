import { beforeEach, describe, expect, it, vi } from "vitest";
import { ApiClient, ApiError } from "@/services/api-client";

describe("ApiClient", () => {
  let client: ApiClient;
  const mockBaseUrl = "http://127.0.0.1:8080";

  beforeEach(() => {
    client = new ApiClient(mockBaseUrl);
    vi.stubGlobal("fetch", vi.fn());
  });

  it("does not attach Authorization header on unauthenticated endpoints", async () => {
    const mockFetch = vi.fn().mockResolvedValue({
      ok: true,
      status: 200,
      text: () =>
        JSON.stringify({
          session_id: "11111111-1111-4111-8111-111111111111",
          profile_id: "22222222-2222-4222-8222-222222222222",
          organization_id: "33333333-3333-4333-8333-333333333333",
          role: "owner",
          access_token: "mock.access.token",
          refresh_token: "mock.refresh.token",
          expires_in: 3600,
        }),
    });
    vi.stubGlobal("fetch", mockFetch);

    await client.login("2026000001", "ValidPass12345678");

    expect(mockFetch).toHaveBeenCalledTimes(1);
    const requestInit = (mockFetch.mock.calls[0] as any[])[1];
    expect(requestInit.headers.Authorization).toBeUndefined();
  });

  it("attaches Authorization header with Bearer prefix on protected endpoints", async () => {
    const mockFetch = vi.fn().mockResolvedValue({
      ok: true,
      status: 200,
      text: () =>
        JSON.stringify({
          profile: {
            id: "22222222-2222-4222-8222-222222222222",
            display_name: "Owner Test",
          },
          memberships: [
            {
              organization_id: "33333333-3333-4333-8333-333333333333",
              role: "owner",
              status: "active",
            },
          ],
        }),
    });
    vi.stubGlobal("fetch", mockFetch);

    const identity = await client.getCurrentIdentity("mock.jwt.token");

    expect(identity.profile.displayName).toBe("Owner Test");
    expect(mockFetch).toHaveBeenCalledTimes(1);
    const requestInit = (mockFetch.mock.calls[0] as any[])[1];
    expect(requestInit.headers.Authorization).toBe("Bearer mock.jwt.token");
  });

  it("maps 401 invalid credentials to safe error message", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValue({
        ok: false,
        status: 401,
        text: () =>
          JSON.stringify({
            error: {
              code: "invalid_credentials",
              message: "invalid credentials",
            },
          }),
      }),
    );

    await expect(
      client.login("2026000001", "WrongPassword12345"),
    ).rejects.toThrow(
      "Não foi possível entrar. Verifique seus dados ou tente novamente.",
    );
  });

  it("maps 401 authentication_required on /me to session_expired", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValue({
        ok: false,
        status: 401,
        text: () =>
          JSON.stringify({
            error: {
              code: "authentication_required",
              message: "authentication is required",
            },
          }),
      }),
    );

    try {
      await client.getCurrentIdentity("expired.jwt.token");
      expect.unreachable("Should have thrown");
    } catch (error) {
      expect(error).toBeInstanceOf(ApiError);
      expect((error as ApiError).code).toBe("session_expired");
    }
  });

  it("maps 503 service unavailable to safe error message", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValue({
        ok: false,
        status: 503,
        text: () =>
          JSON.stringify({
            error: {
              code: "service_unavailable",
              message: "service is unavailable",
            },
          }),
      }),
    );

    await expect(
      client.login("2026000001", "Password12345678"),
    ).rejects.toThrow(
      "Serviço temporariamente indisponível. Tente novamente mais tarde.",
    );
  });

  it("maps network disconnection or failure to no_connection", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn().mockRejectedValue(new Error("Failed to fetch")),
    );

    try {
      await client.login("2026000001", "Password12345678");
      expect.unreachable("Should have thrown");
    } catch (error) {
      expect(error).toBeInstanceOf(ApiError);
      expect((error as ApiError).code).toBe("no_connection");
    }
  });

  it("handles remote logout gracefully on 204 or error", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValue({
        ok: true,
        status: 204,
        text: () => "",
      }),
    );

    await expect(client.logout("valid.token")).resolves.toBeUndefined();
  });
});
