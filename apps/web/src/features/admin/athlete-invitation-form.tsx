"use client";

import { useState, type FormEvent } from "react";
import { LoaderCircle, Send } from "lucide-react";
import { authStyles as styles } from "@/features/auth/auth-shell";

export function AthleteInvitationForm({ teams = [] }: { teams?: { id: string; name: string }[], organizationId?: string }) {
  const [error, setError] = useState("");
  const [success, setSuccess] = useState("");
  const [loading, setLoading] = useState(false);

  async function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setError("");
    setSuccess("");
    setLoading(true);

    const formData = new FormData(event.currentTarget);
    formData.set("enrollment_delivery_channel", "email"); // Sempre email

    const data = Object.fromEntries(formData.entries());

    try {
      const response = await fetch("/api/admin/athlete-invitations", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(data),
      });
      if (!response.ok) {
        setError(
          response.status === 503
            ? "Não foi possível enviar agora. Tente novamente mais tarde."
            : "Confira os campos e tente novamente."
        );
        return;
      }
      setSuccess("Convite criado e matrícula enviada com sucesso para o e-mail do atleta.");
      event.currentTarget.reset();
    } catch {
      setError("Não foi possível enviar agora. Tente novamente mais tarde.");
    } finally {
      setLoading(false);
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
      
      <button className={styles.primary} disabled={loading} type="submit">
        {loading ? <><LoaderCircle className={styles.spinner} size={18} /> Enviando…</> : <><Send size={18} /> Criar convite e enviar matrícula</>}
      </button>
    </form>
  );
}
