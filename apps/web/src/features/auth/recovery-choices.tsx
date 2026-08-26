import Link from "next/link";
import { ChevronRight, Mail, Smartphone } from "lucide-react";
import { authStyles as styles } from "./auth-shell";

export function RecoveryChoices() {
  return (
    <div className={styles.choices}>
      <Link className={styles.choice} href="/recuperar-acesso/verificar">
        <Smartphone aria-hidden="true" size={19} />
        <span>Celular cadastrado •••• ••42</span>
        <ChevronRight aria-hidden="true" size={18} />
      </Link>
      <Link className={styles.choice} href="/recuperar-acesso/verificar">
        <Mail aria-hidden="true" size={19} />
        <span>E-mail cadastrado c•••@•••.com</span>
        <ChevronRight aria-hidden="true" size={18} />
      </Link>
      <p className={styles.support}>
        Estas opções são mascaradas e ilustrativas. Nenhuma conta é confirmada nesta prévia.
      </p>
    </div>
  );
}
