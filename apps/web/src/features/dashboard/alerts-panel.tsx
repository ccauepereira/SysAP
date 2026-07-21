import { Activity, ArrowRight, Bell, CalendarClock } from "lucide-react";
import { alerts } from "./dashboard-data";

const alertIcons = {
  critical: Activity,
  warning: Bell,
  neutral: CalendarClock,
} as const;

export function AlertsPanel() {
  return (
    <section aria-labelledby="alerts-heading" className="panel alerts-panel">
      <div className="section-heading">
        <div>
          <Bell aria-hidden="true" size={19} />
          <h2 id="alerts-heading">Alertas prioritários</h2>
        </div>
        <span aria-disabled="true" className="future-link">
          Ver todos <span className="visually-hidden">— em breve</span>
        </span>
      </div>
      <ul className="alert-list">
        {alerts.map((alert) => {
          const Icon = alertIcons[alert.level];
          return (
            <li className={`alert alert-${alert.level}`} key={alert.id}>
              <Icon aria-hidden="true" size={25} />
              <span className="alert-copy">
                <strong>{alert.title}</strong>
                <small>{alert.guidance}</small>
              </span>
              <ArrowRight aria-hidden="true" size={17} />
            </li>
          );
        })}
      </ul>
    </section>
  );
}
