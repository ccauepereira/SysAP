import React, {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useMemo,
  useState,
  type ReactNode,
} from "react";
import { useRouter, useSegments } from "expo-router";
import { ApiError } from "@/services/api-client";
import {
  AuthService,
  type IAuthService,
} from "@/services/auth-service";
import { ApiClient } from "@/services/api-client";
import { SecureSessionStorage } from "@/storage/secure-session-storage";
import type {
  AuthSession,
  AuthStatus,
  CurrentIdentity,
  UserRole,
} from "@/types/auth";

export interface AuthContextValue {
  status: AuthStatus;
  session: AuthSession | null;
  identity: CurrentIdentity | null;
  activeRole: UserRole | null;
  error: string | null;
  login(enrollmentNumber: string, password: string): Promise<void>;
  logout(): Promise<void>;
  clearError(): void;
}

const AuthContext = createContext<AuthContextValue | null>(null);

const defaultAuthService = new AuthService(
  new SecureSessionStorage(),
  new ApiClient(),
);

interface AuthProviderProps {
  children: ReactNode;
  authService?: IAuthService;
}

export function computeRedirectRoute(
  status: AuthStatus,
  activeRole: UserRole | null,
  segments: string[],
): string | null {
  if (status === "loading") return null;

  const inAuthGroup = segments[0] === "(auth)";
  const inOwnerGroup = segments[0] === "(owner)";
  const inAthleteGroup = segments[0] === "(athlete)";

  if (status === "unauthenticated") {
    if (!inAuthGroup) {
      return "/(auth)/login";
    }
  } else if (status === "authenticated") {
    if (activeRole === "owner") {
      if (inAuthGroup || inAthleteGroup || segments.length === 0) {
        return "/(owner)";
      }
    } else if (activeRole === "athlete") {
      if (inAuthGroup || inOwnerGroup || segments.length === 0) {
        return "/(athlete)";
      }
    } else {
      return "LOGOUT_REQUIRED";
    }
  }
  return null;
}

export function AuthProvider({
  children,
  authService = defaultAuthService,
}: AuthProviderProps) {
  const [status, setStatus] = useState<AuthStatus>("loading");
  const [session, setSession] = useState<AuthSession | null>(null);
  const [identity, setIdentity] = useState<CurrentIdentity | null>(null);
  const [error, setError] = useState<string | null>(null);

  const router = useRouter();
  const segments = useSegments();

  const activeRole: UserRole | null = useMemo(() => {
    if (!identity || !identity.memberships) return null;
    const active = identity.memberships.find((m) => m.status === "active");
    return active ? active.role : null;
  }, [identity]);

  const clearError = useCallback(() => {
    setError(null);
  }, []);

  const login = useCallback(
    async (enrollmentNumber: string, password: string) => {
      setError(null);
      try {
        const result = await authService.login(enrollmentNumber, password);
        setSession(result.session);
        setIdentity(result.identity);
        setStatus("authenticated");
      } catch (err) {
        setSession(null);
        setIdentity(null);
        setStatus("unauthenticated");

        if (err instanceof ApiError) {
          setError(err.message);
        } else {
          setError("Não foi possível entrar. Verifique seus dados ou tente novamente.");
        }
        throw err;
      }
    },
    [authService],
  );

  const logout = useCallback(async () => {
    try {
      await authService.logout();
    } finally {
      setSession(null);
      setIdentity(null);
      setError(null);
      setStatus("unauthenticated");
    }
  }, [authService]);

  // Initial session restoration
  useEffect(() => {
    let mounted = true;

    async function init() {
      try {
        const restored = await authService.restoreSession();
        if (mounted) {
          if (restored) {
            setSession(restored.session);
            setIdentity(restored.identity);
            setStatus("authenticated");
          } else {
            setSession(null);
            setIdentity(null);
            setStatus("unauthenticated");
          }
        }
      } catch {
        if (mounted) {
          setSession(null);
          setIdentity(null);
          setStatus("unauthenticated");
        }
      }
    }

    init();

    return () => {
      mounted = false;
    };
  }, [authService]);

  // Role-based route protection and redirection
  useEffect(() => {
    const target = computeRedirectRoute(status, activeRole, segments);
    if (!target) return;
    if (target === "LOGOUT_REQUIRED") {
      logout();
    } else {
      router.replace(target as any);
    }
  }, [status, activeRole, segments, router, logout]);

  const value = useMemo<AuthContextValue>(
    () => ({
      status,
      session,
      identity,
      activeRole,
      error,
      login,
      logout,
      clearError,
    }),
    [status, session, identity, activeRole, error, login, logout, clearError],
  );

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>;
}

export function useAuth(): AuthContextValue {
  const context = useContext(AuthContext);
  if (!context) {
    throw new Error("useAuth must be used within an AuthProvider");
  }
  return context;
}
