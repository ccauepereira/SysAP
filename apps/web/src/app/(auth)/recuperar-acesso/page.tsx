import type { Metadata } from "next";
import { AuthShell, PreviewNotice } from "@/features/auth/auth-shell";
import { RecoveryChoices } from "@/features/auth/recovery-choices";

export const metadata: Metadata = { title: "Recuperar acesso" };

export default function RecoveryPage() {
  return (
    <AuthShell
      description="Escolha como deseja visualizar o recebimento do código de recuperação."
      progress={{ current: 1, total: 3 }}
      title="Recupere seu acesso"
    >
      <PreviewNotice />
      <RecoveryChoices />
    </AuthShell>
  );
}
