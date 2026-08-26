import { beforeEach, describe, expect, it, vi } from "vitest";
import { AuthService } from "@/services/auth-service";
import { ApiError, type IApiClient } from "@/services/api-client";
import type { ISecureSessionStorage } from "@/storage/secure-session-storage";
import type { AuthSession, CurrentIdentity } from "@/types/auth";

describe("AuthService", () => {
  let authService: AuthService;
  let mockStorage: ISecureSessionStorage;
  let mockApiClient: IApiClient;
  let storedSession: AuthSession | null = null;
  let currentTime: number;

  const mockOwnerLogin = {
    sessionId: "11111111-1111-4111-8111-111111111111",
    profileId: "22222222-2222-4222-8222-222222222222",
    organizationId: "33333333-3333-4333-8333-333333333333",
    role: "owner" as const,
    accessToken: "mock.owner.jwt",
    refreshToken: "mock.owner.refresh",
    expiresIn: 3600,
  };

  const mockOwnerIdentity: CurrentIdentity = {
    profile: {
      id: "22222222-2222-4222-8222-222222222222",
      displayName: "Artur Owner",
    },
    memberships: [
      {
        organizationId: "33333333-3333-4333-8333-333333333333",
        role: "owner",
        status: "active",
      },
    ],
  };

  const mockAthleteIdentity: CurrentIdentity = {
    profile: {
      id: "44444444-4444-4444-8444-444444444444",
      displayName: "Atleta Exemplo",
    },
    memberships: [
      {
        organizationId: "33333333-3333-4333-8333-333333333333",
        role: "athlete",
        status: "active",
      },
    ],
  };

  beforeEach(() => {
    storedSession = null;
    currentTime = 1700000000000;

    mockStorage = {
      saveSession: vi.fn(async (session: AuthSession) => {
        storedSession = session;
      }),
      getSession: vi.fn(async () => storedSession),
      clearSession: vi.fn(async () => {
        storedSession = null;
      }),
    };

    mockApiClient = {
      login: vi.fn(),
      refresh: vi.fn(),
      getCurrentIdentity: vi.fn(),
      logout: vi.fn(),
    };

    authService = new AuthService(
      mockStorage,
      mockApiClient,
      () => currentTime,
    );
  });

  it("1. completes successful login with owner role and confirms identity from API", async () => {
    vi.mocked(mockApiClient.login).mockResolvedValueOnce(mockOwnerLogin);
    vi.mocked(mockApiClient.getCurrentIdentity).mockResolvedValueOnce(
      mockOwnerIdentity,
    );

    const result = await authService.login("2026000001", "ValidOwnerPassword123");

    expect(result.session.role).toBe("owner");
    expect(result.identity.profile.displayName).toBe("Artur Owner");
    expect(mockStorage.saveSession).toHaveBeenCalledTimes(1);
    expect(mockApiClient.getCurrentIdentity).toHaveBeenCalledWith(
      "mock.owner.jwt",
    );
    expect(storedSession?.role).toBe("owner");
  });

  it("2. completes successful login with athlete role and confirms identity from API", async () => {
    const athleteLogin = {
      ...mockOwnerLogin,
      role: "athlete" as const,
      accessToken: "mock.athlete.jwt",
    };
    vi.mocked(mockApiClient.login).mockResolvedValueOnce(athleteLogin);
    vi.mocked(mockApiClient.getCurrentIdentity).mockResolvedValueOnce(
      mockAthleteIdentity,
    );

    const result = await authService.login(
      "2026000002",
      "ValidAthletePassword123",
    );

    expect(result.session.role).toBe("athlete");
    expect(result.identity.profile.displayName).toBe("Atleta Exemplo");
    expect(mockStorage.saveSession).toHaveBeenCalledTimes(1);
  });

  it("3. invalid credentials reject login and do not save session", async () => {
    vi.mocked(mockApiClient.login).mockRejectedValueOnce(
      new ApiError(
        "invalid_credentials",
        "Não foi possível entrar. Verifique seus dados ou tente novamente.",
        401,
      ),
    );

    await expect(
      authService.login("2026000001", "WrongPassword123"),
    ).rejects.toThrow(
      "Não foi possível entrar. Verifique seus dados ou tente novamente.",
    );

    expect(mockStorage.saveSession).not.toHaveBeenCalled();
    expect(storedSession).toBeNull();
  });

  it("4. suspended profile during identity confirmation clears session and returns secure error", async () => {
    vi.mocked(mockApiClient.login).mockResolvedValueOnce(mockOwnerLogin);
    vi.mocked(mockApiClient.getCurrentIdentity).mockResolvedValueOnce({
      profile: { id: "22222222-2222-4222-8222-222222222222", displayName: "Suspended" },
      memberships: [
        {
          organizationId: "33333333-3333-4333-8333-333333333333",
          role: "athlete",
          status: "suspended",
        },
      ],
    });

    await expect(
      authService.login("2026000001", "ValidPassword12345"),
    ).rejects.toThrow("Seu acesso está temporariamente suspenso");

    expect(mockStorage.clearSession).toHaveBeenCalled();
    expect(storedSession).toBeNull();
  });

  it("5. inactive membership during identity confirmation clears session", async () => {
    vi.mocked(mockApiClient.login).mockResolvedValueOnce(mockOwnerLogin);
    vi.mocked(mockApiClient.getCurrentIdentity).mockResolvedValueOnce({
      profile: { id: "22222222-2222-4222-8222-222222222222", displayName: "No Memberships" },
      memberships: [],
    });

    await expect(
      authService.login("2026000001", "ValidPassword12345"),
    ).rejects.toThrow("Nenhum vínculo ativo encontrado");

    expect(mockStorage.clearSession).toHaveBeenCalled();
    expect(storedSession).toBeNull();
  });

  it("6. unknown role without mobile screen does not receive access and clears session", async () => {
    vi.mocked(mockApiClient.login).mockResolvedValueOnce(mockOwnerLogin);
    vi.mocked(mockApiClient.getCurrentIdentity).mockResolvedValueOnce({
      profile: { id: "22222222-2222-4222-8222-222222222222", displayName: "Trainer Role" },
      memberships: [
        {
          organizationId: "33333333-3333-4333-8333-333333333333",
          role: "trainer",
          status: "active",
        },
      ],
    });

    await expect(
      authService.login("2026000001", "ValidPassword12345"),
    ).rejects.toThrow("Acesso disponível apenas para atletas e administradores");

    expect(mockStorage.clearSession).toHaveBeenCalled();
    expect(storedSession).toBeNull();
  });

  it("7. missing session during restoreSession returns null", async () => {
    const restored = await authService.restoreSession();
    expect(restored).toBeNull();
  });

  it("10. secure storage failure aborts login", async () => {
    vi.mocked(mockApiClient.login).mockResolvedValueOnce(mockOwnerLogin);
    vi.mocked(mockStorage.saveSession).mockRejectedValueOnce(
      new Error("Storage disk full"),
    );

    await expect(
      authService.login("2026000001", "ValidPassword12345"),
    ).rejects.toThrow("Storage disk full");

    expect(mockApiClient.getCurrentIdentity).not.toHaveBeenCalled();
  });

  it("16. concurrent token refresh calls share a single in-flight request", async () => {
    storedSession = {
      version: 1,
      sessionId: "11111111-1111-4111-8111-111111111111",
      profileId: "22222222-2222-4222-8222-222222222222",
      organizationId: "33333333-3333-4333-8333-333333333333",
      role: "owner",
      accessToken: "old.token",
      refreshToken: "old.refresh",
      expiresAt: currentTime - 1000, // Expired
    };

    vi.mocked(mockApiClient.refresh).mockImplementation(async () => {
      // Simulate small latency
      return {
        accessToken: "new.rotated.jwt",
        refreshToken: "new.rotated.refresh",
        tokenType: "Bearer",
        expiresIn: 3600,
      };
    });

    const [refresh1, refresh2, refresh3] = await Promise.all([
      authService.refreshSession(),
      authService.refreshSession(),
      authService.refreshSession(),
    ]);

    expect(mockApiClient.refresh).toHaveBeenCalledTimes(1);
    expect(refresh1?.accessToken).toBe("new.rotated.jwt");
    expect(refresh2?.accessToken).toBe("new.rotated.jwt");
    expect(refresh3?.accessToken).toBe("new.rotated.jwt");
  });

  it("17. failed refresh clears session safely without creating infinite retry loop", async () => {
    storedSession = {
      version: 1,
      sessionId: "11111111-1111-4111-8111-111111111111",
      profileId: "22222222-2222-4222-8222-222222222222",
      organizationId: "33333333-3333-4333-8333-333333333333",
      role: "owner",
      accessToken: "old.token",
      refreshToken: "invalid.refresh",
      expiresAt: currentTime - 1000,
    };

    vi.mocked(mockApiClient.refresh).mockRejectedValueOnce(
      new ApiError("session_expired", "Refresh token revoked", 401),
    );

    const refreshed = await authService.refreshSession();
    expect(refreshed).toBeNull();
    expect(mockStorage.clearSession).toHaveBeenCalled();
    expect(storedSession).toBeNull();
  });

  it("18. logout clears local session even if remote network call throws", async () => {
    storedSession = {
      version: 1,
      sessionId: "11111111-1111-4111-8111-111111111111",
      profileId: "22222222-2222-4222-8222-222222222222",
      organizationId: "33333333-3333-4333-8333-333333333333",
      role: "owner",
      accessToken: "valid.jwt",
      refreshToken: "valid.refresh",
      expiresAt: currentTime + 3600 * 1000,
    };

    vi.mocked(mockApiClient.logout).mockRejectedValueOnce(
      new Error("Network connection dropped"),
    );

    await authService.logout();

    expect(mockApiClient.logout).toHaveBeenCalledWith("valid.jwt");
    expect(mockStorage.clearSession).toHaveBeenCalled();
    expect(storedSession).toBeNull();
  });
});
