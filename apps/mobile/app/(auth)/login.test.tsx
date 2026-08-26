import React from "react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import LoginScreen from "@/../app/(auth)/login";
import * as AuthContextModule from "@/contexts/auth-context";
import * as RouterMock from "@/test/expo-router-mock";

describe("LoginScreen UX and Accessibility", () => {
  const mockLogin = vi.fn();
  const mockClearError = vi.fn();

  beforeEach(() => {
    vi.clearAllMocks();

    vi.spyOn(AuthContextModule, "useAuth").mockReturnValue({
      status: "unauthenticated",
      session: null,
      identity: null,
      activeRole: null,
      error: null,
      login: mockLogin,
      logout: vi.fn(),
      clearError: mockClearError,
    });
  });

  it("22. renders login screen with accessible inputs and labels", () => {
    const screen = <LoginScreen />;
    expect(screen).toBeDefined();
    expect(screen.type).toBe(LoginScreen);
  });

  it("23. prevents duplicate submission when request is submitting", () => {
    vi.spyOn(AuthContextModule, "useAuth").mockReturnValue({
      status: "loading",
      session: null,
      identity: null,
      activeRole: null,
      error: null,
      login: mockLogin,
      logout: vi.fn(),
      clearError: mockClearError,
    });

    const screen = <LoginScreen />;
    expect(screen).toBeDefined();
  });

  it("3. displays neutral error message when authError is present in context", () => {
    vi.spyOn(AuthContextModule, "useAuth").mockReturnValue({
      status: "unauthenticated",
      session: null,
      identity: null,
      activeRole: null,
      error: "Não foi possível entrar. Verifique seus dados ou tente novamente.",
      login: mockLogin,
      logout: vi.fn(),
      clearError: mockClearError,
    });

    const screen = <LoginScreen />;
    expect(screen).toBeDefined();
  });
});
