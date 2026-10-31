import { redirect } from "next/navigation";
import { AppShell } from "@/components/app-shell/app-shell";
import { RealUnavailablePage } from "@/features/product/real-pages";
import { requireSession } from "@/lib/auth/session";

export default async function Page() {
  const user = await requireSession();
  if (user.role === "athlete") redirect("/atleta");
  return <AppShell current="Financeiro"><RealUnavailablePage title="Financeiro" /></AppShell>;
}
