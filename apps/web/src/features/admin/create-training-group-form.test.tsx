import { fireEvent, render, screen } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { CreateTrainingGroupForm } from "./create-training-group-form";

const refresh = vi.hoisted(() => vi.fn());
vi.mock("next/navigation", () => ({ useRouter: () => ({ refresh }) }));

afterEach(() => { vi.restoreAllMocks(); refresh.mockClear(); });

describe("criação de turma", () => {
  it("only reports success after the API confirms creation", async () => {
    vi.spyOn(global, "fetch").mockResolvedValueOnce(new Response(JSON.stringify({ id: "10000000-0000-4000-8000-000000000001", name: "Turma fictícia" }), { status: 201 }));
    render(<CreateTrainingGroupForm />);
    fireEvent.change(screen.getByLabelText("Nome da turma"), { target: { value: "Turma fictícia" } });
    const form = screen.getByRole("button", { name: "Criar turma" }).closest("form");
    if (!form) throw new Error("Missing training group form");
    fireEvent.submit(form);
    expect(await screen.findByRole("status")).toHaveTextContent("Turma criada");
    expect(refresh).toHaveBeenCalledOnce();
  });

  it("does not report success for a conflict", async () => {
    vi.spyOn(global, "fetch").mockResolvedValueOnce(new Response(null, { status: 409 }));
    render(<CreateTrainingGroupForm />);
    fireEvent.change(screen.getByLabelText("Nome da turma"), { target: { value: "Turma fictícia" } });
    const form = screen.getByRole("button", { name: "Criar turma" }).closest("form");
    if (!form) throw new Error("Missing training group form");
    fireEvent.submit(form);
    expect(await screen.findByRole("alert")).toHaveTextContent("Já existe uma turma");
    expect(screen.queryByRole("status")).not.toBeInTheDocument();
    expect(refresh).not.toHaveBeenCalled();
  });
});
