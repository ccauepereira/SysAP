import type { Metadata } from "next";
import { AuthShell } from "@/features/auth/auth-shell";
import { ActivationStartForm } from "@/features/auth/activation-forms";

export const metadata: Metadata = { title: "Ativar conta" };

export default function ActivationPage() {
  return (
    <AuthShell
      description="Informe sua matrícula para receber o primeiro código de confirmação."
      progress={{ current: 1, total: 3 }}
      title="Ative sua conta"
    >
      <ActivationStartForm />
    </AuthShell>
  );
}
