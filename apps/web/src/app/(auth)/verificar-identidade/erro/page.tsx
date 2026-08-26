import type { Metadata } from "next";
import Link from "next/link";
import { CircleAlert, UsersRound } from "lucide-react";
import { AuthShell, authStyles as styles } from "@/features/auth/auth-shell";

export const metadata: Metadata = { title: "Código não confirmado" };

export default function IdentityVerificationErrorPage() {
  return (
    <AuthShell
      description="O código informado não é válido ou já expirou."
      eyebrow={<><UsersRound aria-hidden="true" size={15} /> Equipe AP</>}
      title="Código não confirmado"
    >
      <div className={styles.state} role="alert">
        <span className={`${styles.stateIcon} ${styles.stateIconDanger}`}>
          <CircleAlert aria-hidden="true" size={31} />
        </span>
        <Link className={styles.primary} href="/verificar-identidade">Tentar novamente</Link>
        <Link className={styles.textLink} href="/estado/mfa-obrigatorio">
          Recuperação administrativa
        </Link>
      </div>
    </AuthShell>
  );
}
