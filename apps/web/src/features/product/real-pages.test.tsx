import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { RealAthletes, RealHome, RealUnavailablePage } from "./real-pages";

describe("áreas reais sem dados demonstrativos", () => {
  it("offers the owner a path to create a group or athlete without inventing a list", () => {
    render(<RealAthletes canInvite />);
    expect(screen.getByRole("link", { name: /Cadastrar atleta ou criar turma/ })).toHaveAttribute("href", "/admin/atletas/novo");
    expect(screen.getByText(/listagem de atletas ainda não está disponível/i)).toBeVisible();
    expect(screen.queryByText(/Demo/)).not.toBeInTheDocument();
  });
  it("does not offer administrative creation to other roles", () => {
    render(<RealAthletes canInvite={false} />);
    expect(screen.queryByRole("link", { name: /Cadastrar atleta/ })).not.toBeInTheDocument();
  });
  it("does not show invented totals or records in the real dashboard", () => {
    render(<RealHome />);
    expect(screen.queryByText(/Demo|R\$/)).not.toBeInTheDocument();
  });
  it("shows an unavailable state instead of sample financial records", () => {
    render(<RealUnavailablePage title="Financeiro" />);
    expect(screen.getByText(/ainda não está conectada/)).toBeVisible();
    expect(screen.queryByText(/Demo|R\$/)).not.toBeInTheDocument();
  });
});
