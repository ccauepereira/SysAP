import type { Metadata } from "next";
import { AuthShell, PreviewNotice } from "@/features/auth/auth-shell";
import { PreviewOTPForm } from "@/features/auth/preview-otp-form";

export const metadata: Metadata = { title: "Código de recuperação" };

export default function RecoveryVerificationPage() {
  return (
    <AuthShell
      description="Digite o código de seis dígitos do canal mascarado selecionado."
      progress={{ current: 2, total: 3 }}
      title="Digite o código"
    >
      <PreviewNotice />
      <PreviewOTPForm nextHref="/recuperar-acesso/nova-senha" recovery />
    </AuthShell>
  );
}
