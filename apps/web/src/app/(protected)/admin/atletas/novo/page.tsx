import type { Metadata } from "next";
import Link from "next/link";
import { redirect } from "next/navigation";
import { Card } from "@/features/product/product-ui";
import { AthleteInvitationForm, type TeamOption } from "@/features/admin/athlete-invitation-form";
import { getAccessToken } from "@/lib/auth/cookies";
import { requireSession } from "@/lib/auth/session";

export const metadata: Metadata = { title: "Cadastrar atleta" };

async function getActiveTeams(
  organizationId: string,
): Promise<{ status: "ready"; teams: TeamOption[] } | { status: "auth_required" } | { status: "error" }> {
  try {
    const accessToken = await getAccessToken();
    if (!accessToken || !organizationId) {
      return { status: "auth_required" };
    }
    const apiUrl = process.env.SYSAP_API_BASE_URL || "http://127.0.0.1:8080";
    const response = await fetch(
      `${apiUrl}/v1/organizations/${encodeURIComponent(organizationId)}/training-groups`,
      {
        method: "GET",
        headers: {
          Authorization: `Bearer ${accessToken}`,
          "X-Organization-ID": organizationId,
        },
        cache: "no-store",
      },
    );
    if (response.status === 401 || response.status === 403) {
      return { status: "auth_required" };
    }
    if (!response.ok) {
      return { status: "error" };
    }
    const data = (await response.json()) as {
      items?: Array<{ id?: unknown; name?: unknown }>;
    };
    if (!Array.isArray(data.items)) {
      return { status: "error" };
    }
    const teams: TeamOption[] = [];
    for (const item of data.items) {
      if (typeof item.id !== "string" || typeof item.name !== "string") return { status: "error" };
      teams.push({ id: item.id, name: item.name });
    }
    return { status: "ready", teams };
  } catch {
    return { status: "error" };
  }
}

export default async function NewAthletePage() {
  const user = await requireSession();
  if (user.role !== "owner" && user.role !== "trainer") redirect("/");
  const teams = await getActiveTeams(user.organizationId);

  return (
    <div className="product-page">
      <header className="product-header">
        <h1>Cadastrar atleta</h1>
        <p>Cadastre o atleta presencialmente. A matrícula será gerada de forma permanente pelo sistema.</p>
      </header>

      <div className="product-actions">
        <Link className="product-button product-button-secondary" href="/atletas">
          Voltar para lista de atletas
        </Link>
        <Link className="product-button product-button-secondary" href="/admin/turmas">
          Gerenciar turmas
        </Link>
      </div>

      <Card>
        {teams.status === "auth_required" ? (
          <div>
            <p role="alert">
              Confira sua verificação de identidade em duas etapas para cadastrar atletas.
            </p>
            <div className="product-actions" style={{ marginTop: "1rem" }}>
              <Link className="product-button product-button-primary" href="/verificar-identidade">
                Confirmar identidade
              </Link>
            </div>
          </div>
        ) : teams.status === "error" ? (
          <div>
            <p role="alert">
              Não foi possível consultar as turmas da academia. Atualize a página antes de continuar.
            </p>
          </div>
        ) : (
          <AthleteInvitationForm teams={teams.teams} />
        )}
      </Card>
    </div>
  );
}
