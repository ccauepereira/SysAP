import type { Metadata } from "next";
import Link from "next/link";
import { CircleCheck } from "lucide-react";
import { AuthShell, authStyles as styles } from "@/features/auth/auth-shell";

export const metadata: Metadata = { title: "Ativação concluída" };

export default function ActivationCompletePage() {
  return (
    <AuthShell
      description="Sua conta está pronta para o primeiro acesso."
      title="Conta ativada"
    >
      <div className={styles.state}>
        <span className={styles.stateIcon}><CircleCheck aria-hidden="true" size={31} /></span>
        <Link className={styles.primary} href="/login">Ir para o login</Link>
      </div>
    </AuthShell>
  );
}
