"use client";

import { useState, type FormEvent } from "react";
import { LoaderCircle, Send } from "lucide-react";
import { authStyles as styles } from "@/features/auth/auth-shell";

export function AthleteInvitationForm() {
  const [error, setError] = useState(""); const [success, setSuccess] = useState(""); const [loading, setLoading] = useState(false);
  async function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault(); setError(""); setSuccess(""); setLoading(true);
    const data = Object.fromEntries(new FormData(event.currentTarget).entries());
    try { const response = await fetch("/api/admin/athlete-invitations", { method: "POST", headers: { "Content-Type": "application/json" }, body: JSON.stringify(data) }); if (!response.ok) { setError(response.status === 503 ? "Não foi possível enviar agora. Tente novamente mais tarde." : "Confira os campos e tente novamente."); return; } setSuccess("Convite criado e matrícula enviada pelo canal selecionado."); event.currentTarget.reset(); } catch { setError("Não foi possível enviar agora. Tente novamente mais tarde."); } finally { setLoading(false); }
  }
  return <form className={styles.form} noValidate onSubmit={submit}>
    <div className={styles.field}><label className={styles.label} htmlFor="admin-full-name">Nome completo</label><input id="admin-full-name" name="full_name" required /></div>
    <div className={styles.field}><label className={styles.label} htmlFor="admin-phone">Celular</label><input id="admin-phone" name="phone_e164" inputMode="tel" required /></div>
    <div className={styles.field}><label className={styles.label} htmlFor="admin-email">E-mail</label><input id="admin-email" name="email" type="email" autoComplete="email" required /></div>
    <div className={styles.field}><label className={styles.label} htmlFor="admin-birth">Data de nascimento</label><input id="admin-birth" name="birth_date" type="date" required /></div>
    <div className={styles.field}><label className={styles.label} htmlFor="admin-locality">Bairro/cidade</label><input id="admin-locality" name="locality" required /></div>
    <div className={styles.field}><label className={styles.label} htmlFor="admin-team">Turma</label><select id="admin-team" name="team_id" required><option value="">Selecione a turma</option><option value="00000000-0000-4000-8000-000000000001">Turma de homologação</option></select></div>
    <div className={styles.field}><label className={styles.label} htmlFor="admin-position">Posição</label><select id="admin-position" name="football_position" required><option value="">Selecione a posição</option><option value="goalkeeper">Goleiro</option><option value="defender">Defensor</option><option value="midfielder">Meio-campista</option><option value="forward">Atacante</option></select></div>
    <fieldset className={styles.field}><legend className={styles.label}>Enviar matrícula por</legend><label><input defaultChecked name="enrollment_delivery_channel" type="radio" value="sms" /> SMS</label><label><input name="enrollment_delivery_channel" type="radio" value="email" /> E-mail</label></fieldset>
    {error ? <p className={styles.formError} role="alert">{error}</p> : null}{success ? <p className={styles.success} role="status">{success}</p> : null}
    <button className={styles.primary} disabled={loading} type="submit">{loading ? <><LoaderCircle className={styles.spinner} size={18} /> Enviando…</> : <><Send size={18} /> Criar convite e enviar matrícula</>}</button>
  </form>;
}
