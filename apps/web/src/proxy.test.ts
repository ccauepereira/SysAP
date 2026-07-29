import { describe, expect, it } from "vitest";
import { NextRequest } from "next/server";
import { proxy } from "./proxy";

function request(path: string, cookie?: string) {
  return new NextRequest(
    `http://localhost:3000${path}`,
    cookie ? { headers: { cookie } } : {},
  );
}

describe("optimistic protected-route proxy", () => {
  it.each(["/", "/admin/atletas/novo", "/atleta", "/atletas", "/treinos/hoje", "/alertas"]) (
    "redirects %s without an access cookie",
    (path) => {
      const response = proxy(request(path));
      expect(response.status).toBe(307);
      expect(response.headers.get("location")).toBe("http://localhost:3000/login");
    },
  );

  it("allows the server-side session guard to validate a present cookie", () => {
    const response = proxy(request("/", "sysap-access=header.payload.signature"));
    expect(response.status).toBe(200);
  });

  it("redirects a clearly malformed access cookie", () => {
    const response = proxy(request("/", "sysap-access=malformed-cookie"));
    expect(response.status).toBe(307);
    expect(response.headers.get("location")).toBe("http://localhost:3000/login");
  });

  it("does not intercept public authentication routes", () => {
    const response = proxy(request("/login"));
    expect(response.status).toBe(200);
  });
});
