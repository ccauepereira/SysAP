import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { AthleteInvitationForm } from "./athlete-invitation-form";
import { ManualActivationCode } from "./manual-activation-code";

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

function fillRequiredFields() {
  fireEvent.change(screen.getByLabelText("Nome completo"), { target: { value: "Atleta fictício" } });
  fireEvent.change(screen.getByLabelText("Celular"), { target: { value: "+15555550123" } });
  fireEvent.change(screen.getByRole("textbox", { name: "E-mail" }), { target: { value: "athlete@example.test" } });
  fireEvent.change(screen.getByLabelText("Data de nascimento"), { target: { value: "2010-01-15" } });
  fireEvent.change(screen.getByLabelText("Bairro/cidade"), { target: { value: "Cidade fictícia" } });
  fireEvent.change(screen.getByLabelText("Turma"), { target: { value: teamId } });
  fireEvent.change(screen.getByLabelText("Posição"), { target: { value: "midfielder" } });
}

afterEach(() => {
  vi.restoreAllMocks();
  vi.unstubAllGlobals();
});

describe("AthleteInvitationForm delivery recovery", () => {
  it("creates a manual invitation, displays the generated enrollment number, normalizes BR phone numbers, and shows the one-time code only after issuance", async () => {
    vi.stubGlobal("crypto", { randomUUID: vi.fn(() => "40000000-0000-4000-8000-000000000001") });
    const fetchMock = vi.spyOn(global, "fetch")
      .mockResolvedValueOnce(new Response(JSON.stringify({ ...invitation, enrollment_number: "2026000001", delivery_channel: "manual", delivery_status: "not_applicable" }), { status: 201 }))
      .mockResolvedValueOnce(new Response(JSON.stringify({ enrollment_number: "2026000001", code: "123456", expires_at: "2026-10-31T13:00:00Z" }), { status: 201 }));
    render(<AthleteInvitationForm teams={teams} />);
    fillRequiredFields();
    fireEvent.change(screen.getByLabelText("Celular"), { target: { value: "(85) 99999-0123" } });
    fireEvent.submit(screen.getByRole("button", { name: /Criar convite/ }).closest("form")!);
    expect(await screen.findByText("Matrícula gerada: 2026000001")).toBeVisible();
    expect(screen.getByRole("status")).toHaveTextContent("Matrícula 2026000001 gerada");
    fireEvent.click(await screen.findByRole("button", { name: "Gerar código de ativação manual" }));
    expect(await screen.findByText("Matrícula: 2026000001")).toBeVisible();
    expect(screen.getByText("Código temporário: 123456")).toBeVisible();
    expect(JSON.parse(String(fetchMock.mock.calls[0]?.[1]?.body))).toMatchObject({
      phone_e164: "+5585999990123",
      enrollment_delivery_channel: "manual",
    });
  });
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
    fireEvent.click(screen.getByRole("radio", { name: "E-mail (quando configurado)" }));
    fireEvent.submit(screen.getByRole("button", { name: /Criar convite/ }).closest("form")!);

    expect(await screen.findByText("Ativação: pendente")).toBeVisible();
    expect(screen.getByText("Entrega: falha confirmada")).toBeVisible();
    const createRequest = fetchMock.mock.calls.at(0);
    expect(JSON.parse(String(createRequest?.[1]?.body))).not.toHaveProperty("enrollment_delivery_channel_mock");
    expect(JSON.parse(String(createRequest?.[1]?.body))).toMatchObject({ team_id: teamId, enrollment_delivery_channel: "email" });
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

  it("requires refreshing delivery state after an ambiguous resend request", async () => {
    vi.stubGlobal("crypto", { randomUUID: vi.fn(() => "40000000-0000-4000-8000-000000000003") });
    const fetchMock = vi.spyOn(global, "fetch")
      .mockResolvedValueOnce(new Response(JSON.stringify(invitation), { status: 201 }))
      .mockRejectedValueOnce(new Error("network"))
      .mockResolvedValueOnce(new Response(JSON.stringify({ ...invitation, delivery_status: "unknown" }), { status: 200 }));
    render(<AthleteInvitationForm teams={teams} />);
    fillRequiredFields();
    fireEvent.submit(screen.getByRole("button", { name: /Criar convite/ }).closest("form")!);
    fireEvent.click(await screen.findByRole("button", { name: "Reenviar matrícula" }));
    expect(await screen.findByText(/Atualize o estado do convite antes de tentar outro reenvio/)).toBeVisible();
    expect(screen.queryByRole("button", { name: "Reenviar matrícula" })).not.toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "Atualizar estado" }));
    expect(await screen.findByText(/requer verificação/i)).toBeVisible();
    expect(fetchMock).toHaveBeenCalledTimes(3);
    expect(screen.queryByRole("button", { name: "Reenviar matrícula" })).not.toBeInTheDocument();
  });

  it("explains that the form requires an available team list", () => {
    render(<AthleteInvitationForm />);
    expect(screen.getByText(/Nenhuma turma ativa encontrada/i)).toBeVisible();
    expect(screen.queryByRole("button", { name: /Criar convite/ })).not.toBeInTheDocument();
  });

  it("does not submit changed data after an ambiguous creation attempt", async () => {
    vi.stubGlobal("crypto", { randomUUID: vi.fn(() => "40000000-0000-4000-8000-000000000004") });
    const fetchMock = vi.spyOn(global, "fetch")
      .mockRejectedValueOnce(new Error("network"))
      .mockResolvedValueOnce(new Response(JSON.stringify(invitation), { status: 201 }));
    render(<AthleteInvitationForm teams={teams} />);
    fillRequiredFields();
    const name = screen.getByLabelText("Nome completo");
    fireEvent.change(name, { target: { value: "Atleta fictício" } });
    const form = screen.getByRole("button", { name: /Criar convite/ }).closest("form")!;
    fireEvent.submit(form);
    expect(await screen.findByText(/Não foi possível confirmar o cadastro/)).toBeVisible();
    fireEvent.change(name, { target: { value: "Outro atleta" } });
    fireEvent.submit(form);
    expect(await screen.findByText(/Restaure os dados anteriores/)).toBeVisible();
    expect(fetchMock).toHaveBeenCalledTimes(1);
    fireEvent.change(name, { target: { value: "Atleta fictício" } });
    fireEvent.submit(form);
    expect(await screen.findByText("Ativação: pendente")).toBeVisible();
    expect(fetchMock).toHaveBeenCalledTimes(2);
    expect(fetchMock.mock.calls[1]?.[1]?.headers).toEqual(fetchMock.mock.calls[0]?.[1]?.headers);
  });

  it("does not send an incomplete form or present a previous success as a new submission", async () => {
    vi.stubGlobal("crypto", { randomUUID: vi.fn(() => "40000000-0000-4000-8000-000000000005") });
    const fetchMock = vi.spyOn(global, "fetch").mockResolvedValueOnce(new Response(JSON.stringify(invitation), { status: 201 }));
    render(<AthleteInvitationForm teams={teams} />);
    fillRequiredFields();
    const form = screen.getByRole("button", { name: /Criar convite/ }).closest("form");
    if (!form) throw new Error("Missing invitation form");
    fireEvent.submit(form);
    expect(await screen.findByText(/Convite criado e conta pendente/)).toBeVisible();
    fireEvent.submit(form);
    expect(screen.getByRole("alert")).toHaveTextContent("Preencha os campos obrigatórios");
    expect(screen.queryByText(/Convite criado e conta pendente/)).not.toBeInTheDocument();
    expect(screen.getByText(/Último convite criado/)).toBeVisible();
    expect(fetchMock).toHaveBeenCalledTimes(1);
  });

  it("simulates an invitation without calling the API", async () => {
    const fetchMock = vi.spyOn(global, "fetch");
    const onDemoCreate = vi.fn();
    render(<AthleteInvitationForm teams={teams} onDemoCreate={onDemoCreate} />);
    const form = screen.getByRole("button", { name: /Criar convite/ }).closest("form")!;
    fireEvent.submit(form);
    expect(screen.getByRole("alert")).toHaveTextContent("Preencha todos os campos");
    fireEvent.change(screen.getByLabelText("Nome completo"), { target: { value: "Atleta fictício" } });
    fireEvent.change(screen.getByLabelText("Celular"), { target: { value: "+15555550123" } });
    fireEvent.change(screen.getByRole("textbox", { name: "E-mail" }), { target: { value: "athlete@example.test" } });
    fireEvent.change(screen.getByLabelText("Data de nascimento"), { target: { value: "2010-01-15" } });
    fireEvent.change(screen.getByLabelText("Bairro/cidade"), { target: { value: "Cidade fictícia" } });
    fireEvent.change(screen.getByLabelText("Turma"), { target: { value: teamId } });
    fireEvent.change(screen.getByLabelText("Posição"), { target: { value: "midfielder" } });
    fireEvent.submit(form);
    expect(screen.getByRole("button", { name: /Enviando/ })).toBeDisabled();
    expect(await screen.findByText(/Convite criado na demonstração/)).toBeVisible();
    expect(onDemoCreate).toHaveBeenCalledOnce();
    expect(fetchMock).not.toHaveBeenCalled();
  });

  it("does not create an invitation after leaving or restoring during loading", async () => {
    const onDemoCreate = vi.fn();
    const view = render(<AthleteInvitationForm teams={teams} onDemoCreate={onDemoCreate} />);
    fireEvent.change(screen.getByLabelText("Nome completo"), { target: { value: "Atleta fictício" } });
    fireEvent.change(screen.getByLabelText("Celular"), { target: { value: "+15555550123" } });
    fireEvent.change(screen.getByRole("textbox", { name: "E-mail" }), { target: { value: "athlete@example.test" } });
    fireEvent.change(screen.getByLabelText("Data de nascimento"), { target: { value: "2010-01-15" } });
    fireEvent.change(screen.getByLabelText("Bairro/cidade"), { target: { value: "Cidade fictícia" } });
    fireEvent.change(screen.getByLabelText("Turma"), { target: { value: teamId } });
    fireEvent.change(screen.getByLabelText("Posição"), { target: { value: "midfielder" } });
    fireEvent.submit(screen.getByRole("button", { name: /Criar convite/ }).closest("form")!);
    view.unmount();
    await new Promise((resolve) => setTimeout(resolve, 450));
    expect(onDemoCreate).not.toHaveBeenCalled();
  });
});

