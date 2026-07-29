import type { Metadata } from "next";
import { redirect } from "next/navigation";
import { requireSession } from "@/lib/auth/session";
import { authStyles as styles } from "@/features/auth/auth-shell";
import { LogoutButton } from "@/features/auth/logout-button";

export const metadata: Metadata = { title: "Painel do atleta" };

export default async function AthletePage() {
  const user = await requireSession();
  if (user.role !== "athlete") redirect("/");
  return <main className={styles.state}><h1>Painel do atleta</h1><p>Painel inicial — dados demonstrativos até o primeiro treino registrado.</p><p>Bem-vindo, {user.name}.</p><LogoutButton /></main>;
}
