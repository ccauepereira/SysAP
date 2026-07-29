import type { Metadata } from "next";
import { AuthShell } from "@/features/auth/auth-shell";
import { RecoveryStartForm } from "@/features/auth/recovery-forms";

export const metadata: Metadata = { title: "Recuperar acesso" };

export default function RecoveryPage() {
  return (
    <AuthShell
      description="Informe sua matrícula para iniciar a recuperação da conta."
      progress={{ current: 1, total: 3 }}
      title="Recupere seu acesso"
    >
      <RecoveryStartForm />
    </AuthShell>
  );
}
