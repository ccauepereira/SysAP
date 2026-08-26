import { describe, expect, it } from "vitest";
import { validateApiBaseUrl } from "@/config/env";

describe("env configuration and base URL validation", () => {
  it("allows valid HTTPS URLs in production", () => {
    const url = validateApiBaseUrl("https://api.sysap.example.com", true);
    expect(url).toBe("https://api.sysap.example.com");
  });

  it("rejects HTTP URLs in production", () => {
    expect(() =>
      validateApiBaseUrl("http://api.sysap.example.com", true),
    ).toThrow("Production API base URL must use HTTPS");
  });

  it("allows localhost and private LAN HTTP in development", () => {
    expect(validateApiBaseUrl("http://127.0.0.1:8080", false)).toBe(
      "http://127.0.0.1:8080",
    );
    expect(validateApiBaseUrl("http://10.0.2.2:8080", false)).toBe(
      "http://10.0.2.2:8080",
    );
    expect(validateApiBaseUrl("http://192.168.1.50:8080", false)).toBe(
      "http://192.168.1.50:8080",
    );
  });

  it("rejects public non-private HTTP URLs in development", () => {
    expect(() =>
      validateApiBaseUrl("http://8.8.8.8:8080", false),
    ).toThrow(
      "Insecure HTTP is only permitted for localhost or private LAN development addresses",
    );
  });

  it("rejects malformed or empty URLs", () => {
    expect(() => validateApiBaseUrl("", false)).toThrow(
      "API base URL is required",
    );
    expect(() => validateApiBaseUrl("not-a-url", false)).toThrow(
      "API base URL is not a valid URL",
    );
  });
});
