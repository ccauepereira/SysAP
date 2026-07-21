import { Activity, Database, ServerOff, TriangleAlert } from "lucide-react";
import type { SystemStatus } from "@/lib/api/system-status";

const icons = {
  ready: Activity,
  "database-unavailable": Database,
  "api-unavailable": ServerOff,
  "unexpected-response": TriangleAlert,
} as const;

export function SystemStatusCard({ status }: { readonly status: SystemStatus }) {
  const Icon = icons[status.kind];
  return (
    <aside aria-label="Estado da infraestrutura" className={`system-status system-status-${status.kind}`}>
      <Icon aria-hidden="true" size={18} />
      <span>
        <strong>{status.label}</strong>
        <small>{status.description}</small>
      </span>
    </aside>
  );
}
