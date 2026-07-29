import type { Metadata } from "next";
import { redirect } from "next/navigation";
import { AuthShell } from "@/features/auth/auth-shell";
import { AthleteInvitationForm } from "@/features/admin/athlete-invitation-form";
import { requireSession } from "@/lib/auth/session";

export const metadata: Metadata = { title: "Cadastrar atleta" };

export default async function NewAthletePage() {
  const user = await requireSession();
  if (user.role !== "owner") redirect("/");
  return <AuthShell description="Crie um convite e acompanhe a ativação sem expor códigos ou contatos." title="Cadastrar atleta"><AthleteInvitationForm /></AuthShell>;
}
