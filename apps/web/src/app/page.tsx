import { AppShell } from "@/components/app-shell/app-shell";
import { Dashboard } from "@/features/dashboard/dashboard";
import { requireSession } from "@/lib/auth/session";
import { redirect } from "next/navigation";
import { getSystemStatus } from "@/lib/api/system-status";
import {
  formatDashboardDate,
  formatDashboardMachineDate,
} from "@/lib/date/format-dashboard-date";

export const dynamic = "force-dynamic";

export default async function HomePage() {
  const user = await requireSession();
  if (user.role === "athlete") redirect("/atleta");
  const systemStatus = await getSystemStatus();
  const now = new Date();
  const formattedDate = formatDashboardDate(now);
  const machineDate = formatDashboardMachineDate(now);

  return (
    <AppShell>
      <Dashboard
        formattedDate={formattedDate}
        machineDate={machineDate}
        systemStatus={systemStatus}
      />
    </AppShell>
  );
}
