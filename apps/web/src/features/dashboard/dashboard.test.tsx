import { render, screen, within } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { AppShell } from "@/components/app-shell/app-shell";
import { Dashboard } from "./dashboard";
import { alerts, athletes } from "./dashboard-data";

const readyStatus = {
  kind: "ready",
  label: "API online · banco pronto",
  description: "Serviços de fundação disponíveis.",
} as const;

function renderDashboard() {
  return render(
    <AppShell>
      <Dashboard
        formattedDate="Quarta-feira, 04 de junho de 2025"
        machineDate="2025-06-04"
        systemStatus={readyStatus}
      />
    </AppShell>,
  );
}

describe("Dashboard", () => {
  it("renders all four demonstration metrics", () => {
    renderDashboard();
    const metricHeading = screen.getByRole("heading", {
      name: "Indicadores demonstrativos",
    });
    const metricSection = metricHeading.closest("section");
    expect(metricSection).not.toBeNull();
    const metricQueries = within(metricSection as HTMLElement);

    expect(metricQueries.getByText("48")).toBeInTheDocument();
    expect(metricQueries.getByText("92%")).toBeInTheDocument();
    expect(metricQueries.getByText("7,6")).toBeInTheDocument();
    expect(metricQueries.getByText("1.245 UA")).toBeInTheDocument();
  });

  it("identifies the content as demonstration data", () => {
    renderDashboard();
    expect(screen.getByText("Dados demonstrativos")).toBeVisible();
    expect(screen.getAllByText(/demonstração/i).length).toBeGreaterThanOrEqual(4);
  });

  it("provides semantic navigation with an active item", () => {
    renderDashboard();
    const navigation = screen.getByRole("navigation", { name: "Navegação inferior" });
    expect(navigation).toBeInTheDocument();
    expect(within(navigation).getByText("Início").parentElement).toHaveAttribute(
      "aria-current",
      "page",
    );
  });

  it("marks future actions as unavailable", () => {
    renderDashboard();
    const newTraining = screen.getByRole("button", { name: /Novo treino/i });
    const attendance = screen.getByRole("button", { name: /Fazer chamada/i });

    expect(newTraining).toBeDisabled();
    expect(newTraining).toHaveAttribute("aria-disabled", "true");
    expect(attendance).toBeDisabled();
  });

  it("renders an accessible desktop table", () => {
    renderDashboard();
    const table = screen.getByRole("table", {
      name: "Atletas demonstrativos e indicadores da semana",
    });

    for (const heading of ["Atleta", "Presença", "Prontidão", "Carga semanal", "Status"]) {
      expect(within(table).getByRole("columnheader", { name: heading })).toBeInTheDocument();
    }
  });

  it("derives the semantic mobile list from the same athlete fixture", () => {
    renderDashboard();
    const list = screen.getByRole("list", {
      name: "Lista mobile de atletas demonstrativos",
    });

    expect(within(list).getAllByRole("listitem")).toHaveLength(athletes.length);
    for (const athlete of athletes) {
      expect(within(list).getByText(athlete.name)).toBeInTheDocument();
    }
  });

  it("uses safe alert language without medical diagnosis", () => {
    renderDashboard();
    const alertsPanel = screen.getByRole("heading", { name: "Alertas prioritários" }).closest("section");
    expect(alertsPanel).not.toBeNull();
    const copy = alertsPanel?.textContent ?? "";

    for (const alert of alerts) expect(copy).toContain(alert.title);
    expect(copy).not.toMatch(/risco de lesão|diagnóstico|prognóstico/i);
  });

  it("exposes an accessible chart description", () => {
    renderDashboard();
    expect(
      screen.getByRole("img", { name: /Evolução demonstrativa da prontidão média/i }),
    ).toBeInTheDocument();
    expect(screen.getByText(/Série demonstrativa de prontidão média:/i)).toBeInTheDocument();
  });

  it("renders the dashboard when the API is unavailable", () => {
    render(
      <Dashboard
        formattedDate="Quarta-feira, 04 de junho de 2025"
        machineDate="2025-06-04"
        systemStatus={{
          kind: "api-unavailable",
          label: "API indisponível",
          description: "Os dados demonstrativos permanecem acessíveis.",
        }}
      />,
    );

    expect(screen.getByRole("heading", { name: "Boa tarde, Artur" })).toBeInTheDocument();
    expect(screen.getByText("API indisponível")).toBeInTheDocument();
    expect(screen.queryByText(/ECONNREFUSED|http:\/\//i)).not.toBeInTheDocument();
  });
});
