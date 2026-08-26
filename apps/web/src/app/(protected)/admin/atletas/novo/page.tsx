import type { Metadata } from "next";
import { redirect } from "next/navigation";
import { AuthShell } from "@/features/auth/auth-shell";
import { AthleteInvitationForm } from "@/features/admin/athlete-invitation-form";
import { requireSession } from "@/lib/auth/session";
import { getAccessToken } from "@/lib/auth/cookies";

export const metadata: Metadata = { title: "Cadastrar atleta" };

async function getTeams(organizationId: string) {
  const token = await getAccessToken();
  const apiUrl = process.env.SYSAP_API_BASE_URL || "http://127.0.0.1:8080";
  try {
    const res = await fetch(`${apiUrl}/v1/organizations/${organizationId}/teams`, {
      headers: { Authorization: `Bearer ${token}` },
      cache: "no-store",
    });
    if (res.ok) {
      const data = await res.json();
      // Assume returns an array of teams or object with data array
      return Array.isArray(data) ? data : (data.data || []);
    }
  } catch (error) {
    console.error("Failed to fetch teams:", error);
  }
  return []; // Mock can be handled in the client or returned empty
}

export default async function NewAthletePage() {
  const user = await requireSession();
  if (user.role !== "owner" && user.role !== "trainer") redirect("/");

  const teams = await getTeams(user.organizationId);

  return (
    <AuthShell description="Crie um convite e acompanhe a ativação sem expor códigos ou contatos." title="Cadastrar atleta">
      <AthleteInvitationForm teams={teams} organizationId={user.organizationId} />
    </AuthShell>
  );
}
