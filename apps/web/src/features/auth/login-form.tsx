"use client";

import { useRef, useState, type FormEvent } from "react";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { KeyRound, LoaderCircle, LogIn, UserRoundPlus } from "lucide-react";
import { PasswordField } from "./password-field";
import { authStyles as styles } from "./auth-shell";

type LoginErrors = {
  readonly enrollment?: string;
  readonly password?: string;
  readonly form?: string;
};

export function LoginForm() {
  const router = useRouter();
  const enrollmentRef = useRef<HTMLInputElement>(null);
  const [errors, setErrors] = useState<LoginErrors>({});
  const [loading, setLoading] = useState(false);
  const [failedAttempts, setFailedAttempts] = useState<{ count: number; enrollment: string }>({ count: 0, enrollment: "" });

  async function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const form = event.currentTarget;
    const data = new FormData(form);
    const enrollment = String(data.get("enrollment_number") ?? "");
    const password = String(data.get("password") ?? "");

    if (!/^\d{10}$/.test(enrollment)) {
      setErrors({ enrollment: "Informe os 10 dígitos da matrícula." });
      enrollmentRef.current?.focus();
      return;
    }
    if (password.length === 0) {
      setErrors({ password: "Informe sua senha." });
      const passwordInput = form.elements.namedItem("password");
      if (passwordInput instanceof HTMLElement) passwordInput.focus();
      return;
    }

    setErrors({});
    setLoading(true);
    try {
      const response = await fetch("/api/auth/login", {
        body: JSON.stringify({ enrollment_number: enrollment, password }),
        headers: { "Content-Type": "application/json" },
        method: "POST",
      });

      if (response.ok) {
        const result = (await response.json()) as { role?: string };
        form.reset();
        router.replace(result.role === "athlete" ? "/atleta" : "/");
        router.refresh();
        return;
      }

      if (response.status === 401) {
        const newCount = failedAttempts.enrollment === enrollment ? failedAttempts.count + 1 : 1;
        setFailedAttempts({ count: newCount, enrollment });
        
        if (newCount === 2) {
          setErrors({ form: "Não foi possível entrar com os dados informados. Por segurança, mais uma tentativa inválida bloqueará temporariamente este acesso." });
        } else {
          setErrors({ form: "Não foi possível entrar com os dados informados." });
        }
      } else {
        setErrors({ form: "O acesso está temporariamente indisponível. Tente novamente." });
      }
    } catch {
      setErrors({ form: "O acesso está temporariamente indisponível. Tente novamente." });
    } finally {
      setLoading(false);
    }
  }

  return (
    <form className={styles.form} noValidate onSubmit={submit}>
      <div className={styles.field}>
        <label className={styles.label} htmlFor="enrollment-number">Matrícula</label>
        <div className={styles.control}>
          <KeyRound aria-hidden="true" size={19} />
          <input
            aria-describedby={errors.enrollment ? "enrollment-error" : undefined}
            aria-invalid={errors.enrollment ? "true" : undefined}
            autoComplete="username"
            id="enrollment-number"
            inputMode="numeric"
            maxLength={10}
            name="enrollment_number"
            onInput={(event) => {
              event.currentTarget.value = event.currentTarget.value.replace(/\D/g, "").slice(0, 10);
            }}
            pattern="[0-9]{10}"
            placeholder="Digite os 10 dígitos"
            ref={enrollmentRef}
            required
          />
        </div>
        {errors.enrollment ? (
          <p className={styles.fieldError} id="enrollment-error">{errors.enrollment}</p>
        ) : null}
      </div>

      <PasswordField
        autoComplete="current-password"
        error={errors.password}
        id="password"
        label="Senha"
        name="password"
        placeholder="Digite sua senha"
      />

      {errors.form ? <p className={styles.formError} role="alert">{errors.form}</p> : null}

      <button className={styles.primary} disabled={loading} type="submit">
        {loading ? (
          <>
            <LoaderCircle aria-hidden="true" className={styles.spinner} size={19} />
            Entrando…
          </>
        ) : (
          <>
            <LogIn aria-hidden="true" size={19} />
            Entrar
          </>
        )}
      </button>

      <nav aria-label="Outras formas de acesso" className={styles.links}>
        <Link className={styles.textLink} href="/ativar">
          <UserRoundPlus aria-hidden="true" size={17} />
          Ativar minha conta
        </Link>
        <Link className={styles.textLink} href="/recuperar-acesso">
          <KeyRound aria-hidden="true" size={17} />
          Esqueci minha senha
        </Link>
      </nav>
    </form>
  );
}
