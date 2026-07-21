import { Dashboard } from "@/features/dashboard/dashboard";
import { getSystemStatus } from "@/lib/api/system-status";
import {
  formatDashboardDate,
  formatDashboardMachineDate,
} from "@/lib/date/format-dashboard-date";

export const dynamic = "force-dynamic";

export default async function HomePage() {
  const systemStatus = await getSystemStatus();
  const now = new Date();
  const formattedDate = formatDashboardDate(now);
  const machineDate = formatDashboardMachineDate(now);

  return (
    <Dashboard
      formattedDate={formattedDate}
      machineDate={machineDate}
      systemStatus={systemStatus}
    />
  );
}