describe("ManualActivationCode issuance", () => {
  const invitationId = "20000000-0000-4000-8000-000000000001";

  it("shows enrollment, code and expiry when issuance is confirmed", async () => {
    const fetchMock = vi.spyOn(global, "fetch").mockResolvedValueOnce(new Response(JSON.stringify({ enrollment_number: "2026000001", code: "654321", expires_at: "2026-10-31T19:00:00Z" }), { status: 201 }));
    render(<ManualActivationCode invitationId={invitationId} />);
    fireEvent.click(screen.getByRole("button", { name: "Gerar código de ativação manual" }));
    expect(await screen.findByText("Código temporário: 654321")).toBeVisible();
    expect(screen.getByText("Matrícula: 2026000001")).toBeVisible();
    expect(screen.getByText(/Válido até:/)).toBeVisible();
    expect(fetchMock).toHaveBeenCalledTimes(1);
  });

  it("explains an expired admin session (401) without suggesting a blind retry", async () => {
    vi.spyOn(global, "fetch").mockResolvedValueOnce(new Response(JSON.stringify({ error: "authentication_required" }), { status: 401 }));
    render(<ManualActivationCode invitationId={invitationId} />);
    fireEvent.click(screen.getByRole("button", { name: "Gerar código de ativação manual" }));
    expect(await screen.findByRole("alert")).toHaveTextContent(/sessão administrativa expirou/);
  });

  it("explains missing MFA (403 mfa_required)", async () => {
    vi.spyOn(global, "fetch").mockResolvedValueOnce(new Response(JSON.stringify({ error: "mfa_required" }), { status: 403 }));
    render(<ManualActivationCode invitationId={invitationId} />);
    fireEvent.click(screen.getByRole("button", { name: "Gerar código de ativação manual" }));
    expect(await screen.findByRole("alert")).toHaveTextContent(/duas etapas/);
  });

  it("does not suggest a blind reissue after a network failure", async () => {
    const fetchMock = vi.spyOn(global, "fetch").mockRejectedValueOnce(new Error("network"));
    render(<ManualActivationCode invitationId={invitationId} />);
    fireEvent.click(screen.getByRole("button", { name: "Gerar código de ativação manual" }));
    expect(await screen.findByRole("alert")).toHaveTextContent(/A emissão não foi confirmada/);
    expect(screen.getByRole("alert")).toHaveTextContent(/Não solicite outro código agora/);
    expect(screen.queryByText(/Código temporário/)).not.toBeInTheDocument();
    expect(fetchMock).toHaveBeenCalledTimes(1);
  });

  it("treats server errors as an ambiguous result and never retries automatically", async () => {
    const fetchMock = vi.spyOn(global, "fetch").mockResolvedValueOnce(new Response(JSON.stringify({ error: "issuance_unknown" }), { status: 503 }));
    render(<ManualActivationCode invitationId={invitationId} />);
    fireEvent.click(screen.getByRole("button", { name: "Gerar código de ativação manual" }));
    expect(await screen.findByRole("alert")).toHaveTextContent(/Não solicite outro código agora/);
    expect(fetchMock).toHaveBeenCalledTimes(1);
  });

  it("does not display malformed success data and treats it as ambiguous", async () => {
    const fetchMock = vi.spyOn(global, "fetch").mockResolvedValueOnce(new Response(JSON.stringify({ enrollment_number: "2026000001", code: "bad", expires_at: "2026-10-31T19:00:00Z" }), { status: 201 }));
    render(<ManualActivationCode invitationId={invitationId} />);
    fireEvent.click(screen.getByRole("button", { name: "Gerar código de ativação manual" }));
    expect(await screen.findByRole("alert")).toHaveTextContent(/Não solicite outro código agora/);
    expect(screen.queryByText(/Código temporário/)).not.toBeInTheDocument();
    expect(fetchMock).toHaveBeenCalledTimes(1);
  });
});
