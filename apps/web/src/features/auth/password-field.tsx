"use client";

import { useState } from "react";
import { Eye, EyeOff, LockKeyhole } from "lucide-react";
import { authStyles as styles } from "./auth-shell";

type PasswordFieldProps = {
  readonly autoComplete: "current-password" | "new-password";
  readonly error?: string | undefined;
  readonly id: string;
  readonly label: string;
  readonly name: string;
  readonly placeholder: string;
  readonly required?: boolean;
  readonly showStrength?: boolean;
};

export function PasswordField({
  autoComplete,
  error,
  id,
  label,
  name,
  placeholder,
  required = true,
  showStrength = false,
}: PasswordFieldProps) {
  const [visible, setVisible] = useState(false);
  const [password, setPassword] = useState("");
  const errorId = `${id}-error`;
  const strengthId = `${id}-strength`;
  const score = Number(password.length >= 15) + Number(/[a-z]/.test(password) && /[A-Z]/.test(password)) + Number((password.match(/[0-9]/g) ?? []).length >= 4) + Number((password.match(/[^\p{L}\p{N}\s]/gu) ?? []).length >= 2);
  const strength = score >= 4 ? "forte" : score >= 2 ? "razoável" : "fraca";
  const describedBy = [error ? errorId : undefined, showStrength ? strengthId : undefined].filter(Boolean).join(" ");

  return (
    <div className={styles.field}>
      <label className={styles.label} htmlFor={id}>
        {label}
      </label>
      <div className={styles.control}>
        <LockKeyhole aria-hidden="true" size={19} />
        <input
          aria-describedby={describedBy || undefined}
          aria-invalid={error ? "true" : undefined}
          autoComplete={autoComplete}
          id={id}
          name={name}
          onChange={showStrength ? (event) => setPassword(event.currentTarget.value) : undefined}
          placeholder={placeholder}
          required={required}
          type={visible ? "text" : "password"}
        />
        <button
          aria-label={visible ? "Ocultar senha" : "Mostrar senha"}
          className={styles.toggle}
          onClick={() => setVisible((current) => !current)}
          type="button"
        >
          {visible ? <EyeOff aria-hidden="true" size={20} /> : <Eye aria-hidden="true" size={20} />}
        </button>
      </div>
      {error ? <p className={styles.fieldError} id={errorId}>{error}</p> : null}
      {showStrength ? <p className={styles.timer} id={strengthId} aria-live="polite">{password ? `Força estimada: ${strength}.` : "A força estimada aparecerá enquanto você digita."} Esta estimativa não garante segurança.</p> : null}
    </div>
  );
}
