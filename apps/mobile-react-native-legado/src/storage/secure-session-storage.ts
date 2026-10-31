import * as SecureStore from "expo-secure-store";
import type { AuthSession, UserRole } from "@/types/auth";

const SESSION_STORAGE_KEY = "sysap_session_v1";

export interface ISecureSessionStorage {
  saveSession(session: AuthSession): Promise<void>;
  getSession(): Promise<AuthSession | null>;
  clearSession(): Promise<void>;
}

function isValidRole(role: unknown): role is UserRole {
  return role === "owner" || role === "athlete" || role === "trainer";
}

export function validateStoredSession(data: unknown): AuthSession | null {
  if (!data || typeof data !== "object") {
    return null;
  }

  const candidate = data as Record<string, unknown>;

  if (candidate.version !== 1) {
    return null;
  }

  if (
    typeof candidate.sessionId !== "string" ||
    !candidate.sessionId ||
    typeof candidate.profileId !== "string" ||
    !candidate.profileId ||
    typeof candidate.organizationId !== "string" ||
    !candidate.organizationId ||
    !isValidRole(candidate.role) ||
    typeof candidate.accessToken !== "string" ||
    !candidate.accessToken ||
    typeof candidate.refreshToken !== "string" ||
    !candidate.refreshToken ||
    typeof candidate.expiresAt !== "number" ||
    !Number.isFinite(candidate.expiresAt) ||
    candidate.expiresAt <= 0
  ) {
    return null;
  }

  return {
    version: 1,
    sessionId: candidate.sessionId,
    profileId: candidate.profileId,
    organizationId: candidate.organizationId,
    role: candidate.role,
    accessToken: candidate.accessToken,
    refreshToken: candidate.refreshToken,
    expiresAt: candidate.expiresAt,
  };
}

export class SecureSessionStorage implements ISecureSessionStorage {
  async saveSession(session: AuthSession): Promise<void> {
    try {
      const validated = validateStoredSession(session);
      if (!validated) {
        throw new Error("Invalid session structure before saving");
      }
      const serialized = JSON.stringify(validated);
      await SecureStore.setItemAsync(SESSION_STORAGE_KEY, serialized, {
        keychainAccessible: SecureStore.AFTER_FIRST_UNLOCK,
      });
    } catch (error) {
      // Re-throw so caller knows storage failed (cannot proceed unpersisted)
      throw new Error(
        `Failed to securely persist session: ${error instanceof Error ? error.message : "unknown"}`,
      );
    }
  }

  async getSession(): Promise<AuthSession | null> {
    try {
      const raw = await SecureStore.getItemAsync(SESSION_STORAGE_KEY);
      if (!raw) {
        return null;
      }

      let parsed: unknown;
      try {
        parsed = JSON.parse(raw);
      } catch {
        // Corrupted JSON -> clean up securely
        await this.clearSession();
        return null;
      }

      const session = validateStoredSession(parsed);
      if (!session) {
        // Malformed or incompatible session -> clean up securely
        await this.clearSession();
        return null;
      }

      return session;
    } catch {
      // Storage read error -> safely clear and return null
      await this.clearSession().catch(() => {});
      return null;
    }
  }

  async clearSession(): Promise<void> {
    try {
      await SecureStore.deleteItemAsync(SESSION_STORAGE_KEY);
    } catch {
      // Best-effort cleanup
    }
  }
}
