"use client";

import { useCallback, useState, type FormEvent } from "react";
import { useRouter } from "next/navigation";
import { LoaderCircle, ShieldCheck } from "lucide-react";
import { OTPField } from "./otp-field";
import { authStyles as styles } from "./auth-shell";

export function MFAForm() {
  const router = useRouter();
  const [complete, setComplete] = useState(false);
  const [verifying, setVerifying] = useState(false);
  const handleComplete = useCallback(() => setComplete(true), []);

  function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!complete || verifying) return;
    setVerifying(true);
    window.setTimeout(() => router.push("/verificar-identidade/erro"), 550);
  }

  return (
    <form className={styles.form} onSubmit={submit}>
      <OTPField disabled={verifying} label="Código do autenticador" onComplete={handleComplete} />
      <button className={styles.primary} disabled={!complete || verifying} type="submit">
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
