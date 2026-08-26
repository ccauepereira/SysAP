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
};

export function PasswordField({
  autoComplete,
  error,
  id,
  label,
  name,
  placeholder,
  required = true,
}: PasswordFieldProps) {
  const [visible, setVisible] = useState(false);
  const errorId = `${id}-error`;

  return (
    <div className={styles.field}>
      <label className={styles.label} htmlFor={id}>
        {label}
      </label>
      <div className={styles.control}>
        <LockKeyhole aria-hidden="true" size={19} />
        <input
          aria-describedby={error ? errorId : undefined}
          aria-invalid={error ? "true" : undefined}
          autoComplete={autoComplete}
          id={id}
          name={name}
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
    </div>
  );
}
