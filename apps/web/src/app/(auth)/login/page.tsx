import type { Metadata } from "next";
import { AuthIntro } from "@/features/auth/intro";
import { AuthShell } from "@/features/auth/auth-shell";
import { LoginForm } from "@/features/auth/login-form";
import { getSession } from "@/lib/auth/session";
import { redirect } from "next/navigation";

export const metadata: Metadata = { title: "Entrar" };
export const dynamic = "force-dynamic";

export default async function LoginPage() {
  const session = await getSession();
  if (session.status === "authenticated") redirect("/");

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
