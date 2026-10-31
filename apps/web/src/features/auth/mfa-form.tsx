"use client";

import { useCallback, useEffect, useRef, useState, type FormEvent } from "react";
import { useRouter } from "next/navigation";
import { LoaderCircle, ShieldCheck } from "lucide-react";
import { OTPField } from "./otp-field";
import { authStyles as styles } from "./auth-shell";

type MFAState = {
  readonly status: "challenge_ready" | "enroll_required";
  readonly factor_id: string;
  readonly secret?: string | undefined;
};

export function MFAForm() {
  const router = useRouter();
  const [code, setCode] = useState("");
  const [verifying, setVerifying] = useState(false);
  const [error, setError] = useState("");
  const [mfaState, setMfaState] = useState<MFAState | null>(null);
  const initPromise = useRef<Promise<MFAState | null> | null>(null);

  const handleComplete = useCallback((value: string) => {
    setCode(value);
  }, []);

  useEffect(() => {
    let active = true;
    async function loadMFA() {
      try {
        if (typeof fetch !== "function") return;
        if (!initPromise.current) {
          initPromise.current = (async () => {
            const response = await fetch("/api/auth/mfa", {
              method: "GET",
              cache: "no-store",
            });
            if (!response || !response.ok) return null;
            const data = (await response.json()) as {
              status?: "challenge_ready" | "enroll_required";
              factor_id?: string;
              secret?: string;
            };
            if (data.factor_id && (data.status === "challenge_ready" || data.status === "enroll_required")) {
              return {
                status: data.status,
                factor_id: data.factor_id,
                secret: data.secret,
              };
            }
            return null;
          })();
        }
        const loaded = await initPromise.current;
        if (active && loaded) {
          setMfaState(loaded);
        }
      } catch {
        // Keep form interactive even if initial status check fails in tests or offline.
      }
    }
    void loadMFA();
    return () => {
      active = false;
    };
  }, []);

  async function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (code.length !== 6 || verifying) return;
    setVerifying(true);
    setError("");

    try {
      let factorId = mfaState?.factor_id;
      if (!factorId) {
        const statusRes = await fetch("/api/auth/mfa", { method: "GET", cache: "no-store" });
        if (statusRes.ok) {
          const statusData = (await statusRes.json()) as { factor_id?: string };
          factorId = statusData.factor_id;
        }
      }

      if (!factorId) {
        setError("Não foi possível iniciar a verificação em duas etapas. Faça login novamente.");
        return;
      }

      const response = await fetch("/api/auth/mfa/verify", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ factor_id: factorId, code }),
      });

      if (!response.ok) {
        setError(
          response.status === 401
            ? "Código inválido ou expirado. Confira seu autenticador e tente novamente."
            : "Não foi possível confirmar sua identidade agora. Tente novamente.",
        );
        return;
      }

      router.replace("/");
      router.refresh();
    } catch {
      setError("Não foi possível confirmar sua identidade agora. Tente novamente.");
    } finally {
      setVerifying(false);
    }
  }

  return (
    <form className={styles.form} onSubmit={submit}>
      {mfaState?.status === "enroll_required" && mfaState.secret ? (
        <div className={styles.field}>
          <p className={styles.timer}>
            Primeiro acesso administrativo: cadastre a chave TOTP abaixo no seu aplicativo autenticador e digite o código de 6 dígitos gerado.
          </p>
          <code style={{ display: "block", padding: "0.75rem", wordBreak: "break-all", userSelect: "all" }}>
            {mfaState.secret}
          </code>
        </div>
      ) : null}
      <OTPField disabled={verifying} label="Código do autenticador" onComplete={handleComplete} />
      {error ? <p className={styles.formError} role="alert">{error}</p> : null}
      <button className={styles.primary} disabled={code.length !== 6 || verifying} type="submit">
        {verifying ? (
          <>
            <LoaderCircle aria-hidden="true" className={styles.spinner} size={19} />
            Verificando…
          </>
        ) : (
          <>
            <ShieldCheck aria-hidden="true" size={19} />
            Confirmar identidade
          </>
        )}
      </button>
    </form>
  );
}
