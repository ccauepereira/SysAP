import { ApiError, type IApiClient } from "@/services/api-client";
import type { ISecureSessionStorage } from "@/storage/secure-session-storage";
import type { AuthSession, CurrentIdentity, UserRole } from "@/types/auth";

export interface IAuthService {
  login(
    enrollmentNumber: string,
    password: string,
  ): Promise<{ session: AuthSession; identity: CurrentIdentity }>;
  restoreSession(): Promise<{
    session: AuthSession;
    identity: CurrentIdentity;
  } | null>;
  refreshSession(): Promise<AuthSession | null>;
  logout(): Promise<void>;
  getCurrentSession(): Promise<AuthSession | null>;
}

const EXPIRY_BUFFER_MS = 60 * 1000; // 60 seconds buffer before token expiry

export class AuthService implements IAuthService {
  private inFlightRefresh: Promise<AuthSession | null> | null = null;

  constructor(
    private readonly storage: ISecureSessionStorage,
    private readonly apiClient: IApiClient,
    private readonly now: () => number = () => Date.now(),
  ) {}

  async getCurrentSession(): Promise<AuthSession | null> {
    return this.storage.getSession();
  }

  async login(
    enrollmentNumber: string,
    password: string,
  ): Promise<{ session: AuthSession; identity: CurrentIdentity }> {
    // 1. Authenticate with credentials via API
    const loginResult = await this.apiClient.login(enrollmentNumber, password);

    const expiresAt = this.now() + loginResult.expiresIn * 1000;
    const session: AuthSession = {
      version: 1,
      sessionId: loginResult.sessionId,
      profileId: loginResult.profileId,
      organizationId: loginResult.organizationId,
      role: loginResult.role,
      accessToken: loginResult.accessToken,
      refreshToken: loginResult.refreshToken,
      expiresAt,
    };

    // 2. Persist session securely in SecureStore
    // If persistence fails, this throws and aborts login
    await this.storage.saveSession(session);

    // 3. Confirm profile, memberships, and role authoritatively with the API
    try {
      const identity = await this.apiClient.getCurrentIdentity(
        session.accessToken,
      );
      this.validateIdentityAndRole(identity);
      return { session, identity };
    } catch (error) {
      // If server rejects identity (e.g. suspended, inactive, unauthorized role),
      // wipe session from storage immediately
      await this.storage.clearSession();
      throw error;
    }
  }

  async restoreSession(): Promise<{
    session: AuthSession;
    identity: CurrentIdentity;
  } | null> {
    const session = await this.storage.getSession();
    if (!session) {
      return null;
    }

    let activeSession = session;

    // Check if access token is expired or close to expiring
    const isExpiringSoon = this.now() >= activeSession.expiresAt - EXPIRY_BUFFER_MS;
    if (isExpiringSoon) {
      const refreshed = await this.refreshSession();
      if (!refreshed) {
        // Refresh failed -> session cleared in refreshSession()
        return null;
      }
      activeSession = refreshed;
    }

    // Authoritatively query server for current identity and active membership
    try {
      const identity = await this.apiClient.getCurrentIdentity(
        activeSession.accessToken,
      );
      this.validateIdentityAndRole(identity);
      return { session: activeSession, identity };
    } catch {
      // If server returns 401 (revoked), suspended, or network down during restore
      await this.storage.clearSession();
      return null;
    }
  }

  async refreshSession(): Promise<AuthSession | null> {
    // Under concurrency, reuse the existing in-flight refresh promise
    if (this.inFlightRefresh) {
      return this.inFlightRefresh;
    }

    this.inFlightRefresh = (async () => {
      try {
        const current = await this.storage.getSession();
        if (!current || !current.refreshToken) {
          await this.storage.clearSession();
          return null;
        }

        const refreshResult = await this.apiClient.refresh(current.refreshToken);
        const updatedSession: AuthSession = {
          ...current,
          accessToken: refreshResult.accessToken,
          refreshToken: refreshResult.refreshToken,
          expiresAt: this.now() + refreshResult.expiresIn * 1000,
        };

        await this.storage.saveSession(updatedSession);
        return updatedSession;
      } catch {
        // Refresh failed -> wipe local session to prevent infinite retry loops
        await this.storage.clearSession();
        return null;
      } finally {
        this.inFlightRefresh = null;
      }
    })();

    return this.inFlightRefresh;
  }

  async logout(): Promise<void> {
    try {
      const current = await this.storage.getSession();
      if (current && current.accessToken) {
        await this.apiClient.logout(current.accessToken);
      }
    } catch {
      // Ignore network errors during remote logout
    } finally {
      // Local session must ALWAYS be cleared
      await this.storage.clearSession();
    }
  }

  private validateIdentityAndRole(identity: CurrentIdentity): void {
    if (!identity.profile || !identity.profile.id) {
      throw new ApiError(
        "service_unavailable",
        "Perfil não encontrado na resposta da API.",
      );
    }

    if (!Array.isArray(identity.memberships) || identity.memberships.length === 0) {
      throw new ApiError(
        "membership_inactive",
        "Nenhum vínculo ativo encontrado para este usuário.",
      );
    }

    const activeMembership = identity.memberships.find(
      (m) => m.status === "active",
    );

    if (!activeMembership) {
      const suspendedMembership = identity.memberships.find(
        (m) => m.status === "suspended",
      );
      if (suspendedMembership) {
        throw new ApiError(
          "account_suspended",
          "Seu acesso está temporariamente suspenso. Procure o responsável.",
        );
      }
      throw new ApiError(
        "membership_inactive",
        "Seu vínculo com a organização não está ativo.",
      );
    }

    const role: UserRole = activeMembership.role;
    if (role !== "owner" && role !== "athlete") {
      // Trainer or unknown role without mobile screen in this phase
      throw new ApiError(
        "unauthorized_role",
        "Acesso disponível apenas para atletas e administradores neste aplicativo.",
      );
    }
  }
}
