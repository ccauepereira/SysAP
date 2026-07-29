"use client";

import {
  useEffect,
  useRef,
  useState,
  type ClipboardEvent,
  type KeyboardEvent,
} from "react";
import { authStyles as styles } from "./auth-shell";

type OTPFieldProps = {
  readonly disabled?: boolean;
  readonly label?: string;
  readonly name?: string;
  readonly onComplete?: (code: string) => void;
};

export function OTPField({
  disabled = false,
  label = "Código de seis dígitos",
  name = "code",
  onComplete,
}: OTPFieldProps) {
  const [digits, setDigits] = useState(["", "", "", "", "", ""]);
  const inputs = useRef<Array<HTMLInputElement | null>>([]);

  useEffect(() => {
    if (digits.every(Boolean)) onComplete?.(digits.join(""));
  }, [digits, onComplete]);

  function update(index: number, value: string) {
    const digit = value.replace(/\D/g, "").slice(-1);
    setDigits((current) => current.map((item, position) => position === index ? digit : item));
    if (digit && index < 5) inputs.current[index + 1]?.focus();
  }

  function keyDown(index: number, event: KeyboardEvent<HTMLInputElement>) {
    if (event.key === "Backspace" && !digits[index] && index > 0) {
      inputs.current[index - 1]?.focus();
    }
    if (event.key === "ArrowLeft" && index > 0) inputs.current[index - 1]?.focus();
    if (event.key === "ArrowRight" && index < 5) inputs.current[index + 1]?.focus();
  }

  function paste(event: ClipboardEvent<HTMLInputElement>) {
    const pasted = event.clipboardData.getData("text").replace(/\D/g, "").slice(0, 6);
    if (!pasted) return;
    event.preventDefault();
    const next = Array.from({ length: 6 }, (_, index) => pasted[index] ?? "");
    setDigits(next);
    inputs.current[Math.min(pasted.length, 6) - 1]?.focus();
  }

  return (
    <fieldset className={styles.otp}>
      <legend className="visually-hidden">{label}</legend>
      {digits.map((digit, index) => (
        <input
          aria-label={`Dígito ${index + 1} de 6`}
          autoComplete={index === 0 ? "one-time-code" : "off"}
          disabled={disabled}
          inputMode="numeric"
          key={index}
          maxLength={1}
          name={`${name}-${index + 1}`}
          onChange={(event) => update(index, event.target.value)}
          onKeyDown={(event) => keyDown(index, event)}
          onPaste={paste}
          ref={(element) => {
            inputs.current[index] = element;
          }}
          type="text"
          value={digit}
        />
      ))}
    </fieldset>
  );
}
