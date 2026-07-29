"use client";

import { useState, type FormEvent } from "react";
import { useRouter } from "next/navigation";
import { ArrowRight } from "lucide-react";
import { PasswordField } from "./password-field";
import { authStyles as styles } from "./auth-shell";

type PreviewPasswordFormProps = {
  readonly nextHref: string;
  readonly submitLabel: string;
};

export function PreviewPasswordForm({
  nextHref,
  submitLabel,
}: PreviewPasswordFormProps) {
  const router = useRouter();
  const [errors, setErrors] = useState<{ password?: string; confirmation?: string }>({});

  function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const form = event.currentTarget;
    const data = new FormData(form);
    const password = String(data.get("new_password") ?? "");
    const confirmation = String(data.get("password_confirmation") ?? "");

    if (password.length < 15) {
      setErrors({ password: "Use pelo menos 15 caracteres." });
      (form.elements.namedItem("new_password") as HTMLElement | null)?.focus();
      return;
    }
    if (password !== confirmation) {
      setErrors({ confirmation: "As senhas precisam ser iguais." });
      (form.elements.namedItem("password_confirmation") as HTMLElement | null)?.focus();
      return;
    }

    form.reset();
    setErrors({});
    router.push(nextHref);
  }

  return (
    <form className={styles.form} noValidate onSubmit={submit}>
      <PasswordField
        autoComplete="new-password"
        error={errors.password}
        id="new-password"
        label="Nova senha"
        name="new_password"
        placeholder="Crie uma senha segura"
      />
      <PasswordField
        autoComplete="new-password"
        error={errors.confirmation}
        id="password-confirmation"
        label="Confirmar nova senha"
        name="password_confirmation"
        placeholder="Digite a senha novamente"
      />
      <ul className={styles.requirements}>
        <li>Use uma frase longa, fácil de lembrar e difícil de adivinhar.</li>
        <li>Evite reutilizar uma senha de outro serviço.</li>
      </ul>
      <button className={styles.primary} type="submit">
        {submitLabel}
        <ArrowRight aria-hidden="true" size={18} />
      </button>
    </form>
  );
}
