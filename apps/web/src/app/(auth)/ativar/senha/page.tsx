import type { Metadata } from "next";
import { AuthShell } from "@/features/auth/auth-shell";
import { ActivationPasswordForm } from "@/features/auth/activation-forms";

export const metadata: Metadata = { title: "Criar senha" };

export default function ActivationPasswordPage() {
  return (
    <AuthShell
      description="Crie uma senha longa e exclusiva para proteger sua conta."
      progress={{ current: 3, total: 3 }}
      title="Crie sua senha"
    >
      <ActivationPasswordForm />
    </AuthShell>
  );
}
