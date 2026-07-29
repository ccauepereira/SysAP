import type { Metadata } from "next";
import { AuthShell, PreviewNotice } from "@/features/auth/auth-shell";
import { PreviewPasswordForm } from "@/features/auth/preview-password-form";

export const metadata: Metadata = { title: "Criar senha" };

export default function ActivationPasswordPage() {
  return (
    <AuthShell
      description="Crie uma senha longa e exclusiva para proteger sua conta."
      progress={{ current: 3, total: 3 }}
      title="Crie sua senha"
    >
      <PreviewNotice />
      <PreviewPasswordForm nextHref="/ativar/concluida" submitLabel="Visualizar conclusão" />
    </AuthShell>
  );
}
