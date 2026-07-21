const dashboardDateFormatter = new Intl.DateTimeFormat("pt-BR", {
  timeZone: "America/Fortaleza",
  weekday: "long",
  day: "2-digit",
  month: "long",
  year: "numeric",
});

const dashboardMachineDateFormatter = new Intl.DateTimeFormat("en-CA", {
  timeZone: "America/Fortaleza",
  year: "numeric",
  month: "2-digit",
  day: "2-digit",
});

export function formatDashboardDate(date: Date): string {
  const formatted = dashboardDateFormatter.format(date);
  return formatted.charAt(0).toUpperCase() + formatted.slice(1);
}

export function formatDashboardMachineDate(date: Date): string {
  return dashboardMachineDateFormatter.format(date);
}
