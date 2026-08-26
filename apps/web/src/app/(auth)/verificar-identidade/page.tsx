import type { Metadata } from "next";
import { UsersRound } from "lucide-react";
import { AuthShell, PreviewNotice } from "@/features/auth/auth-shell";
import { MFAForm } from "@/features/auth/mfa-form";

export const metadata: Metadata = { title: "Confirmar identidade" };

export default function IdentityVerificationPage() {
  return (
    <AuthShell
      description="Digite o código de seis dígitos gerado pelo seu autenticador."
      eyebrow={<><UsersRound aria-hidden="true" size={15} /> Equipe AP</>}
      title="Confirme sua identidade"
    >
      <PreviewNotice />
      <MFAForm />
    </AuthShell>
  );
}
