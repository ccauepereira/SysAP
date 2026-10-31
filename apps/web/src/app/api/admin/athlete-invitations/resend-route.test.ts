import { beforeEach, describe, expect, it, vi } from "vitest";
import { NextRequest } from "next/server";
import { POST as CreatePOST } from "./route";
import { POST as ManualActivationPOST } from "./[invitationId]/manual-activation/route";
import { POST } from "./[invitationId]/resend/route";

vi.mock("@/lib/auth/cookies", () => ({ getAccessToken: vi.fn() }));
vi.mock("@/lib/auth/session", () => ({ getSession: vi.fn() }));
vi.mock("@/lib/auth/csrf", () => ({ validateOrigin: vi.fn(() => true) }));

import { getAccessToken } from "@/lib/auth/cookies";
import { getSession } from "@/lib/auth/session";

describe("POST /api/admin/athlete-invitations", () => {
  beforeEach(() => {
    vi.resetAllMocks();
    vi.mocked(getAccessToken).mockResolvedValue("opaque-server-token");
    vi.mocked(getSession).mockResolvedValue({
      status: "authenticated",
      user: { role: "owner", organizationId: "org-id" },
    } as never);
  });

  it("normalizes Brazilian phone numbers and forwards enrollment_number from the backend", async () => {
    const fetchMock = vi.fn().mockResolvedValue(new Response(JSON.stringify({
      invitation_id: "11111111-1111-4111-8111-111111111111",
      profile_id: "22222222-2222-4222-8222-222222222222",
      enrollment_number: "2026000001",
      status: "pending_activation",
      expires_at: "2026-10-20T00:00:00Z",
      delivery_channel: "manual",
      delivery_status: "not_applicable",
    }), { status: 201 }));
    vi.stubGlobal("fetch", fetchMock);

    const request = new NextRequest("http://localhost/api/admin/athlete-invitations", {
      method: "POST",
      headers: {
        origin: "http://localhost",
        "content-type": "application/json",
        "Idempotency-Key": "33333333-3333-4333-8333-333333333333",
      },
      body: JSON.stringify({
        full_name: "Atleta Teste",
        phone_e164: "(85) 99999-0111",
        email: "athlete@example.test",
        birth_date: "2010-01-15",
        locality: "Fortaleza",
        team_id: "44444444-4444-4444-8444-444444444444",
        football_position: "midfielder",
        enrollment_delivery_channel: "manual",
      }),
    });
    const response = await CreatePOST(request);

    expect(response.status).toBe(201);
    expect(await response.json()).toEqual({
      invitation_id: "11111111-1111-4111-8111-111111111111",
      profile_id: "22222222-2222-4222-8222-222222222222",
      enrollment_number: "2026000001",
      status: "pending_activation",
      expires_at: "2026-10-20T00:00:00Z",
      delivery_channel: "manual",
      delivery_status: "not_applicable",
    });
    expect(JSON.parse(String(fetchMock.mock.calls[0]?.[1]?.body))).toMatchObject({
      phone_e164: "+5585999990111",
      enrollment_delivery_channel: "manual",
    });
  });
});

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

describe("POST /api/admin/athlete-invitations/:id/manual-activation", () => {
  beforeEach(() => {
    vi.resetAllMocks();
    vi.mocked(getAccessToken).mockResolvedValue("opaque-server-token");
    vi.mocked(getSession).mockResolvedValue({
      status: "authenticated",
      user: { role: "owner", organizationId: "org-id" },
    } as never);
  });

  it("returns 201 with enrollment_number, 6-digit code, and expires_at when authorized", async () => {
    vi.stubGlobal("fetch", vi.fn().mockResolvedValue(new Response(JSON.stringify({
      enrollment_number: "2026000001",
      code: "123456",
      expires_at: "2026-10-31T19:00:00Z",
    }), { status: 201 })));

    const request = new NextRequest("http://localhost/api/admin/athlete-invitations/11111111-1111-4111-8111-111111111111/manual-activation", {
      method: "POST",
      headers: { origin: "http://localhost" },
    });
    const response = await ManualActivationPOST(request, { params: Promise.resolve({ invitationId: "11111111-1111-4111-8111-111111111111" }) });

    expect(response.status).toBe(201);
    expect(response.headers.get("Cache-Control")).toBe("no-store");
    expect(await response.json()).toEqual({
      enrollment_number: "2026000001",
      code: "123456",
      expires_at: "2026-10-31T19:00:00Z",
    });
  });

  it("returns 401 authentication_required instead of 403 when the admin session was ended or expired", async () => {
    vi.mocked(getAccessToken).mockResolvedValue(undefined);
    vi.mocked(getSession).mockResolvedValue({
      status: "unauthenticated",
      reason: "no_session",
    } as never);

    const request = new NextRequest("http://localhost/api/admin/athlete-invitations/11111111-1111-4111-8111-111111111111/manual-activation", {
      method: "POST",
      headers: { origin: "http://localhost" },
    });
    const response = await ManualActivationPOST(request, { params: Promise.resolve({ invitationId: "11111111-1111-4111-8111-111111111111" }) });

    expect(response.status).toBe(401);
    expect(await response.json()).toEqual({ error: "authentication_required" });
  });

  it("forwards mfa_required on 403 when the backend rejects a session without AAL2", async () => {
    vi.stubGlobal("fetch", vi.fn().mockResolvedValue(new Response(JSON.stringify({
      error: { code: "mfa_required", message: "administrative mfa required" },
    }), { status: 403 })));

    const request = new NextRequest("http://localhost/api/admin/athlete-invitations/11111111-1111-4111-8111-111111111111/manual-activation", {
      method: "POST",
      headers: { origin: "http://localhost" },
    });
    const response = await ManualActivationPOST(request, { params: Promise.resolve({ invitationId: "11111111-1111-4111-8111-111111111111" }) });

    expect(response.status).toBe(403);
    expect(await response.json()).toEqual({ error: "mfa_required" });
  });
});
