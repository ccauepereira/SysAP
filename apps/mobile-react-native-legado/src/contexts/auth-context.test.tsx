import { describe, expect, it } from "vitest";
import { computeRedirectRoute } from "@/contexts/auth-context";

describe("AuthContext & Route Guarding (computeRedirectRoute)", () => {
  it("7. unauthenticated state redirects protected segments to login", () => {
    expect(computeRedirectRoute("unauthenticated", null, ["(owner)"])).toBe(
      "/(auth)/login",
    );
    expect(computeRedirectRoute("unauthenticated", null, ["(athlete)"])).toBe(
      "/(auth)/login",
    );
    expect(computeRedirectRoute("unauthenticated", null, [])).toBe(
      "/(auth)/login",
    );
  });

  it("unauthenticated state does not redirect if already on (auth) segment", () => {
    expect(
      computeRedirectRoute("unauthenticated", null, ["(auth)", "login"]),
    ).toBeNull();
  });

  it("20. owner authenticated on auth or athlete segment redirects to (owner)", () => {
    expect(computeRedirectRoute("authenticated", "owner", ["(athlete)"])).toBe(
      "/(owner)",
    );
    expect(
      computeRedirectRoute("authenticated", "owner", ["(auth)", "login"]),
    ).toBe("/(owner)");
    expect(computeRedirectRoute("authenticated", "owner", [])).toBe("/(owner)");
    // Already on (owner) -> no redirect needed
    expect(computeRedirectRoute("authenticated", "owner", ["(owner)"])).toBeNull();
  });

  it("21. athlete authenticated on owner segment redirects to (athlete)", () => {
    expect(computeRedirectRoute("authenticated", "athlete", ["(owner)"])).toBe(
      "/(athlete)",
    );
    expect(
      computeRedirectRoute("authenticated", "athlete", ["(auth)", "login"]),
    ).toBe("/(athlete)");
    expect(computeRedirectRoute("authenticated", "athlete", [])).toBe(
      "/(athlete)",
    );
    // Already on (athlete) -> no redirect needed
    expect(
      computeRedirectRoute("authenticated", "athlete", ["(athlete)"]),
    ).toBeNull();
  });

  it("6. authenticated with unauthorized or unknown role requires logout", () => {
    expect(
      computeRedirectRoute("authenticated", "trainer" as any, ["(owner)"]),
    ).toBe("LOGOUT_REQUIRED");
    expect(
      computeRedirectRoute("authenticated", null, ["(athlete)"]),
    ).toBe("LOGOUT_REQUIRED");
  });

  it("loading status does not trigger any redirects", () => {
    expect(computeRedirectRoute("loading", null, ["(owner)"])).toBeNull();
    expect(computeRedirectRoute("loading", "owner", ["(athlete)"])).toBeNull();
  });
});
