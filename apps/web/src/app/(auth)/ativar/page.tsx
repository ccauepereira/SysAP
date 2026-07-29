import type { Metadata } from "next";
import { AuthShell, PreviewNotice } from "@/features/auth/auth-shell";
import { PreviewEnrollmentForm } from "@/features/auth/preview-enrollment-form";

export const metadata: Metadata = { title: "Ativar conta" };

export default function ActivationPage() {
  return (
    <AuthShell
      description="Informe sua matrícula para iniciar a apresentação do fluxo de ativação."
      progress={{ current: 1, total: 3 }}
      title="Ative sua conta"
    >
      <PreviewNotice />
      <PreviewEnrollmentForm nextHref="/ativar/verificar" />
    </AuthShell>
  );
}
