import type { Metadata } from "next";
import Link from "next/link";
import { CircleCheck, ShieldCheck } from "lucide-react";
import { AuthShell, authStyles as styles } from "@/features/auth/auth-shell";

export const metadata: Metadata = { title: "Senha redefinida" };

export default function RecoveryCompletePage() {
  return (
    <AuthShell
      description="Sua senha foi atualizada e já pode ser utilizada para entrar."
      title="Senha redefinida"
    >
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
