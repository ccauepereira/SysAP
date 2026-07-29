import type { Metadata } from "next";
import Link from "next/link";
import { CircleCheck } from "lucide-react";
import { AuthShell, PreviewNotice, authStyles as styles } from "@/features/auth/auth-shell";

export const metadata: Metadata = { title: "Ativação concluída" };

export default function ActivationCompletePage() {
  return (
    <AuthShell
      description="Esta é a apresentação do estado final. A ativação real será integrada na 2G.2."
      title="Conta ativada"
    >
      <PreviewNotice />
      <div className={styles.state}>
        <span className={styles.stateIcon}><CircleCheck aria-hidden="true" size={31} /></span>
        <Link className={styles.primary} href="/login">Ir para o login</Link>
      </div>
    </AuthShell>
  );
}
