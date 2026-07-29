"use client";

import Image from "next/image";
import { authStyles as styles } from "./auth-shell";

export function AuthIntro() {
  return (
    <div aria-hidden="true" className={styles.intro}>
      <Image
        alt=""
        height={240}
        priority
        src="/brand/artur-performance-logo.png"
        width={240}
      />
    </div>
  );
}
