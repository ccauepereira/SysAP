import Image from "next/image";
import type { ReactNode } from "react";
import { ShieldCheck } from "lucide-react";
import styles from "./auth-shell.module.css";

type AuthShellProps = {
  readonly children: ReactNode;
  readonly eyebrow?: ReactNode;
  readonly title: string;
  readonly description: string;
  readonly progress?: { readonly current: 1 | 2 | 3; readonly total: 3 };
};

export function AuthShell({
  children,
  description,
  eyebrow,
  progress,
  title,
}: AuthShellProps) {
  return (
    <div className={styles.stage}>
      <header className={styles.brandHeader}>
        <Image
          alt=""
          aria-hidden="true"
          height={80}
          priority
          src="/brand/artur-performance-logo.png"
          width={80}
        />
        <span>Artur Performance</span>
      </header>
      <section className={styles.card}>
        {progress ? <AuthProgress {...progress} /> : null}
        <header className={styles.cardHeader}>
          {eyebrow ? <div className={styles.eyebrow}>{eyebrow}</div> : null}
          <h1>{title}</h1>
          <p>{description}</p>
        </header>
        {children}
        <p className={styles.security}>
          <ShieldCheck aria-hidden="true" size={18} />
          Acesso protegido para a sua jornada com a equipe AP.
        </p>
      </section>
    </div>
  );
}

export function PreviewNotice() {
  return (
    <p className={styles.preview} role="note">
      <ShieldCheck aria-hidden="true" size={17} />
      Prévia visual: nenhuma informação será enviada ou armazenada nesta etapa.
    </p>
  );
}

function AuthProgress({
  current,
  total,
}: Readonly<{ current: 1 | 2 | 3; total: 3 }>) {
  return (
    <div
      aria-label={`Etapa ${current} de ${total}`}
      className={styles.progress}
      role="progressbar"
      aria-valuemax={total}
      aria-valuemin={1}
      aria-valuenow={current}
    >
      <strong>{current} de {total}</strong>
      <div aria-hidden="true" className={styles.progressTrack}>
        <span className={current >= 1 ? styles.done : undefined} />
        <i />
        <span className={current >= 2 ? styles.done : undefined} />
        <i />
        <span className={current >= 3 ? styles.done : undefined} />
      </div>
    </div>
  );
}

export { styles as authStyles };
