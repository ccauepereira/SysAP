import Image from "next/image";
import {
  Bell,
  CalendarDays,
  Home,
  Menu,
  Users,
} from "lucide-react";
import type { ReactNode } from "react";

const navigation = [
  { label: "Início", icon: Home, current: true },
  { label: "Atletas", icon: Users, current: false },
  { label: "Treinos", icon: CalendarDays, current: false },
  { label: "Alertas", icon: Bell, current: false },
  { label: "Menu", icon: Menu, current: false },
] as const;

type AppShellProps = {
  readonly children: ReactNode;
};

export function AppShell({ children }: AppShellProps) {
  return (
    <div className="app-shell">
      <a className="skip-link" href="#main-content">
        Pular para o conteúdo
      </a>

      <aside aria-label="Navegação principal" className="sidebar">
        <div className="sidebar-brand">
          <Image
            alt="Artur Performance"
            className="brand-logo"
            height={108}
            priority
            src="/brand/artur-performance-logo.png"
            width={108}
          />
        </div>
        <nav aria-label="Seções do painel">
          <ul className="sidebar-navigation">
            {navigation.map(({ current, icon: Icon, label }) => (
              <li key={label}>
                <span
                  aria-current={current ? "page" : undefined}
                  aria-disabled={current ? undefined : "true"}
                  className="sidebar-navigation-item"
                >
                  <Icon aria-hidden="true" size={20} strokeWidth={1.8} />
                  <span>{label}</span>
                  {!current && <small>Em breve</small>}
                </span>
              </li>
            ))}
          </ul>
        </nav>
        <div className="sidebar-profile" aria-label="Perfil demonstrativo">
          <span className="avatar avatar-small" aria-hidden="true">
            AP
          </span>
          <span>
            <strong>Artur</strong>
            <small>Treinador</small>
          </span>
        </div>
      </aside>

      <div className="mobile-brand-row">
        <Image
          alt="Artur Performance"
          className="brand-logo brand-logo-mobile"
          height={72}
          priority
          src="/brand/artur-performance-logo.png"
          width={72}
        />
        <span className="mobile-brand-name">Artur Performance</span>
        <span aria-label="Alertas demonstrativos" className="mobile-alert-icon">
          <Bell aria-hidden="true" size={24} />
        </span>
        <span className="avatar avatar-small" aria-label="Perfil demonstrativo de Artur">
          AP
        </span>
      </div>

      <main id="main-content">{children}</main>

      <nav aria-label="Navegação inferior" className="bottom-navigation">
        <ul>
          {navigation.map(({ current, icon: Icon, label }) => (
            <li key={label}>
              <span
                aria-current={current ? "page" : undefined}
                aria-disabled={current ? undefined : "true"}
                className="bottom-navigation-item"
              >
                <Icon aria-hidden="true" size={23} strokeWidth={1.8} />
                <span>{label}</span>
              </span>
            </li>
          ))}
        </ul>
      </nav>
    </div>
  );
}
