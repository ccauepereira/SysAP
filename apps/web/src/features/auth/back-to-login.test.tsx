import { render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { BackToLogin } from "./back-to-login";

const path = vi.hoisted(() => ({ current: "/ativar" }));
vi.mock("next/navigation", () => ({ usePathname: () => path.current }));

describe("retorno ao login", () => {
  it.each(["/ativar", "/ativar/verificar", "/recuperar-acesso", "/recuperar-acesso/nova-senha"])("offers login from %s", (current) => {
    path.current = current;
    render(<BackToLogin />);
    expect(screen.getByRole("link", { name: "Voltar ao login" })).toHaveAttribute("href", "/login");
  });
  it("does not add a return link to login itself", () => {
    path.current = "/login";
    render(<BackToLogin />);
    expect(screen.queryByRole("link", { name: "Voltar ao login" })).not.toBeInTheDocument();
  });
});
