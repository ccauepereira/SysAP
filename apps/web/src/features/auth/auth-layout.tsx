import type { ReactNode } from "react";
import styles from "./auth-shell.module.css";

export function AuthLayout({ children }: Readonly<{ children: ReactNode }>) {
  return (
    <div className={styles.viewport}>
      <svg
        aria-hidden="true"
        className={styles.tactics}
        preserveAspectRatio="xMidYMid slice"
        viewBox="0 0 1440 900"
      >
        <g fill="none" stroke="currentColor" strokeWidth="2">
          <path d="M-70 150h390v600H-70M320 150v600M-70 450h390" />
          <circle cx="320" cy="450" r="98" />
          <path d="M1115 65h390v770h-390M1115 450h390" />
          <circle cx="1115" cy="450" r="98" />
          <path d="M60 212c120-105 220-80 270 25M1380 705c-132 88-230 52-274-35" strokeDasharray="12 15" />
          <path d="m315 224 14 13 7-18M1121 676l-16-7-1 18" />
          <circle cx="116" cy="680" r="11" />
          <circle cx="1325" cy="188" r="11" />
          <path d="m150 92 18 18m0-18-18 18M1260 785l18 18m0-18-18 18" />
        </g>
      </svg>
      <div aria-hidden="true" className={styles.ambientGlow} />
      <main className={styles.main} id="main-content">
        {children}
      </main>
    </div>
  );
}
