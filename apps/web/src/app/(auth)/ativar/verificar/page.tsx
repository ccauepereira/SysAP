import type { Metadata } from "next";
import { AuthShell } from "@/features/auth/auth-shell";
import { ActivationOTPForm } from "@/features/auth/activation-forms";

export const metadata: Metadata = { title: "Confirmar ativação" };

export default function ActivationVerificationPage() {
  return (
    <AuthShell
      description="Digite o código temporário de seis dígitos."
      progress={{ current: 2, total: 3 }}
      title="Confirme seu acesso"
    >
      <ActivationOTPForm />
    </AuthShell>
  );
}
