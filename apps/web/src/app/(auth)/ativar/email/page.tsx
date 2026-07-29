import type { Metadata } from "next";
import { AuthShell } from "@/features/auth/auth-shell";
import { ActivationOTPForm } from "@/features/auth/activation-forms";

export const metadata: Metadata = { title: "Confirmar e-mail" };

export default function ActivationEmailPage() {
  return <AuthShell description="Digite o código enviado ao e-mail cadastrado." progress={{ current: 2, total: 3 }} title="Confirme seu e-mail"><ActivationOTPForm /></AuthShell>;
}
