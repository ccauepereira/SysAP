import type { Metadata } from "next";
import Link from "next/link";
import { CircleCheck, ShieldCheck } from "lucide-react";
import { AuthShell, PreviewNotice, authStyles as styles } from "@/features/auth/auth-shell";

export const metadata: Metadata = { title: "Senha redefinida" };

export default function RecoveryCompletePage() {
  return (
    <AuthShell
      description="Esta é a apresentação do estado final. A recuperação real será integrada na 2G.2."
      title="Senha redefinida"
    >
      <PreviewNotice />
      <div className={styles.state}>
        <span className={styles.stateIcon}><CircleCheck aria-hidden="true" size={31} /></span>
        <p className={styles.preview}>
          <ShieldCheck aria-hidden="true" size={18} />
          Suas sessões ativas foram encerradas por segurança.
        </p>
        <Link className={styles.primary} href="/login">Ir para o login</Link>
      </div>
    </AuthShell>
  );
}
