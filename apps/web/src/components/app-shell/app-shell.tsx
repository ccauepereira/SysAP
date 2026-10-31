import Image from "next/image";
import Link from "next/link";
import { CalendarDays, Home, Users, Wallet } from "lucide-react";
import type { ReactNode } from "react";
import { LogoutButton } from "@/features/auth/logout-button";

const navigation = [
  { label: "Início", icon: Home, href: "/" },
  { label: "Atletas", icon: Users, href: "/atletas" },
  { label: "Treinos", icon: CalendarDays, href: "/treinos" },
  { label: "Financeiro", icon: Wallet, href: "/financeiro" },
] as const;

type AppShellProps = {
  readonly children: ReactNode;
  readonly current?: "Início" | "Atletas" | "Treinos" | "Financeiro";
  readonly demo?: boolean;
};

export function AppShell({ children, current = "Início", demo = false }: AppShellProps) {
  return (
    <div className="app-shell">
      <a className="skip-link" href="#main-content">Pular para o conteúdo</a>
      <aside aria-label="Navegação principal" className="sidebar">
        <div className="sidebar-brand">
          <Image alt="Artur Performance" className="brand-logo" height={108} priority src="/brand/artur-performance-logo.png" width={108} />
          <strong className="product-name">SysAP</strong>
        </div>
        <nav aria-label="Seções do painel">
          <ul className="sidebar-navigation">
            {navigation.map(({ icon: Icon, label, href }) => (
              <li key={label}><Link aria-current={current === label ? "page" : undefined} className="sidebar-navigation-item" href={demo ? `/demonstracao${href === "/" ? "" : href}` : href}><Icon aria-hidden="true" size={20} strokeWidth={1.8} /><span>{label}</span></Link></li>
            ))}
          </ul>
        </nav>
        {!demo ? <div style={{ marginTop: "auto", paddingTop: "1rem" }}><LogoutButton /></div> : null}
      </aside>
      <div className="mobile-brand-row">
        <Image alt="Artur Performance" className="brand-logo brand-logo-mobile" height={72} priority src="/brand/artur-performance-logo.png" width={72} />
        <span className="mobile-product-name">SysAP<small>Painel web</small></span>
        {!demo ? <LogoutButton /> : null}
      </div>
      <main id="main-content">{children}</main>
      <nav aria-label="Navegação inferior" className="bottom-navigation">
        <ul>{navigation.map(({ icon: Icon, label, href }) => (
          <li key={label}><Link aria-current={current === label ? "page" : undefined} className="bottom-navigation-item" href={demo ? `/demonstracao${href === "/" ? "" : href}` : href}><Icon aria-hidden="true" size={22} strokeWidth={1.8} /><span>{label}</span></Link></li>
        ))}</ul>
      </nav>
    </div>
  );
}
