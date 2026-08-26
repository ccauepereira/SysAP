import type { Metadata } from "next";
import { AuthShell } from "@/features/auth/auth-shell";
import { RecoveryPasswordForm } from "@/features/auth/recovery-forms";

export const metadata: Metadata = { title: "Nova senha" };

export default function RecoveryNewPasswordPage() {
  return (
    <AuthShell
      description="Crie uma nova senha de no mínimo 15 caracteres."
      progress={{ current: 3, total: 3 }}
      title="Nova senha"
    >
      <RecoveryPasswordForm />
    </AuthShell>
  );
}
