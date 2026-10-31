import type { ReactNode } from "react";
import { AuthLayout } from "@/features/auth/auth-layout";
import { BackToLogin } from "@/features/auth/back-to-login";

export default function AuthenticationLayout({
  children,
}: Readonly<{ children: ReactNode }>) {
  return (
    <>
      <a className="skip-link" href="#main-content">
        Pular para o conteúdo
      </a>
      <AuthLayout>{children}<BackToLogin /></AuthLayout>
    </>
  );
}
