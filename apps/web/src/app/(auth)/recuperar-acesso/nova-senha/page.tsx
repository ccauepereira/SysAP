import type { Metadata } from "next";
import { AuthShell, PreviewNotice } from "@/features/auth/auth-shell";
import { PreviewPasswordForm } from "@/features/auth/preview-password-form";

export const metadata: Metadata = { title: "Redefinir senha" };

export default function RecoveryPasswordPage() {
  return (
    <AuthShell
      description="Crie uma nova senha longa e exclusiva para a sua conta."
      progress={{ current: 3, total: 3 }}
      title="Redefina sua senha"
    >
      <PreviewNotice />
      <PreviewPasswordForm
        nextHref="/recuperar-acesso/concluida"
        submitLabel="Visualizar conclusão"
      />
    </AuthShell>
  );
}
