"use client";
import Link from "next/link";
import { usePathname } from "next/navigation";

export function BackToLogin() {
  const pathname = usePathname();
  if (!pathname?.startsWith("/ativar") && !pathname?.startsWith("/recuperar-acesso") && !pathname?.startsWith("/verificar-identidade")) return null;
  return <Link className="product-button product-button-secondary" href="/login">Voltar ao login</Link>;
}
