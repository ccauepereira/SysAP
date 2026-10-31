import { redirect } from "next/navigation";
import Link from "next/link";
import { AppShell } from "@/components/app-shell/app-shell";
import { Card } from "@/features/product/product-ui";
import { requireSession } from "@/lib/auth/session";

export default async function Page() {
  const user = await requireSession();
  if (user.role === "athlete") redirect("/atleta");

  return (
    <AppShell current="Treinos">
      <div className="product-page">
        <header className="product-header">
          <h1>Treinos</h1>
          <p>Sessões de treino, chamadas de presença e turmas.</p>
        </header>

        <div className="product-actions">
          <Link className="product-button product-button-primary" href="/admin/turmas">
            Gerenciar turmas
          </Link>
          <Link className="product-button product-button-secondary" href="/atletas">
            Ver atletas
          </Link>
        </div>

        <Card>
          <h2>Gestão de Turmas</h2>
          <p>
            Consulte as turmas ativas da academia (Sub-15, Sub-17) ou cadastre novas turmas para organizar seus atletas.
          </p>
          <div className="product-actions" style={{ marginTop: "0.5rem" }}>
            <Link className="product-button product-button-primary" href="/admin/turmas">
              Acessar turmas
            </Link>
          </div>
        </Card>
      </div>
    </AppShell>
  );
}
