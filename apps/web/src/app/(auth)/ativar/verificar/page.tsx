import type { Metadata } from "next";
import { AuthShell } from "@/features/auth/auth-shell";
import { ActivationOTPForm } from "@/features/auth/activation-forms";

export const metadata: Metadata = { title: "Confirmar ativação" };

export default function ActivationVerificationPage() {
  return (
    <AuthShell
      description="Digite o código de seis dígitos enviado por SMS."
      progress={{ current: 2, total: 3 }}
      title="Confirme seu acesso"
    >
      <ActivationOTPForm channel="sms" />
    </AuthShell>
  );
}
