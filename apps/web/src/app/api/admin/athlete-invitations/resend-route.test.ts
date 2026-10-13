import { beforeEach, describe, expect, it, vi } from "vitest";
import { NextRequest } from "next/server";
import { POST } from "./[invitationId]/resend/route";

vi.mock("@/lib/auth/cookies", () => ({ getAccessToken: vi.fn() }));
vi.mock("@/lib/auth/session", () => ({ getSession: vi.fn() }));
vi.mock("@/lib/auth/csrf", () => ({ validateOrigin: vi.fn(() => true) }));

import { getAccessToken } from "@/lib/auth/cookies";
import { getSession } from "@/lib/auth/session";

describe("POST /api/admin/athlete-invitations/:id/resend", () => {
  beforeEach(() => {
    vi.resetAllMocks();
    vi.mocked(getAccessToken).mockResolvedValue("opaque-server-token");
    vi.mocked(getSession).mockResolvedValue({
      status: "authenticated",
      user: { role: "owner", organizationId: "org-id" },
    } as never);
  });

  it("preserves every required field from the invitation response contract", async () => {
    vi.stubGlobal("fetch", vi.fn().mockResolvedValue(new Response(JSON.stringify({
      invitation_id: "11111111-1111-4111-8111-111111111111",
      profile_id: "22222222-2222-4222-8222-222222222222",
      status: "pending_activation",
      expires_at: "2026-10-20T00:00:00Z",
      delivery_channel: "email",
      delivery_status: "sent",
    }), { status: 202 })));

    const request = new NextRequest("http://localhost/api/admin/athlete-invitations/11111111-1111-4111-8111-111111111111/resend", {
      method: "POST",
      headers: { origin: "http://localhost", "Idempotency-Key": "33333333-3333-4333-8333-333333333333" },
    });
    const response = await POST(request, { params: Promise.resolve({ invitationId: "11111111-1111-4111-8111-111111111111" }) });

    expect(response.status).toBe(202);
    expect(await response.json()).toEqual({
      invitation_id: "11111111-1111-4111-8111-111111111111",
      profile_id: "22222222-2222-4222-8222-222222222222",
      status: "pending_activation",
      expires_at: "2026-10-20T00:00:00Z",
      delivery_channel: "email",
      delivery_status: "sent",
    });
  });
});
