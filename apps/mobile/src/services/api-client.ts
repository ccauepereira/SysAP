import { getApiBaseUrl, validateApiBaseUrl } from "@/config/env";
import type { CurrentIdentity, SafeAuthErrorCode, UserRole } from "@/types/auth";

export class ApiError extends Error {
  readonly code: SafeAuthErrorCode;
  readonly status?: number;

  constructor(code: SafeAuthErrorCode, message: string, status?: number) {
    super(message);
    this.name = "ApiError";
    this.code = code;
    this.status = status;
  }
}

export interface ApiLoginResult {
  sessionId: string;
  profileId: string;
  organizationId: string;
  role: UserRole;
  accessToken: string;
  refreshToken: string;
  expiresIn: number; // in seconds
}

export interface ApiRefreshResult {
  accessToken: string;
  refreshToken: string;
  tokenType: string;
  expiresIn: number;
}

export interface IApiClient {
  login(enrollmentNumber: string, password: string): Promise<ApiLoginResult>;
  refresh(refreshToken: string): Promise<ApiRefreshResult>;
  getCurrentIdentity(accessToken: string): Promise<CurrentIdentity>;
  logout(accessToken: string): Promise<void>;
}

const MAX_RESPONSE_BYTES = 64 * 1024; // 64 KB limit
const DEFAULT_TIMEOUT_MS = 15000;

export class ApiClient implements IApiClient {
  private readonly baseUrl: string;

  constructor(baseUrl?: string) {
    const raw = baseUrl || getApiBaseUrl();
    const isProd = process.env.NODE_ENV === "production";
    this.baseUrl = validateApiBaseUrl(raw, isProd);
  }

  getBaseUrl(): string {
    return this.baseUrl;
  }

  private isSameHost(targetUrl: string): boolean {
    try {
      const parsedTarget = new URL(targetUrl, this.baseUrl);
      const parsedBase = new URL(this.baseUrl);
      return (
        parsedTarget.protocol === parsedBase.protocol &&
        parsedTarget.host === parsedBase.host
      );
    } catch {
      return false;
    }
  }

  private async request<T>(
    endpoint: string,
    options: {
      method: "GET" | "POST";
      body?: unknown;
      token?: string;
      timeoutMs?: number;
    },
  ): Promise<T> {
    const fullUrl = `${this.baseUrl}${endpoint}`;

    if (!this.isSameHost(fullUrl)) {
      throw new ApiError(
        "generic_error",
        "Não foi possível conectar com o servidor.",
      );
    }

    const headers: Record<string, string> = {
      Accept: "application/json",
    };

    if (options.body) {
      headers["Content-Type"] = "application/json";
    }

    if (options.token) {
      if (!options.token.trim() || options.token.length > 16384) {
        throw new ApiError(
          "invalid_credentials",
          "Token de autenticação inválido.",
        );
      }
      headers["Authorization"] = `Bearer ${options.token}`;
    }

    const controller = new AbortController();
    const timeout = setTimeout(
      () => controller.abort(),
      options.timeoutMs ?? DEFAULT_TIMEOUT_MS,
    );

    try {
      const response = await fetch(fullUrl, {
        method: options.method,
        headers,
        body: options.body ? JSON.stringify(options.body) : undefined,
        redirect: "manual",
        signal: controller.signal,
      });

      clearTimeout(timeout);

      // Handle 204 No Content
      if (response.status === 204) {
        return undefined as unknown as T;
      }

      const text = await response.text();
      if (text.length > MAX_RESPONSE_BYTES) {
        throw new ApiError(
          "service_unavailable",
          "Resposta do servidor excedeu o tamanho permitido.",
          response.status,
        );
      }

      let data: unknown = null;
      if (text.length > 0) {
        try {
          data = JSON.parse(text);
        } catch {
          // Non-JSON response
        }
      }

      if (!response.ok) {
        this.handleErrorResponse(response.status, data);
      }

      return data as T;
    } catch (error) {
      clearTimeout(timeout);
      if (error instanceof ApiError) {
        throw error;
      }

      if (error instanceof Error && error.name === "AbortError") {
        throw new ApiError(
          "no_connection",
          "O servidor demorou para responder. Verifique sua conexão.",
        );
      }

      // Network error or fetch failure
      throw new ApiError(
        "no_connection",
        "Sem conexão com o servidor. Verifique sua internet.",
      );
    }
  }

