import type { Metadata } from "next";
import { AuthShell, PreviewNotice } from "@/features/auth/auth-shell";
import { PreviewOTPForm } from "@/features/auth/preview-otp-form";

export const metadata: Metadata = { title: "Confirmar ativação" };

export default function ActivationVerificationPage() {
  return (
    <AuthShell
      description="Digite o código de seis dígitos enviado ao canal cadastrado •••• ••42."
      progress={{ current: 2, total: 3 }}
      title="Confirme seu acesso"
    >
      <PreviewNotice />
      <PreviewOTPForm nextHref="/ativar/senha" />
    </AuthShell>
  );
}
