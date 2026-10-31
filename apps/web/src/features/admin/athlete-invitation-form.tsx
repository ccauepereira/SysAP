"use client";

import { useEffect, useRef, useState, type FormEvent } from "react";
import { LoaderCircle, Send } from "lucide-react";
import { authStyles as styles } from "@/features/auth/auth-shell";
import { ManualActivationCode } from "./manual-activation-code";

type Invitation = {
  invitation_id: string;
  enrollment_number?: string;
  status: "pending_activation" | "activated" | "expired";
  expires_at: string;
  delivery_channel: "email" | "sms" | "manual";
  delivery_status: "sending" | "sent" | "failed" | "unknown" | "not_applicable";
};

function normalizePhoneE164(raw: string): string {
  const trimmed = raw.trim();
  if (!trimmed) return "";
  const digits = trimmed.replace(/\D/g, "");
  if (!digits) return trimmed;
  if (trimmed.startsWith("+")) return `+${digits}`;
  if ((digits.length === 10 || digits.length === 11) && !digits.startsWith("55")) {
    return `+55${digits}`;
  }
  return `+${digits}`;
}

function readInvitation(value: unknown): Invitation {
  if (typeof value !== "object" || value === null) throw new Error("Invalid invitation response");
  const item = Object.fromEntries(Object.entries(value));
  if (typeof item.invitation_id !== "string" || !/^[0-9a-f]{8}-[0-9a-f]{4}-[1-8][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$/i.test(item.invitation_id)
    || (item.status !== "pending_activation" && item.status !== "activated" && item.status !== "expired")
    || typeof item.expires_at !== "string" || Number.isNaN(Date.parse(item.expires_at))
    || (item.delivery_channel !== "email" && item.delivery_channel !== "sms" && item.delivery_channel !== "manual")
    || (item.delivery_status !== "sending" && item.delivery_status !== "sent" && item.delivery_status !== "failed" && item.delivery_status !== "unknown" && item.delivery_status !== "not_applicable")) {
    throw new Error("Invalid invitation response");
  }
  const enrollmentNumber = typeof item.enrollment_number === "string" && /^\d{10}$/.test(item.enrollment_number)
    ? item.enrollment_number
    : undefined;
  return {
    invitation_id: item.invitation_id,
    ...(enrollmentNumber ? { enrollment_number: enrollmentNumber } : {}),
    status: item.status,
    expires_at: item.expires_at,
    delivery_channel: item.delivery_channel,
    delivery_status: item.delivery_status,
  };
}

export type TeamOption = {
  readonly id: string;
  readonly name: string;
};

export function AthleteInvitationForm({ teams = [], onDemoCreate }: { teams?: TeamOption[], organizationId?: string; onDemoCreate?: (data: FormData) => void }) {
  const [error, setError] = useState("");
  const [success, setSuccess] = useState("");
  const [loading, setLoading] = useState(false);
  const [deliveryLoading, setDeliveryLoading] = useState(false);
  const [resendNeedsRefresh, setResendNeedsRefresh] = useState(false);
  const [invitation, setInvitation] = useState<Invitation | null>(null);
  const createIdempotencyKey = useRef<string | null>(null);
  const createPayload = useRef<string | null>(null);
  const resendIdempotencyKey = useRef<string | null>(null);
  const submitInFlight = useRef(false);
  const mounted = useRef(true);
  useEffect(() => {
    mounted.current = true;
    return () => { mounted.current = false; };
  }, []);

  async function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const form = event.currentTarget;
    if (submitInFlight.current) return;
    setError("");
    setSuccess("");
    if (!form.checkValidity()) {
      setError(onDemoCreate ? "Preencha todos os campos para simular o cadastro." : "Preencha os campos obrigatórios e confira o e-mail informado.");
      return;
    }
    submitInFlight.current = true;
    setLoading(true);
    const formData = new FormData(form);
    formData.delete("enrollment_delivery_channel_mock");

    if (onDemoCreate) {
      if (!["full_name", "phone_e164", "email", "birth_date", "locality", "team_id", "football_position"].every((field) => String(formData.get(field) ?? "").trim())) {
        setError("Preencha todos os campos para simular o cadastro.");
        setLoading(false);
        submitInFlight.current = false;
        return;
      }
      await new Promise((resolve) => setTimeout(resolve, 400));
      if (!mounted.current) return;
      onDemoCreate(formData);
      form.reset();
      setSuccess("Convite criado na demonstração. Nenhum e-mail foi enviado.");
      setLoading(false);
      submitInFlight.current = false;
      return;
    }

    const rawPhone = String(formData.get("phone_e164") ?? "");
    if (rawPhone) {
      formData.set("phone_e164", normalizePhoneE164(rawPhone));
    }
    const payload = JSON.stringify(Object.fromEntries(formData.entries()));
    if (createIdempotencyKey.current && createPayload.current !== payload) {
      setError("O resultado anterior ainda não foi confirmado. Restaure os dados anteriores para tentar novamente com segurança.");
      setLoading(false);
      submitInFlight.current = false;
      return;
    }
    createIdempotencyKey.current ??= crypto.randomUUID();
    createPayload.current = payload;

    try {
      const response = await fetch("/api/admin/athlete-invitations", {
        method: "POST",
        headers: { "Content-Type": "application/json", "Idempotency-Key": createIdempotencyKey.current },
        body: payload,
      });
      if (!response.ok) {
        let errorCode = "";
        try {
          const errorBody = await response.json();
          if (errorBody && typeof errorBody === "object" && "error" in errorBody && typeof errorBody.error === "string") {
            errorCode = errorBody.error;
          }
        } catch {
          // ignore non-json error response
        }
        if ((response.status >= 400 && response.status < 500 && response.status !== 409) || errorCode === "data_conflict") {
          createIdempotencyKey.current = null;
          createPayload.current = null;
        }
        setError(
          errorCode === "data_conflict"
            ? "Já existe um atleta com estes dados ou um conflito no cadastro. Confira os dados informados."
            : response.status >= 500 || response.status === 409
              ? "Não foi possível confirmar o cadastro. Tente novamente sem alterar os dados."
              : response.status === 403
                ? "Confira sua permissão e a confirmação de identidade em duas etapas."
                : "Confira os campos e tente novamente."
        );
        return;
      }
      const result = readInvitation(await response.json());
      setInvitation(result);
      setResendNeedsRefresh(false);
      createIdempotencyKey.current = null;
      createPayload.current = null;
      setSuccess(
        result.delivery_status === "not_applicable"
          ? result.enrollment_number
            ? `Cadastro autorizado. Matrícula ${result.enrollment_number} gerada. Gere um código manual para o aluno.`
            : "Cadastro autorizado. Gere um código manual para o aluno."
          : result.delivery_status === "sent"
            ? "Convite criado; a solicitação de envio foi aceita pelo serviço de e-mail."
            : "Convite criado e conta pendente de ativação. A entrega não foi confirmada."
      );
      form.reset();
    } catch {
      setError("Não foi possível confirmar o cadastro. Tente novamente sem alterar os dados.");
    } finally {
      setLoading(false);
      submitInFlight.current = false;
    }
  }


  async function refreshDelivery() {
    if (!invitation) return;
    setDeliveryLoading(true);
    setError("");
    try {
      const response = await fetch(`/api/admin/athlete-invitations/${invitation.invitation_id}`, { cache: "no-store" });
      if (!response.ok) {
        if (response.status === 401) {
          setError("Sua sessão administrativa expirou ou foi encerrada (por exemplo, ao sair em outra aba). Faça login novamente.");
          return;
        }
        throw new Error("unavailable");
      }
      const next = readInvitation(await response.json());
      setInvitation((prev) => ({
        ...next,
        ...(next.enrollment_number ? {} : prev?.enrollment_number ? { enrollment_number: prev.enrollment_number } : {}),
      }));
      setResendNeedsRefresh(false);
    } catch {
      setError("Não foi possível consultar o estado agora. Tente novamente.");
    } finally {
      setDeliveryLoading(false);
    }
  }

  async function resendInvitation() {
    if (!invitation) return;
    resendIdempotencyKey.current ??= crypto.randomUUID();
    setDeliveryLoading(true);
    setError("");
    try {
      const response = await fetch(`/api/admin/athlete-invitations/${invitation.invitation_id}/resend`, {
        method: "POST",
        headers: { "Idempotency-Key": resendIdempotencyKey.current },
      });
      if (!response.ok) {
        if (response.status >= 500) setResendNeedsRefresh(true);
        setError(response.status === 429 ? "Limite de reenvios atingido. Tente mais tarde." : "O reenvio não está disponível para este estado do convite.");
        return;
      }
      const next = readInvitation(await response.json());
      setInvitation((prev) => ({
        ...next,
        ...(next.enrollment_number ? {} : prev?.enrollment_number ? { enrollment_number: prev.enrollment_number } : {}),
      }));
      resendIdempotencyKey.current = null;
      setSuccess("Nova tentativa de entrega registrada; a conta continua pendente de ativação.");
    } catch {
      setResendNeedsRefresh(true);
      setError("Não foi possível confirmar a tentativa. Atualize o estado antes de tentar novamente.");
    } finally {
      setDeliveryLoading(false);
    }
  }

  if (teams.length === 0) {
    return (
      <div className={styles.form}>
        <p style={{ textAlign: "center", color: "var(--sysap-color-text-dimmed)" }}>
          Nenhuma turma ativa encontrada. Crie uma turma para cadastrar o atleta.
        </p>
      </div>
    );
  }

  return (
    <form className={styles.form} noValidate onChange={() => setSuccess("")} onSubmit={submit} aria-describedby={error ? "invitation-form-error" : undefined}>
      <p>Ao cadastrar, você autoriza o aluno a ativar o acesso. A conta permanece pendente até a ativação ser concluída.</p>
      <div className={styles.field}>
        <label className={styles.label} htmlFor="admin-full-name">Nome completo</label>
        <input id="admin-full-name" name="full_name" required />
      </div>
      <div className={styles.field}>
        <label className={styles.label} htmlFor="admin-phone">Celular</label>
        <input id="admin-phone" name="phone_e164" inputMode="tel" placeholder="Ex.: (85) 99999-0000" required />
      </div>
      <div className={styles.field}>
        <label className={styles.label} htmlFor="admin-email">E-mail</label>
        <input id="admin-email" name="email" type="email" autoComplete="email" required />
      </div>
      <div className={styles.field}>
        <label className={styles.label} htmlFor="admin-birth">Data de nascimento</label>
        <input id="admin-birth" name="birth_date" type="date" required />
      </div>
      <div className={styles.field}>
        <label className={styles.label} htmlFor="admin-locality">Bairro/cidade</label>
        <input id="admin-locality" name="locality" required />
      </div>
      <div className={styles.field}>
        <label className={styles.label} htmlFor="admin-team">Turma</label>
        <select id="admin-team" name="team_id" required>
          <option value="">Selecione a turma</option>
          {teams.map((t: { id: string; name: string }) => (
            <option key={t.id} value={t.id}>{t.name}</option>
          ))}
        </select>
      </div>
      <div className={styles.field}>
        <label className={styles.label} htmlFor="admin-position">Posição</label>
        <select id="admin-position" name="football_position" required>
          <option value="">Selecione a posição</option>
          <option value="goalkeeper">Goleiro</option>
          <option value="defender">Defensor</option>
          <option value="midfielder">Meio-campista</option>
          <option value="forward">Atacante</option>
        </select>
      </div>
      
      <fieldset className={styles.field}>
        <legend className={styles.label}>Entrega da matrícula</legend>
        <label><input defaultChecked name="enrollment_delivery_channel" type="radio" value="manual" /> Repasse manual</label>
        <label style={{ opacity: 0.5, cursor: "not-allowed" }}>
          <input disabled name="enrollment_delivery_channel_mock" type="radio" value="sms" /> SMS (Em breve)
        </label>
        <label>
          <input name="enrollment_delivery_channel" type="radio" value="email" /> E-mail (quando configurado)
        </label>
      </fieldset>

      {error ? <p className={styles.formError} id="invitation-form-error" role="alert">{error}</p> : null}
      {success ? <p className={styles.success} role="status">{success}</p> : null}
      {invitation ? (
        <section aria-label="Acompanhamento do convite" className={styles.success}>
          <p>Último convite criado — o acompanhamento abaixo não corresponde a um novo envio.</p>
          {invitation.enrollment_number ? <p><strong>Matrícula gerada: {invitation.enrollment_number}</strong></p> : null}
          <p>Ativação: {invitation.status === "pending_activation" ? "pendente" : invitation.status === "activated" ? "concluída" : "expirada"}</p>
          <p>Entrega: {invitation.delivery_status === "not_applicable" ? "manual; nenhum e-mail enviado" : invitation.delivery_status === "sent" ? "aceita pelo serviço de e-mail" : invitation.delivery_status === "failed" ? "falha confirmada" : invitation.delivery_status === "unknown" ? "não confirmada; requer verificação" : "em processamento"}</p>
          <p>Expira em: {new Date(invitation.expires_at).toLocaleDateString("pt-BR")}</p>
          {invitation.status === "pending_activation" && invitation.delivery_channel === "manual" ? <ManualActivationCode key={invitation.invitation_id} invitationId={invitation.invitation_id} /> : null}
          <button className={styles.secondary} disabled={deliveryLoading} onClick={refreshDelivery} type="button">Atualizar estado</button>
          {invitation.delivery_status === "failed" && invitation.status === "pending_activation" && !resendNeedsRefresh ? <button className={styles.secondary} disabled={deliveryLoading} onClick={resendInvitation} type="button">Reenviar matrícula</button> : null}
          {resendNeedsRefresh ? <p>Atualize o estado do convite antes de tentar outro reenvio.</p> : null}
          {invitation.delivery_status === "unknown" ? <p>Não tente reenviar antes de confirmar com a equipe se a mensagem foi aceita.</p> : null}
        </section>
      ) : null}
      
      <button className={styles.primary} disabled={loading} type="submit">
        {loading ? <><LoaderCircle className={styles.spinner} size={18} /> Enviando…</> : <><Send size={18} /> Criar convite e enviar matrícula</>}
      </button>
    </form>
  );
}
