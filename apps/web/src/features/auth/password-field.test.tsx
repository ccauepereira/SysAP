import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { PasswordField } from "./password-field";

describe("indicador de força da senha", () => {
  it("informa a estimativa durante a digitação sem alterar o campo", () => {
    render(<PasswordField autoComplete="new-password" id="password" label="Nova senha" name="password" placeholder="Senha" showStrength />);
    const input = screen.getByLabelText("Nova senha");
    expect(input).toHaveAccessibleDescription(/força estimada aparecerá/i);
    fireEvent.change(input, { target: { value: "abc" } });
    expect(input).toHaveAccessibleDescription(/força estimada: fraca/i);
    fireEvent.change(input, { target: { value: "ExemploSeguro1234" } });
    expect(input).toHaveAccessibleDescription(/força estimada: razoável/i);
    fireEvent.change(input, { target: { value: "ExemploSeguro1234!!" } });
    expect(input).toHaveAccessibleDescription(/força estimada: forte/i);
    expect(input).toHaveAttribute("type", "password");
  });
});
