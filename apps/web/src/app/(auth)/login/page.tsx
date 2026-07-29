import type { Metadata } from "next";
import { AuthIntro } from "@/features/auth/intro";
import { AuthShell } from "@/features/auth/auth-shell";
import { LoginForm } from "@/features/auth/login-form";

export const metadata: Metadata = { title: "Entrar" };

export default function LoginPage() {
  return (
    <>
      <AuthIntro />
      <AuthShell
        description="Entre com sua matrícula para acompanhar sua evolução."
        title="Bem-vindo de volta."
      >
        <LoginForm />
      </AuthShell>
    </>
  );
}
