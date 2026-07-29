import { describe, expect, it, vi } from "vitest";
import HomePage from "@/app/page";
import { getSystemStatus } from "@/lib/api/system-status";
import { requireSession } from "@/lib/auth/session";

vi.mock("@/lib/auth/session", () => ({
  requireSession: vi.fn(),
}));

vi.mock("@/lib/api/system-status", () => ({
  getSystemStatus: vi.fn(),
}));

const requireSessionMock = vi.mocked(requireSession);
const getSystemStatusMock = vi.mocked(getSystemStatus);

describe("protected web entry point", () => {
  it("redirects before querying dashboard data when there is no session", async () => {
    requireSessionMock.mockRejectedValueOnce(new Error("NEXT_REDIRECT"));

    await expect(HomePage()).rejects.toThrow("NEXT_REDIRECT");
    expect(getSystemStatusMock).not.toHaveBeenCalled();
  });

  it("checks the session before rendering the dashboard", async () => {
    requireSessionMock.mockResolvedValueOnce({
      id: "profile",
      name: "Example",
      role: "athlete",
      organizationId: "organization",
    });
    getSystemStatusMock.mockResolvedValueOnce({
      kind: "ready",
      label: "API online",
      description: "Serviços disponíveis.",
    });

    const page = await HomePage();
    expect(page).toBeTruthy();
    expect(requireSessionMock.mock.invocationCallOrder[0]).toBeLessThan(
      getSystemStatusMock.mock.invocationCallOrder[0] ?? Number.POSITIVE_INFINITY,
    );
  });
});
