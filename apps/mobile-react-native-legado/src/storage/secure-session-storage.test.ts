import { beforeEach, describe, expect, it, vi } from "vitest";
import {
  SecureSessionStorage,
  validateStoredSession,
} from "@/storage/secure-session-storage";
import type { AuthSession } from "@/types/auth";
import * as SecureStoreMock from "@/test/secure-store-mock";

describe("SecureSessionStorage", () => {
  let storage: SecureSessionStorage;

  const validSession: AuthSession = {
    version: 1,
    sessionId: "11111111-1111-4111-8111-111111111111",
    profileId: "22222222-2222-4222-8222-222222222222",
    organizationId: "33333333-3333-4333-8333-333333333333",
    role: "owner",
    accessToken: "mock.access.token",
    refreshToken: "mock.refresh.token",
    expiresAt: Date.now() + 3600 * 1000,
  };

  beforeEach(() => {
    SecureStoreMock.__clearStore();
    storage = new SecureSessionStorage();
  });

  it("persists and restores a valid session", async () => {
    await storage.saveSession(validSession);
    const restored = await storage.getSession();

    expect(restored).toEqual(validSession);
    expect(restored?.version).toBe(1);
    expect(restored?.role).toBe("owner");
    // Ensure password is never stored or present in session
    expect((restored as any).password).toBeUndefined();
    expect((restored as any).enrollment_number).toBeUndefined();
  });

  it("safely clears and returns null when stored JSON is corrupted", async () => {
    await SecureStoreMock.setItemAsync("sysap_session_v1", "{ malformed json");
    const restored = await storage.getSession();

    expect(restored).toBeNull();
    const afterClear = await SecureStoreMock.getItemAsync("sysap_session_v1");
    expect(afterClear).toBeNull();
  });

  it("safely clears and returns null when stored version or schema is invalid", async () => {
    const invalidVersion = { ...validSession, version: 2 };
    await SecureStoreMock.setItemAsync(
      "sysap_session_v1",
      JSON.stringify(invalidVersion),
    );

    const restored = await storage.getSession();
    expect(restored).toBeNull();
  });

  it("rejects saving an invalid session structure", async () => {
    const invalid = { ...validSession, role: "invalid_role" as any };
    await expect(storage.saveSession(invalid)).rejects.toThrow(
      "Failed to securely persist session",
    );
  });

  it("throws error when underlying SecureStore write fails", async () => {
    vi.spyOn(SecureStoreMock, "setItemAsync").mockRejectedValueOnce(
      new Error("Keystore unavailable"),
    );

    await expect(storage.saveSession(validSession)).rejects.toThrow(
      "Failed to securely persist session: Keystore unavailable",
    );
  });

  it("clears session completely on clearSession()", async () => {
    await storage.saveSession(validSession);
    expect(await storage.getSession()).not.toBeNull();

    await storage.clearSession();
    expect(await storage.getSession()).toBeNull();
  });

  it("validates session object fields strictly in validateStoredSession", () => {
    expect(validateStoredSession(null)).toBeNull();
    expect(validateStoredSession({})).toBeNull();
    expect(validateStoredSession({ ...validSession, expiresAt: -100 })).toBeNull();
    expect(validateStoredSession({ ...validSession, accessToken: "" })).toBeNull();
    expect(validateStoredSession({ ...validSession, sessionId: "" })).toBeNull();
  });
});
