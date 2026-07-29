import type { ReactNode } from "react";
import { AppShell } from "@/components/app-shell/app-shell";
import { requireSession } from "@/lib/auth/session";

export default async function ProtectedLayout({
  children,
}: Readonly<{ children: ReactNode }>) {
  const user = await requireSession();
  if (user.role === "athlete") return <>{children}</>;
  return <AppShell>{children}</AppShell>;
}
