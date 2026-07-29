"use client";

import { useCallback, useEffect, useState, type FormEvent } from "react";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { ArrowRight, RotateCcw } from "lucide-react";
import { OTPField } from "./otp-field";
import { authStyles as styles } from "./auth-shell";

type PreviewOTPFormProps = {
  readonly nextHref: string;
  readonly recovery?: boolean;
};

export function PreviewOTPForm({ nextHref, recovery = false }: PreviewOTPFormProps) {
  const router = useRouter();
  const [codeComplete, setCodeComplete] = useState(false);
  const [seconds, setSeconds] = useState(300);

  useEffect(() => {
    const timer = window.setInterval(() => {
      setSeconds((current) => Math.max(0, current - 1));
    }, 1_000);
    return () => window.clearInterval(timer);
  }, []);

  const complete = useCallback(() => setCodeComplete(true), []);

  function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (codeComplete) router.push(nextHref);
  }

  const minutes = Math.floor(seconds / 60).toString().padStart(2, "0");
  const remainder = (seconds % 60).toString().padStart(2, "0");

  return (
    <form className={styles.form} onSubmit={submit}>
      <OTPField onComplete={complete} />
      <p aria-live="polite" className={styles.timer}>
        {recovery ? "A prévia do código expira em " : "Reenvio disponível em "}
        <strong>{minutes}:{remainder}</strong>
      </p>
      <button className={styles.primary} disabled={!codeComplete} type="submit">
        Continuar para a prévia
        <ArrowRight aria-hidden="true" size={18} />
      </button>
      <button
        aria-disabled={seconds > 0}
        className={styles.secondary}
        disabled={seconds > 0}
        onClick={() => setSeconds(300)}
        type="button"
      >
        <RotateCcw aria-hidden="true" size={17} />
        Reenviar código
      </button>
      <p className={styles.support}>
        Não recebeu o código? <Link href="/estado/servico-indisponivel">Fale com a equipe AP</Link>.
      </p>
    </form>
  );
}
