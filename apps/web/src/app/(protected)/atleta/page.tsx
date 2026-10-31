import type { Metadata } from "next";
import Image from "next/image";
import { redirect } from "next/navigation";
import { requireSession } from "@/lib/auth/session";
import { LogoutButton } from "@/features/auth/logout-button";
import { Card, Badge } from "@/features/product/product-ui";
import { getAccessToken } from "@/lib/auth/cookies";

export const metadata: Metadata = { title: "Painel do Atleta" };

export default async function AthletePage() {
  const user = await requireSession();
  if (user.role !== "athlete") redirect("/");

  let data: { name: string; enrollment: string; groups: string[] } | null = null;
  try {
    const token = await getAccessToken();
    if (token) {
      const apiUrl = process.env.SYSAP_API_BASE_URL || "http://127.0.0.1:8080";
      const response = await fetch(`${apiUrl}/v1/me/athlete`, {
        headers: {
          Authorization: `Bearer ${token}`,
          "X-Organization-ID": user.organizationId,
        },
        cache: "no-store",
      },
    );
    if (response.ok) {
      const body: unknown = await response.json();
      if (
        typeof body === "object" &&
        body !== null &&
        "name" in body &&
        typeof body.name === "string" &&
        "enrollment_number" in body &&
        typeof body.enrollment_number === "string" &&
        /^\d{10}$/.test(body.enrollment_number) &&
        "groups" in body &&
        Array.isArray(body.groups) &&
        body.groups.every((group: unknown) => typeof group === "string")
      ) {
        data = {
          name: body.name,
          enrollment: body.enrollment_number,
          groups: body.groups,
        };
      }
    }
  }
} catch {
  /* Do not expose provider or connection details. */
}

  return (
    <div className="product-page" style={{ maxWidth: "640px", minHeight: "100vh" }}>
      <header
        className="product-header"
        style={{
          textAlign: "center",
          paddingBottom: "1rem",
          borderBottom: "1px solid var(--border)",
        }}
      >
        <div style={{ display: "flex", justifyContent: "center", marginBottom: "0.5rem" }}>
          <Image
            alt="Artur Performance"
            height={80}
            priority
            src="/brand/artur-performance-logo.png"
            width={80}
          />
        </div>
        <span className="product-name" style={{ fontSize: "1.25rem" }}>
          SysAP
        </span>
        <h1 style={{ fontSize: "1.5rem" }}>Painel do atleta</h1>
        <p>Acompanhamento de treinos e identidade esportiva.</p>
      </header>

      <main style={{ display: "grid", gap: "1rem", marginTop: "1rem" }}>
        {data ? (
          <>
            <Card>
              <div className="product-heading-row">
                <div>
                  <p className="product-eyebrow">ATLETA</p>
                  <h2>{data.name}</h2>
                </div>
                <Badge tone="success">Conta ativa</Badge>
              </div>

              <div style={{ marginTop: "0.75rem", display: "grid", gap: "0.5rem" }}>
                <div>
                  <span style={{ fontSize: "0.75rem", color: "var(--text-secondary)" }}>
                    Matrícula institucional
                  </span>
                  <p className="product-mono" style={{ fontSize: "1.1rem", fontWeight: 700, color: "var(--brand-gold)" }}>
                    {data.enrollment}
                  </p>
                </div>

                <div style={{ marginTop: "0.25rem" }}>
                  <span style={{ fontSize: "0.75rem", color: "var(--text-secondary)" }}>
                    Turma vinculada
                  </span>
                  <p style={{ fontSize: "0.95rem", fontWeight: 600, color: "var(--text-primary)" }}>
                    {data.groups.length ? `Turma: ${data.groups.join(", ")}` : "Nenhuma turma ativa vinculada."}
                  </p>
                </div>
              </div>
            </Card>

            <Card>
              <h2>Acompanhamento pós-treino</h2>
              <p>
                Os resumos de treino, controle de frequência e dados de prontidão serão disponibilizados aqui após o encerramento das atividades presenciais.
              </p>
            </Card>
          </>
        ) : (
          <Card>
            <p role="alert">
              Não foi possível consultar seus dados. Atualize a página ou procure a comissão técnica da academia.
            </p>
          </Card>
        )}

        <div style={{ display: "flex", justifyContent: "center", marginTop: "1.5rem" }}>
          <LogoutButton />
        </div>
      </main>
    </div>
  );
}
