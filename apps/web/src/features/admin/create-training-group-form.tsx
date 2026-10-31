"use client";

import { useRef, useState, type FormEvent } from "react";
import { useRouter } from "next/navigation";
import { authStyles as styles } from "@/features/auth/auth-shell";

export function CreateTrainingGroupForm() {
  const router = useRouter();
  const [error, setError] = useState("");
  const [success, setSuccess] = useState("");
  const [loading, setLoading] = useState(false);
  const inFlight = useRef(false);
  async function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (inFlight.current) return;
    const form = event.currentTarget;
    const name = String(new FormData(form).get("name") ?? "").trim();
    setError(""); setSuccess("");
    if (!name || [...name].length > 120) { setError("Informe um nome com até 120 caracteres."); return; }
    inFlight.current = true; setLoading(true);
    try {
      const response = await fetch("/api/admin/training-groups", { method: "POST", headers: { "Content-Type": "application/json" }, body: JSON.stringify({ name }) });
      if (response.status !== 201) {
        setError(response.status === 409 ? "Já existe uma turma com esse nome. Atualize a página para consultar a lista." : response.status === 403 ? "Não foi possível criar a turma. Confira sua permissão e a verificação de identidade." : "Não foi possível confirmar a criação. Atualize a lista antes de tentar novamente.");
        return;
      }
      const result: unknown = await response.json();
      if (typeof result !== "object" || result === null || !("id" in result) || !("name" in result) || typeof result.id !== "string" || typeof result.name !== "string" || result.name !== name) throw new Error("invalid response");
      form.reset(); setSuccess("Turma criada. Ela estará disponível na seleção de turma."); router.refresh();
    } catch { setError("Não foi possível confirmar a criação. Atualize a lista antes de tentar novamente."); }
    finally { inFlight.current = false; setLoading(false); }
  }
  return <form className={styles.form} onSubmit={submit}>
    <div className={styles.field}>
      <label className={styles.label} htmlFor="training-group-name">Nome da turma</label>
      <input
        id="training-group-name"
        name="name"
        maxLength={120}
        placeholder="Ex.: Sub-15 Manhã, Sub-17 Tarde"
        required
        aria-describedby={error ? "training-group-error" : undefined}
      />
    </div>
    {error && <p className={styles.formError} id="training-group-error" role="alert">{error}</p>}
    {success && <p className={styles.success} role="status">{success}</p>}
    <button className={styles.secondary} disabled={loading} type="submit">{loading ? "Criando turma…" : "Criar turma"}</button>
  </form>;
}
