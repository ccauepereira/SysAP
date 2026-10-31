"use client";

import { useRef, useState } from "react";
import { authStyles as styles } from "@/features/auth/auth-shell";

export function ManualActivationCode({ invitationId }: { readonly invitationId: string }) {
  const [value, setValue] = useState<{ enrollment: string; code: string; expiresAt: string } | null>(null);
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(false);
  const inFlight = useRef(false);
  const showUnconfirmed = () => setError("A emissão não foi confirmada. Não solicite outro código agora: a tentativa pode ter sido registrada e um novo código invalidará o anterior. Atualize a tela ou peça a verificação do estado antes de tentar novamente.");
  async function issue() {
    if (inFlight.current) return;
    inFlight.current = true; setError(""); setValue(null); setLoading(true);
    try {
      const response = await fetch(`/api/admin/athlete-invitations/${invitationId}/manual-activation`, { method: "POST", cache: "no-store" });
      if (!response.ok) {
        let errorCode = "";
        try {
          const errorBody: unknown = await response.json();
          if (typeof errorBody === "object" && errorBody !== null && "error" in errorBody && typeof errorBody.error === "string") {
            errorCode = errorBody.error;
          }
        } catch {
          // ignore non-json error response
        }
        if (response.status === 401 || errorCode === "authentication_required") {
          setError("Sua sessão administrativa expirou ou foi encerrada (por exemplo, ao sair em outra aba). Faça login novamente para gerar o código.");
          return;
        }
        if (response.status === 403) {
          setError(
            errorCode === "mfa_required"
              ? "Confirme sua identidade em duas etapas (MFA) para gerar o código de ativação."
              : "Não foi possível emitir o código. Verifique sua permissão, a confirmação em duas etapas (MFA) e se o convite ainda está pendente."
          );
          return;
        }
        if (response.status === 400 || response.status === 422) {
          setError("Não foi possível emitir o código para este convite. Atualize a página e tente novamente.");
          return;
        }
        showUnconfirmed();
        return;
      }
      const result: unknown = await response.json();
      if (typeof result !== "object" || result === null || !("enrollment_number" in result) || typeof result.enrollment_number !== "string" || !/^\d{10}$/.test(result.enrollment_number) || !("code" in result) || typeof result.code !== "string" || !/^\d{6}$/.test(result.code) || !("expires_at" in result) || typeof result.expires_at !== "string" || Number.isNaN(Date.parse(result.expires_at))) throw new Error("invalid response");
      setValue({ enrollment: result.enrollment_number, code: result.code, expiresAt: result.expires_at });
    } catch { showUnconfirmed(); }
    finally { inFlight.current = false; setLoading(false); }
  }
  return <div><button className={styles.secondary} disabled={loading} onClick={issue} type="button">{loading ? "Gerando…" : "Gerar código de ativação manual"}</button>{error ? <p role="alert">{error}</p> : null}{value ? <div role="status"><p>Matrícula: {value.enrollment}</p><p>Código temporário: {value.code}</p><p>Válido até: {new Date(value.expiresAt).toLocaleString("pt-BR", { timeZone: "America/Fortaleza" })}</p><p>Entregue a matrícula e o código ao aluno por um canal seguro. Um novo código invalida o anterior.</p></div> : null}</div>;
}
