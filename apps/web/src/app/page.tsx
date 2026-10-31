import { AppShell } from "@/components/app-shell/app-shell";
import { RealHome } from "@/features/product/real-pages";
import { requireSession } from "@/lib/auth/session";
import { redirect } from "next/navigation";

export const dynamic = "force-dynamic";

export default async function HomePage() {
  const user = await requireSession();
  if (user.role === "athlete") redirect("/atleta");
  return (
    <AppShell>
      <RealHome />
    </AppShell>
  );
}
