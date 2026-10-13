import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { AthleteInvitationForm } from "./athlete-invitation-form";

const teamId = "10000000-0000-4000-8000-000000000001";
const teams = [{ id: teamId, name: "Turma fictícia" }];
const invitation = {
  invitation_id: "20000000-0000-4000-8000-000000000001",
  profile_id: "30000000-0000-4000-8000-000000000001",
  status: "pending_activation",
  expires_at: "2026-10-20T00:00:00.000Z",
  delivery_channel: "email",
  delivery_status: "failed",
};

afterEach(() => {
  vi.restoreAllMocks();
  vi.unstubAllGlobals();
});

describe("AthleteInvitationForm delivery recovery", () => {
  it("keeps the account pending and offers only an explicit resend after a definitive failure", async () => {
    vi.stubGlobal("crypto", { randomUUID: vi.fn(() => "40000000-0000-4000-8000-000000000001") });
    const fetchMock = vi.spyOn(global, "fetch")
      .mockResolvedValueOnce(new Response(JSON.stringify(invitation), { status: 201 }))
      .mockResolvedValueOnce(new Response(JSON.stringify(invitation), { status: 202 }));
    render(<AthleteInvitationForm teams={teams} />);
    fireEvent.change(screen.getByLabelText("Nome completo"), { target: { value: "Atleta fictício" } });
    fireEvent.change(screen.getByLabelText("Celular"), { target: { value: "+15555550123" } });
    fireEvent.change(screen.getByRole("textbox", { name: "E-mail" }), { target: { value: "athlete@example.test" } });
    fireEvent.change(screen.getByLabelText("Data de nascimento"), { target: { value: "2010-01-15" } });
    fireEvent.change(screen.getByLabelText("Bairro/cidade"), { target: { value: "Cidade fictícia" } });
    fireEvent.change(screen.getByLabelText("Turma"), { target: { value: teamId } });
    fireEvent.change(screen.getByLabelText("Posição"), { target: { value: "midfielder" } });
    fireEvent.submit(screen.getByRole("button", { name: /Criar convite/ }).closest("form")!);

    expect(await screen.findByText("Ativação: pendente")).toBeVisible();
    expect(screen.getByText("Entrega: falha confirmada")).toBeVisible();
    fireEvent.click(screen.getByRole("button", { name: "Reenviar matrícula" }));
    await waitFor(() => expect(fetchMock).toHaveBeenCalledTimes(2));
    const resendRequest = fetchMock.mock.calls.at(1);
    expect(resendRequest?.[0]).toContain("/resend");
    expect(resendRequest?.[1]?.headers).toMatchObject({ "Idempotency-Key": "40000000-0000-4000-8000-000000000001" });
    expect(screen.queryByText(/senha|código|matrícula:/i)).not.toBeInTheDocument();
  });

  it("does not offer blind resend after an ambiguous provider result", async () => {
    vi.stubGlobal("crypto", { randomUUID: vi.fn(() => "40000000-0000-4000-8000-000000000002") });
    vi.spyOn(global, "fetch").mockResolvedValueOnce(new Response(JSON.stringify({ ...invitation, delivery_status: "unknown" }), { status: 201 }));
    render(<AthleteInvitationForm teams={teams} />);
    fireEvent.change(screen.getByLabelText("Nome completo"), { target: { value: "Atleta fictício" } });
    fireEvent.change(screen.getByLabelText("Celular"), { target: { value: "+15555550123" } });
    fireEvent.change(screen.getByRole("textbox", { name: "E-mail" }), { target: { value: "athlete@example.test" } });
    fireEvent.change(screen.getByLabelText("Data de nascimento"), { target: { value: "2010-01-15" } });
    fireEvent.change(screen.getByLabelText("Bairro/cidade"), { target: { value: "Cidade fictícia" } });
    fireEvent.change(screen.getByLabelText("Turma"), { target: { value: teamId } });
    fireEvent.change(screen.getByLabelText("Posição"), { target: { value: "midfielder" } });
    fireEvent.submit(screen.getByRole("button", { name: /Criar convite/ }).closest("form")!);
    expect(await screen.findByText(/requer verificação/i)).toBeVisible();
    expect(screen.queryByRole("button", { name: "Reenviar matrícula" })).not.toBeInTheDocument();
  });
});