  private handleErrorResponse(status: number, data: unknown): never {
    let errorCode = "generic_error";
    if (data && typeof data === "object" && "error" in data) {
      const errObj = (data as { error: { code?: string } }).error;
      if (errObj && typeof errObj.code === "string") {
        errorCode = errObj.code;
      }
    }

    switch (status) {
      case 401:
        if (errorCode === "authentication_required") {
          throw new ApiError(
            "session_expired",
            "Sua sessão expirou. Faça login novamente.",
            401,
          );
        }
        throw new ApiError(
          "invalid_credentials",
          "Não foi possível entrar. Verifique seus dados ou tente novamente.",
          401,
        );
      case 403:
        throw new ApiError(
          "access_denied",
          "Acesso não permitido.",
          403,
        );
      case 429:
        throw new ApiError(
          "temporary_failure",
          "Muitas tentativas. Aguarde alguns instantes e tente novamente.",
          429,
        );
      case 503:
        throw new ApiError(
          "service_unavailable",
          "Serviço temporariamente indisponível. Tente novamente mais tarde.",
          503,
        );
      default:
        throw new ApiError(
          "generic_error",
          "Ocorreu um erro ao processar sua solicitação.",
          status,
        );
    }
  }

  async login(
    enrollmentNumber: string,
    password: string,
  ): Promise<ApiLoginResult> {
    const raw = await this.request<{
      session_id: string;
      profile_id: string;
      organization_id: string;
      role: UserRole;
      access_token: string;
      refresh_token: string;
      expires_in: number;
    }>("/v1/auth/login", {
      method: "POST",
      body: {
        enrollment_number: enrollmentNumber,
        password,
      },
    });

    if (
      !raw ||
      !raw.session_id ||
      !raw.profile_id ||
      !raw.organization_id ||
      !raw.role ||
      !raw.access_token ||
      !raw.refresh_token ||
      typeof raw.expires_in !== "number"
    ) {
      throw new ApiError(
        "service_unavailable",
        "Resposta de login inválida do servidor.",
      );
    }

    return {
      sessionId: raw.session_id,
      profileId: raw.profile_id,
      organizationId: raw.organization_id,
      role: raw.role,
      accessToken: raw.access_token,
      refreshToken: raw.refresh_token,
      expiresIn: raw.expires_in,
    };
  }

  async refresh(refreshToken: string): Promise<ApiRefreshResult> {
    const raw = await this.request<{
      access_token: string;
      refresh_token: string;
      token_type: string;
      expires_in: number;
    }>("/v1/auth/refresh", {
      method: "POST",
      body: {
        refresh_token: refreshToken,
      },
    });

    if (
      !raw ||
      !raw.access_token ||
      !raw.refresh_token ||
      typeof raw.expires_in !== "number"
    ) {
      throw new ApiError(
        "session_expired",
        "Não foi possível renovar a sessão.",
      );
    }

    return {
      accessToken: raw.access_token,
      refreshToken: raw.refresh_token,
      tokenType: raw.token_type || "Bearer",
      expiresIn: raw.expires_in,
    };
  }

  async getCurrentIdentity(accessToken: string): Promise<CurrentIdentity> {
    const raw = await this.request<{
      profile: { id: string; display_name: string };
      memberships: Array<{
        organization_id: string;
        role: UserRole;
        status: "active" | "suspended";
      }>;
    }>("/v1/me", {
      method: "GET",
      token: accessToken,
    });

    if (
      !raw ||
      !raw.profile ||
      !raw.profile.id ||
      !raw.profile.display_name ||
      !Array.isArray(raw.memberships)
    ) {
      throw new ApiError(
        "service_unavailable",
        "Identidade inválida retornada pelo servidor.",
      );
    }

    return {
      profile: {
        id: raw.profile.id,
        displayName: raw.profile.display_name,
      },
      memberships: raw.memberships.map((m) => ({
        organizationId: m.organization_id,
        role: m.role,
        status: m.status,
      })),
    };
  }

  async logout(accessToken: string): Promise<void> {
    try {
      await this.request<void>("/v1/auth/logout", {
        method: "POST",
        token: accessToken,
      });
    } catch {
      // Remote logout errors (e.g. network offline, 401 already revoked)
      // are swallowed so local logout always completes.
    }
  }
}
