import { redirect } from "next/navigation";
import Link from "next/link";
import { AppShell } from "@/components/app-shell/app-shell";
import { RealAthletes } from "@/features/product/real-pages";
import { ManualActivationCode } from "@/features/admin/manual-activation-code";
import { ManualRecoveryCode } from "@/features/admin/manual-recovery-code";
import { Card, Badge } from "@/features/product/product-ui";
import { getAccessToken } from "@/lib/auth/cookies";
import { requireSession } from "@/lib/auth/session";

export default async function Page() {
  const user = await requireSession();
  if (user.role === "athlete") redirect("/atleta");
  if (user.role !== "owner" && user.role !== "trainer") {
    return (
      <AppShell current="Atletas">
        <RealAthletes canInvite={false} />
      </AppShell>
    );
  }

  let items: Array<{
    id: string;
    athleteId: string;
    name: string;
    enrollment: string;
    group: string;
    status: string;
  }> | null = null;

  try {
    const token = await getAccessToken();
    if (token) {
      const apiUrl = process.env.SYSAP_API_BASE_URL || "http://127.0.0.1:8080";
      const response = await fetch(
        `${apiUrl}/v1/organizations/${encodeURIComponent(user.organizationId)}/athlete-invitations`,
        {
          headers: {
            Authorization: `Bearer ${token}`,
            "X-Organization-ID": user.organizationId,
          },
          cache: "no-store",
        },
      );
      if (response.ok) {
        const body: unknown = await response.json();
        if (typeof body === "object" && body !== null && "items" in body && Array.isArray(body.items)) {
          const parsed: Array<{
            id: string;
            athleteId: string;
            name: string;
            enrollment: string;
            group: string;
            status: string;
          }> = [];
          for (const item of body.items) {
            if (
              typeof item !== "object" ||
              item === null ||
              !("invitation_id" in item) ||
              typeof item.invitation_id !== "string" ||
              !("name" in item) ||
              typeof item.name !== "string" ||
              !("enrollment_number" in item) ||
              typeof item.enrollment_number !== "string" ||
              !("group" in item) ||
              typeof item.group !== "string" ||
              !("status" in item) ||
              typeof item.status !== "string"
            ) {
              throw new Error("invalid list");
            }
            const athleteId =
              "athlete_id" in item && typeof item.athlete_id === "string" ? item.athlete_id : item.invitation_id;
            parsed.push({
              id: item.invitation_id,
              athleteId,
              name: item.name,
              enrollment: item.enrollment_number,
              group: item.group,
              status: item.status,
            });
          }
          items = parsed;
        }
      }
    }
  } catch {
    /* Render a neutral availability error. */
  }

  return (
    <AppShell current="Atletas">
      <div className="product-page">
        <header className="product-header">
          <h1>Atletas</h1>
          <p>Cadastros, matrículas e situação de acesso dos atletas.</p>
        </header>

        <div className="product-actions">
          <Link className="product-button product-button-primary" href="/admin/atletas/novo">
            Cadastrar atleta
          </Link>
          <Link className="product-button product-button-secondary" href="/admin/turmas">
            Gerenciar turmas
          </Link>
        </div>

        {items === null ? (
          <Card>
            <p role="alert">
              Não foi possível consultar os atletas. Confira sua confirmação de identidade em duas etapas e tente novamente.
            </p>
            <div className="product-actions" style={{ marginTop: "0.5rem" }}>
              <Link className="product-button product-button-secondary" href="/verificar-identidade">
                Confirmar identidade
              </Link>
            </div>
          </Card>
        ) : items.length === 0 ? (
          <Card>
            <p>Nenhum atleta cadastrado na organização.</p>
          </Card>
        ) : (
          <div className="product-list">
            {items.map((item, index) => (
              <Card key={`${item.id}-${index}`}>
                <div className="product-heading-row">
                  <div>
                    <h2>{item.name}</h2>
                    <p className="product-mono" style={{ fontSize: "0.85rem", marginTop: "0.25rem" }}>
                      Matrícula: {item.enrollment}
                    </p>
                    <p style={{ marginTop: "0.25rem" }}>
                      Turma: {item.group || "Sem turma ativa"}
                    </p>
                  </div>
                  <Badge
                    tone={
                      item.status === "activated"
                        ? "success"
                        : item.status === "pending_activation"
                          ? "warning"
                          : "neutral"
                    }
                  >
                    {item.status === "activated"
                      ? "Ativado"
                      : item.status === "pending_activation"
                        ? "Pendente de ativação"
                        : item.status === "suspended"
                          ? "Suspenso"
                          : item.status === "expired"
                            ? "Convite expirado"
                            : "Acesso encerrado"}
                  </Badge>
                </div>

                <div className="product-actions" style={{ marginTop: "0.75rem" }}>
                  {item.status === "pending_activation" ? (
                    <ManualActivationCode invitationId={item.id} />
                  ) : null}
                  {item.status === "activated" ? (
                    <ManualRecoveryCode athleteId={item.athleteId} />
                  ) : null}
                </div>
              </Card>
            ))}
          </div>
        )}
      </div>
    </AppShell>
  );
}
