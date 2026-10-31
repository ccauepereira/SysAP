"use client";

import { useRef, useState } from "react";
import { authStyles as styles } from "@/features/auth/auth-shell";

export function ManualRecoveryCode({ athleteId }: { readonly athleteId: string }) {
  const [value, setValue] = useState<{ enrollment: string; code: string; expiresAt: string } | null>(null);
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(false);
  const inFlight = useRef(false);

  async function issue() {
    if (inFlight.current) return;
    inFlight.current = true;
    setError("");
    setValue(null);
    setLoading(true);
    try {
      const response = await fetch(`/api/admin/athletes/${athleteId}/manual-recovery`, {
        method: "POST",
        cache: "no-store",
      });
      if (!response.ok) throw new Error("issuance failed");
      const result: unknown = await response.json();
      if (
        typeof result !== "object" ||
        result === null ||
        !("enrollment_number" in result) ||
        typeof result.enrollment_number !== "string" ||
        !/^\d{10}$/.test(result.enrollment_number) ||
        !("code" in result) ||
        typeof result.code !== "string" ||
        !/^\d{6}$/.test(result.code) ||
        !("expires_at" in result) ||
        typeof result.expires_at !== "string" ||
        Number.isNaN(Date.parse(result.expires_at))
      ) {
        throw new Error("invalid response");
      }
      setValue({ enrollment: result.enrollment_number, code: result.code, expiresAt: result.expires_at });
    } catch {
      setError("Não foi possível confirmar a emissão. Gere outro código; qualquer anterior será invalidado.");
    } finally {
      inFlight.current = false;
      setLoading(false);
    }
  }

  return (
    <div style={{ marginTop: "0.5rem" }}>
      <button className={styles.secondary} disabled={loading} onClick={issue} type="button">
        {loading ? "Gerando…" : "Iniciar recuperação presencial"}
      </button>
      {error ? <p role="alert">{error}</p> : null}
      {value ? (
        <div role="status" style={{ marginTop: "0.5rem" }}>
          <p><strong>Matrícula:</strong> {value.enrollment}</p>
          <p><strong>Código de recuperação:</strong> {value.code}</p>
          <p><strong>Válido até:</strong> {new Date(value.expiresAt).toLocaleString("pt-BR", { timeZone: "America/Fortaleza" })}</p>
          <p>Entregue a matrícula e o código de recuperação presencialmente ao atleta. Este código é de uso único e invalida qualquer recuperação anterior.</p>
        </div>
      ) : null}
    </div>
  );
}
