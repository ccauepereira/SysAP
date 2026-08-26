import type { Metadata } from "next";
import { AuthShell } from "@/features/auth/auth-shell";
import { RecoveryOTPForm } from "@/features/auth/recovery-forms";

export const metadata: Metadata = { title: "Confirmar acesso" };

export default function RecoveryVerificationPage() {
  return (
    <AuthShell
      description="Digite o código de seis dígitos enviado por e-mail."
      progress={{ current: 2, total: 3 }}
      title="Confirme seu acesso"
    >
      <RecoveryOTPForm />
    </AuthShell>
  );
}
