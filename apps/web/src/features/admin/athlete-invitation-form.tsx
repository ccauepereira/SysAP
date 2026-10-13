"use client";

import { useRef, useState, type FormEvent } from "react";
import { LoaderCircle, Send } from "lucide-react";
import { authStyles as styles } from "@/features/auth/auth-shell";

export function AthleteInvitationForm({ teams = [] }: { teams?: { id: string; name: string }[], organizationId?: string }) {
  const [error, setError] = useState("");
  const [success, setSuccess] = useState("");
  const [loading, setLoading] = useState(false);
  const [deliveryLoading, setDeliveryLoading] = useState(false);
  const [invitation, setInvitation] = useState<{ invitation_id: string; status: string; expires_at: string; delivery_channel: string; delivery_status: string } | null>(null);
  const createIdempotencyKey = useRef<string | null>(null);
  const resendIdempotencyKey = useRef<string | null>(null);

  async function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setError("");
    setSuccess("");
    setLoading(true);
    createIdempotencyKey.current ??= crypto.randomUUID();

    const formData = new FormData(event.currentTarget);
    formData.set("enrollment_delivery_channel", "email"); // Sempre email

    const data = Object.fromEntries(formData.entries());

    try {
      const response = await fetch("/api/admin/athlete-invitations", {
        method: "POST",
        headers: { "Content-Type": "application/json", "Idempotency-Key": createIdempotencyKey.current },
        body: JSON.stringify(data),
      });
      if (!response.ok) {
        if (response.status >= 400 && response.status < 500) createIdempotencyKey.current = null;
        setError(
          response.status === 503
            ? "Não foi possível enviar agora. Tente novamente mais tarde."
            : "Confira os campos e tente novamente."
        );
        return;
      }
      const result = await response.json() as { invitation_id: string; status: string; expires_at: string; delivery_channel: string; delivery_status: string };
      setInvitation(result);
      createIdempotencyKey.current = null;
      setSuccess(result.delivery_status === "sent" ? "Convite criado; a solicitação de envio foi aceita pelo serviço de e-mail." : "Convite criado e conta pendente de ativação. A entrega não foi confirmada.");
      event.currentTarget.reset();
    } catch {
      setError("Não foi possível enviar agora. Tente novamente mais tarde.");
    } finally {
      setLoading(false);
    }
  }

  async function refreshDelivery() {
    if (!invitation) return;
    setDeliveryLoading(true);
    setError("");
    try {
      const response = await fetch(`/api/admin/athlete-invitations/${invitation.invitation_id}`, { cache: "no-store" });
      if (!response.ok) throw new Error("unavailable");
      setInvitation(await response.json());
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
        setError(response.status === 429 ? "Limite de reenvios atingido. Tente mais tarde." : "O reenvio não está disponível para este estado do convite.");
        return;
      }
      setInvitation(await response.json());
      resendIdempotencyKey.current = null;
      setSuccess("Nova tentativa de entrega registrada; a conta continua pendente de ativação.");
    } catch {
      setError("Não foi possível confirmar a tentativa. Atualize o estado antes de tentar novamente.");
    } finally {
      setDeliveryLoading(false);
    }
  }

  if (teams.length === 0) {
    return (
      <div className={styles.form}>
        <p style={{ textAlign: "center", color: "var(--sysap-color-text-dimmed)" }}>
          É necessário ter pelo menos uma turma cadastrada antes de convidar um atleta.
        </p>
      </div>
    );
  }

  return (
    <form className={styles.form} noValidate onSubmit={submit}>
      <div className={styles.field}>
        <label className={styles.label} htmlFor="admin-full-name">Nome completo</label>
        <input id="admin-full-name" name="full_name" required />
      </div>
      <div className={styles.field}>
        <label className={styles.label} htmlFor="admin-phone">Celular</label>
        <input id="admin-phone" name="phone_e164" inputMode="tel" required />
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
        <legend className={styles.label}>Enviar matrícula por</legend>
        <label style={{ opacity: 0.5, cursor: "not-allowed" }}>
          <input disabled name="enrollment_delivery_channel_mock" type="radio" value="sms" /> SMS (Em breve)
        </label>
        <label>
          <input defaultChecked name="enrollment_delivery_channel_mock" type="radio" value="email" /> E-mail
        </label>
      </fieldset>

      {error ? <p className={styles.formError} role="alert">{error}</p> : null}
      {success ? <p className={styles.success} role="status">{success}</p> : null}
      {invitation ? (
        <section aria-label="Acompanhamento do convite" className={styles.success}>
          <p>Ativação: {invitation.status === "pending_activation" ? "pendente" : invitation.status === "activated" ? "concluída" : "expirada"}</p>
          <p>Entrega: {invitation.delivery_status === "sent" ? "aceita pelo serviço de e-mail" : invitation.delivery_status === "failed" ? "falha confirmada" : invitation.delivery_status === "unknown" ? "não confirmada; requer verificação" : "em processamento"}</p>
          <p>Expira em: {new Date(invitation.expires_at).toLocaleDateString("pt-BR")}</p>
          <button className={styles.secondary} disabled={deliveryLoading} onClick={refreshDelivery} type="button">Atualizar estado</button>
          {invitation.delivery_status === "failed" && invitation.status === "pending_activation" ? <button className={styles.secondary} disabled={deliveryLoading} onClick={resendInvitation} type="button">Reenviar matrícula</button> : null}
          {invitation.delivery_status === "unknown" ? <p>Não tente reenviar antes de confirmar com a equipe se a mensagem foi aceita.</p> : null}
        </section>
      ) : null}
      
      <button className={styles.primary} disabled={loading} type="submit">
        {loading ? <><LoaderCircle className={styles.spinner} size={18} /> Enviando…</> : <><Send size={18} /> Criar convite e enviar matrícula</>}
      </button>
    </form>
  );
}
