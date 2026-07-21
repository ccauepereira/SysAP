import { Activity, BarChart3, CheckCircle2, Users } from "lucide-react";
import { metrics, type Metric } from "./dashboard-data";

const icons = {
  athletes: Users,
  attendance: CheckCircle2,
  readiness: Activity,
  load: BarChart3,
} satisfies Record<Metric["icon"], typeof Users>;

export function MetricGrid() {
  return (
    <section aria-labelledby="metrics-heading" className="metrics-panel">
      <h2 className="visually-hidden" id="metrics-heading">
        Indicadores demonstrativos
      </h2>
      <div className="metrics-grid">
        {metrics.map((metric) => {
          const Icon = icons[metric.icon];
          return (
            <article className="metric" key={metric.label}>
              <div className="metric-label">
                <Icon aria-hidden="true" size={22} strokeWidth={1.8} />
                <h3>{metric.label}</h3>
              </div>
              <p className="metric-value">{metric.value}</p>
              <p className="metric-comparison">{metric.comparison}</p>
            </article>
          );
        })}
      </div>
    </section>
  );
}
