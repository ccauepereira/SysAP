"use client";

import { useState, type FormEvent } from "react";
import { useRouter } from "next/navigation";
import { ArrowRight, LoaderCircle } from "lucide-react";
import { OTPField } from "./otp-field";
import { PasswordField } from "./password-field";
import { authStyles as styles } from "./auth-shell";

function messageFor(status: number) {
  return status === 503 ? "O serviço está temporariamente indisponível. Tente novamente." : "Não foi possível confirmar os dados. Tente novamente.";
}

export function RecoveryStartForm() {
  const router = useRouter();
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(false);
  
  async function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const enrollment = String(new FormData(event.currentTarget).get("enrollment_number") ?? "");
    if (!/^\d{10}$/.test(enrollment)) { setError("Informe os 10 dígitos da matrícula."); return; }
    setLoading(true); setError("");
    try {
      const response = await fetch("/api/auth/password-recovery/start", { method: "POST", headers: { "Content-Type": "application/json" }, body: JSON.stringify({ enrollment_number: enrollment }) });
      if (!response.ok && response.status !== 202) { setError(messageFor(response.status)); return; }
      sessionStorage.setItem("sysap-recovery-enrollment", enrollment);
      router.push("/recuperar-acesso/verificar");
    } catch { setError("Não foi possível iniciar agora. Tente novamente."); } finally { setLoading(false); }
  }
  
  return <form className={styles.form} noValidate onSubmit={submit}>
    <div className={styles.field}><label className={styles.label} htmlFor="recovery-enrollment">Matrícula</label><div className={styles.control}><input id="recovery-enrollment" name="enrollment_number" inputMode="numeric" autoComplete="username" maxLength={10} required onInput={(e) => { e.currentTarget.value = e.currentTarget.value.replace(/\D/g, "").slice(0, 10); }} /></div>{error ? <p className={styles.formError} role="alert">{error}</p> : null}</div>
    <button className={styles.primary} disabled={loading} type="submit">{loading ? <><LoaderCircle className={styles.spinner} size={18} /> Enviando…</> : <>Continuar <ArrowRight size={18} /></>}</button>
  </form>;
}

export function RecoveryOTPForm() {
  const router = useRouter(); 
  const [code, setCode] = useState(""); 
  const [error, setError] = useState(""); 
  const [loading, setLoading] = useState(false);
  
  async function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault(); 
    const enrollment = sessionStorage.getItem("sysap-recovery-enrollment");
    if (!enrollment || code.length !== 6) { setError("Informe o código de seis dígitos."); return; }
    setLoading(true); setError("");
    try {
      const response = await fetch("/api/auth/password-recovery/verify", { method: "POST", headers: { "Content-Type": "application/json" }, body: JSON.stringify({ enrollment_number: enrollment, code }) });
      if (!response.ok) { setError(messageFor(response.status)); return; }
      router.push("/recuperar-acesso/nova-senha");
    } catch { setError("Não foi possível confirmar agora. Tente novamente."); } finally { setLoading(false); }
  }
  
  return <form className={styles.form} onSubmit={submit}><OTPField disabled={loading} onComplete={setCode} /><p className={styles.timer} aria-live="polite">Código enviado para seu e-mail.</p>{error ? <p className={styles.formError} role="alert">{error}</p> : null}<button className={styles.primary} disabled={loading || code.length !== 6} type="submit">{loading ? "Verificando…" : <>Continuar <ArrowRight size={18} /></>}</button></form>;
}

export function RecoveryPasswordForm() {
  const router = useRouter(); const [error, setError] = useState(""); const [loading, setLoading] = useState(false);
  async function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault(); const data = new FormData(event.currentTarget); const password = String(data.get("new_password") ?? ""); const confirmation = String(data.get("password_confirmation") ?? "");
    if (password.length < 15 || password !== confirmation) { setError(password.length < 15 ? "Use pelo menos 15 caracteres." : "As senhas precisam ser iguais."); return; }
    setLoading(true); setError("");
    try { const response = await fetch("/api/auth/password-recovery/complete", { method: "POST", headers: { "Content-Type": "application/json" }, body: JSON.stringify({ password, password_confirmation: confirmation }) }); if (!response.ok) { setError(messageFor(response.status)); return; } sessionStorage.removeItem("sysap-recovery-enrollment"); router.push("/recuperar-acesso/concluida"); } catch { setError("Não foi possível concluir agora. Tente novamente."); } finally { setLoading(false); }
  }
  return <form className={styles.form} noValidate onSubmit={submit}><PasswordField autoComplete="new-password" id="recovery-password" label="Nova senha" name="new_password" placeholder="Crie uma senha segura" /><PasswordField autoComplete="new-password" id="recovery-confirmation" label="Confirmar nova senha" name="password_confirmation" placeholder="Digite novamente" />{error ? <p className={styles.formError} role="alert">{error}</p> : null}<button className={styles.primary} disabled={loading} type="submit">{loading ? "Concluindo…" : <>Redefinir senha <ArrowRight size={18} /></>}</button></form>;
}
