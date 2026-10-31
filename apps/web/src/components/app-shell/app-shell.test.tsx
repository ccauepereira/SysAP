import { render, screen, within } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { AppShell } from "./app-shell";

const routes = [
  { current: "Início", href: "/" },
  { current: "Atletas", href: "/atletas" },
  { current: "Treinos", href: "/treinos" },
  { current: "Financeiro", href: "/financeiro" },
] as const;

describe("product navigation", () => {
  it.each(routes)("marks $current in desktop and mobile navigation", ({ current, href }) => {
    render(<AppShell current={current}>Conteúdo</AppShell>);
    for (const name of ["Seções do painel", "Navegação inferior"]) {
      const nav = screen.getByRole("navigation", { name });
      for (const route of routes) {
        const link = within(nav).getByRole("link", { name: route.current });
        expect(link).toHaveAttribute("href", route.href);
        if (route.href === href) expect(link).toHaveAttribute("aria-current", "page");
        else expect(link).not.toHaveAttribute("aria-current");
      }
    }
  });
});
