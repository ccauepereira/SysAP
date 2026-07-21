import { CalendarDays, ClipboardCheck, Plus } from "lucide-react";
import { FutureAction } from "@/components/ui/future-action";
import type { SystemStatus } from "@/lib/api/system-status";
import { AlertsPanel } from "./alerts-panel";
import { AthleteRoster } from "./athlete-roster";
import { EvolutionChart } from "./evolution-chart";
import { MetricGrid } from "./metric-grid";
import { NextTraining } from "./next-training";
import { ReadinessDistribution } from "./readiness-distribution";
import { SystemStatusCard } from "./system-status-card";

type DashboardProps = {
  readonly formattedDate: string;
  readonly machineDate: string;
  readonly systemStatus: SystemStatus;
};

export function Dashboard({ formattedDate, machineDate, systemStatus }: DashboardProps) {
  return (
    <div className="dashboard">
      <header className="dashboard-header">
        <div className="dashboard-introduction">
          <span className="demo-badge">Dados demonstrativos</span>
          <h1>Boa tarde, Artur</h1>
          <p>Resumo operacional de hoje</p>
        </div>
        <div className="dashboard-header-actions">
          <p className="dashboard-date">
            <CalendarDays aria-hidden="true" size={18} />
            <time dateTime={machineDate}>{formattedDate}</time>
          </p>
          <FutureAction className="primary-action desktop-action" label="novo-treino">
            <Plus aria-hidden="true" size={21} />
            <span>Novo treino</span>
            <small>Em breve</small>
          </FutureAction>
        </div>
      </header>

      <MetricGrid />

      <FutureAction className="primary-action mobile-action" label="fazer-chamada">
        <ClipboardCheck aria-hidden="true" size={21} />
        <span>Fazer chamada</span>
        <small>Em breve</small>
      </FutureAction>

      <div className="dashboard-grid">
        <div className="dashboard-main-column">
          <AthleteRoster />
          <EvolutionChart />
        </div>
        <div className="dashboard-side-column">
          <AlertsPanel />
          <ReadinessDistribution />
          <NextTraining />
          <SystemStatusCard status={systemStatus} />
        </div>
      </div>
    </div>
  );
}
