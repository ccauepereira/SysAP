"use client";

import { useRef, useState, type FormEvent } from "react";
import { useRouter } from "next/navigation";
import { ArrowRight, KeyRound } from "lucide-react";
import { authStyles as styles } from "./auth-shell";

export function PreviewEnrollmentForm({ nextHref }: Readonly<{ nextHref: string }>) {
  const router = useRouter();
  const inputRef = useRef<HTMLInputElement>(null);
  const [error, setError] = useState("");

  function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const form = event.currentTarget;
    const enrollment = String(new FormData(form).get("enrollment_number") ?? "");
    if (!/^\d{10}$/.test(enrollment)) {
      setError("Informe os 10 dígitos da matrícula.");
      inputRef.current?.focus();
      return;
    }
    form.reset();
    router.push(nextHref);
  }

  return (
    <form className={styles.form} noValidate onSubmit={submit}>
      <div className={styles.field}>
        <label className={styles.label} htmlFor="preview-enrollment">Matrícula</label>
        <div className={styles.control}>
          <KeyRound aria-hidden="true" size={19} />
          <input
            aria-describedby={error ? "preview-enrollment-error" : undefined}
            aria-invalid={error ? "true" : undefined}
            autoComplete="username"
            id="preview-enrollment"
            inputMode="numeric"
            maxLength={10}
            name="enrollment_number"
            onInput={(event) => {
              event.currentTarget.value = event.currentTarget.value.replace(/\D/g, "").slice(0, 10);
            }}
            pattern="[0-9]{10}"
            placeholder="Digite os 10 dígitos"
            ref={inputRef}
            required
          />
        </div>
        {error ? <p className={styles.fieldError} id="preview-enrollment-error">{error}</p> : null}
      </div>
      <button className={styles.primary} type="submit">
        Continuar para a prévia
        <ArrowRight aria-hidden="true" size={18} />
      </button>
    </form>
  );
}
