import type { Metadata } from "next";
import { notFound } from "next/navigation";
import { AuthShell } from "@/features/auth/auth-shell";
import {
  SystemState,
  systemStates,
  type SystemStateKind,
} from "@/features/auth/system-state";

type StatePageProps = {
  readonly params: Promise<{ status: string }>;
};

export const metadata: Metadata = { title: "Estado do acesso" };

export function generateStaticParams() {
  return Object.keys(systemStates).map((status) => ({ status }));
}

export default async function StatePage({ params }: StatePageProps) {
  const { status } = await params;
  if (!(status in systemStates)) notFound();
  const kind = status as SystemStateKind;
  const state = systemStates[kind];

  return (
    <AuthShell description={state.description} title={state.title}>
      <SystemState kind={kind} />
    </AuthShell>
  );
}
