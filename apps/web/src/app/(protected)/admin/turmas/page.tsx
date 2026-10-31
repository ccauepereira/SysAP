import type { Metadata } from "next";
import Link from "next/link";
import { redirect } from "next/navigation";
import { Card, Badge } from "@/features/product/product-ui";
import { CreateTrainingGroupForm } from "@/features/admin/create-training-group-form";
import { getAccessToken } from "@/lib/auth/cookies";
import { requireSession } from "@/lib/auth/session";

export const metadata: Metadata = { title: "Turmas" };

type TrainingGroup = {
  readonly id: string;
  readonly name: string;
};

async function getTrainingGroups(organizationId: string): Promise<
  | { status: "ready"; items: TrainingGroup[] }
  | { status: "auth_required" }
  | { status: "error" }
> {
  try {
    const token = await getAccessToken();
    if (!token || !organizationId) return { status: "auth_required" };

    const apiUrl = process.env.SYSAP_API_BASE_URL || "http://127.0.0.1:8080";
    const response = await fetch(
      `${apiUrl}/v1/organizations/${encodeURIComponent(organizationId)}/training-groups`,
      {
        headers: {
          Authorization: `Bearer ${token}`,
          "X-Organization-ID": organizationId,
        },
        cache: "no-store",
      },
    );

    if (response.status === 401 || response.status === 403) {
      return { status: "auth_required" };
    }
    if (!response.ok) return { status: "error" };

    const data = (await response.json()) as {
      items?: Array<{ id?: unknown; name?: unknown }>;
    };
    if (!Array.isArray(data.items)) return { status: "error" };

    const items: TrainingGroup[] = [];
    for (const item of data.items) {
      if (typeof item.id === "string" && typeof item.name === "string") {
        items.push({ id: item.id, name: item.name });
      }
    }
    return { status: "ready", items };
  } catch {
    return { status: "error" };
  }
}

export default async function TrainingGroupsPage() {
  const user = await requireSession();
  if (user.role === "athlete") redirect("/atleta");
  if (user.role !== "owner" && user.role !== "trainer") redirect("/");

  const groupsResult = await getTrainingGroups(user.organizationId);

  return (
    <div className="product-page">
      <header className="product-header">
        <h1>Turmas</h1>
        <p>Gestão e consulta das turmas ativas da academia.</p>
      </header>

      <div className="product-actions">
        <Link className="product-button product-button-secondary" href="/atletas">
          Voltar para Atletas
        </Link>
        <Link className="product-button product-button-primary" href="/admin/atletas/novo">
          Cadastrar atleta
        </Link>
      </div>

      <section aria-labelledby="create-group-heading">
        <Card>
          <h2 id="create-group-heading">Criar nova turma</h2>
          <p>
            Informe o nome descritivo da turma com categoria e turno (ex.: Sub-15 Manhã, Sub-17 Tarde).
          </p>
          <CreateTrainingGroupForm />
        </Card>
      </section>

      <section aria-labelledby="existing-groups-heading">
        <h2 id="existing-groups-heading" style={{ fontSize: "1.1rem", marginBottom: "0.5rem" }}>
          Turmas cadastradas
        </h2>
        {groupsResult.status === "auth_required" ? (
          <Card>
            <p role="alert">
              Confira sua verificação de identidade em duas etapas para consultar e criar turmas.
            </p>
            <Link className="product-button product-button-secondary" href="/verificar-identidade">
              Confirmar identidade
            </Link>
          </Card>
        ) : groupsResult.status === "error" ? (
          <Card>
            <p role="alert">Não foi possível carregar as turmas. Atualize a página.</p>
          </Card>
        ) : groupsResult.items.length === 0 ? (
          <Card>
            <p>Nenhuma turma cadastrada no momento.</p>
          </Card>
        ) : (
          <div className="product-list">
            {groupsResult.items.map((group) => (
              <Card key={group.id}>
                <div className="product-heading-row">
                  <div>
                    <h2>{group.name}</h2>
                    <p className="product-mono" style={{ fontSize: "0.75rem", marginTop: "0.25rem" }}>
                      ID: {group.id.slice(0, 8)}…
                    </p>
                  </div>
                  <Badge tone="success">Ativa</Badge>
                </div>
                <div className="product-actions" style={{ marginTop: "0.5rem" }}>
                  <Link
                    className="product-button product-button-secondary"
                    href={`/admin/atletas/novo`}
                    style={{ fontSize: "0.75rem", minHeight: "38px", padding: "0.5rem 0.75rem" }}
                  >
                    Cadastrar atleta nesta turma
                  </Link>
                </div>
              </Card>
            ))}
          </div>
        )}
      </section>
    </div>
  );
}
