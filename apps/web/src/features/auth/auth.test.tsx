import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import ActivationPage from "@/app/(auth)/ativar/page";
import ActivationVerificationPage from "@/app/(auth)/ativar/verificar/page";
import RecoveryPage from "@/app/(auth)/recuperar-acesso/page";
import RecoveryVerificationPage from "@/app/(auth)/recuperar-acesso/verificar/page";
import IdentityVerificationPage from "@/app/(auth)/verificar-identidade/page";
import LoginPage from "@/app/(auth)/login/page";
import { getSession } from "@/lib/auth/session";
import { AuthShell } from "./auth-shell";
import { LoginForm } from "./login-form";
import { OTPField } from "./otp-field";
import { PasswordField } from "./password-field";
import {
  SystemState,
  systemStates,
  type SystemStateKind,
} from "./system-state";

const { push, replace, refresh, redirect } = vi.hoisted(() => ({
  push: vi.fn(),
  replace: vi.fn(),
  refresh: vi.fn(),
  redirect: vi.fn(),
}));

vi.mock("next/navigation", async (importOriginal) => {
  const actual = await importOriginal<typeof import("next/navigation")>();
  return {
    ...actual,
    redirect,
    useRouter: () => ({ push, refresh, replace }),
  };
});

vi.mock("@/lib/auth/session", () => ({
  getSession: vi.fn(),
}));

beforeEach(() => {
  push.mockReset();
  replace.mockReset();
  refresh.mockReset();
  redirect.mockReset();
  vi.restoreAllMocks();
  vi.mocked(getSession).mockClear();
  vi.mocked(getSession).mockResolvedValue({ status: "unauthenticated", reason: "no_session" });
});

describe("authentication routes", () => {
  it.each([
    [ActivationPage, "Ative sua conta"],
    [ActivationVerificationPage, "Confirme seu acesso"],
    [RecoveryPage, "Recupere seu acesso"],
    [RecoveryVerificationPage, "Confirme seu acesso"],
    [IdentityVerificationPage, "Confirme sua identidade"],
  ])("renders one accessible heading for each flow", (Page, heading) => {
    const { unmount } = render(<Page />);
    expect(screen.getByRole("heading", { level: 1, name: heading })).toBeVisible();
    expect(screen.getAllByRole("heading", { level: 1 })).toHaveLength(1);
    unmount();
  });

  it("renders login when the server has no valid session", async () => {
    render(await LoginPage());
    expect(screen.getByRole("heading", { level: 1, name: "Bem-vindo de volta." })).toBeVisible();
  });

  it("redirects login to the dashboard for an authenticated session", async () => {
    vi.mocked(getSession).mockResolvedValue({
      status: "authenticated",
      user: { id: "profile", name: "Example", role: "athlete", organizationId: "organization" },
    });
    await LoginPage();
    expect(redirect).toHaveBeenCalledWith("/");
  });

  it("links login to activation and recovery", () => {
    return LoginPage().then((page) => {
      render(page);
    expect(screen.getByRole("link", { name: /Ativar minha conta/i })).toHaveAttribute(
      "href",
      "/ativar",
    );
    expect(screen.getByRole("link", { name: /Esqueci minha senha/i })).toHaveAttribute(
      "href",
      "/recuperar-acesso",
    );
    });
  });

  it("does not leak PII in the recovery page", () => {
    const { container } = render(<RecoveryPage />);
    expect(container.textContent).not.toMatch(/\b\d{8,}\b|[\w.+-]+@[\w.-]+\.[a-z]{2,}/i);
  });

  it("keeps public activation routes independent from session redirects", () => {
    render(<ActivationPage />);
    expect(getSession).not.toHaveBeenCalled();
  });
});

describe("login form", () => {
  it("filters enrollment input and focuses the first invalid field", () => {
    render(<LoginForm />);
    const enrollment = screen.getByLabelText("Matrícula");
    fireEvent.input(enrollment, { target: { value: "12ab345678901" } });
    expect(enrollment).toHaveValue("1234567890");

    fireEvent.input(enrollment, { target: { value: "123" } });
    fireEvent.submit(screen.getByRole("button", { name: "Entrar" }).closest("form")!);
    expect(screen.getByText("Informe os 10 dígitos da matrícula.")).toBeVisible();
    expect(enrollment).toHaveFocus();
  });

  it("shows and hides the password without changing its value", () => {
    render(
      <PasswordField
        autoComplete="current-password"
        id="test-password"
        label="Senha"
        name="password"
        placeholder="Digite sua senha"
      />,
    );
    const password = screen.getByLabelText("Senha");
    fireEvent.change(password, { target: { value: "frase segura fictícia" } });
    expect(password).toHaveAttribute("type", "password");
    fireEvent.click(screen.getByRole("button", { name: "Mostrar senha" }));
    expect(password).toHaveAttribute("type", "text");
    expect(password).toHaveValue("frase segura fictícia");
    fireEvent.click(screen.getByRole("button", { name: "Ocultar senha" }));
    expect(password).toHaveAttribute("type", "password");
  });

  it("shows a neutral error without leaking the upstream response", async () => {
    vi.spyOn(global, "fetch").mockResolvedValueOnce(
      new Response(JSON.stringify({
        error: {
          code: "invalid_credentials",
          message: "private upstream detail",
          request_id: "private-request-id",
        },
      }), { status: 401 }),
    );
    const { container } = render(<LoginForm />);
    fireEvent.change(screen.getByLabelText("Matrícula"), { target: { value: "2026000001" } });
    fireEvent.change(screen.getByLabelText("Senha"), { target: { value: "senha fictícia" } });
    fireEvent.submit(screen.getByRole("button", { name: "Entrar" }).closest("form")!);

    expect(await screen.findByRole("alert")).toHaveTextContent(
      "Não foi possível entrar com os dados informados.",
    );
    expect(container.textContent).not.toMatch(/private upstream|private-request-id|invalid_credentials/);
  });

  it("locks the button while the BFF request is pending", async () => {
    let resolveRequest: ((response: Response) => void) | undefined;
    vi.spyOn(global, "fetch").mockImplementationOnce(
      () => new Promise<Response>((resolve) => { resolveRequest = resolve; }),
    );
    render(<LoginForm />);
    fireEvent.change(screen.getByLabelText("Matrícula"), { target: { value: "2026000001" } });
    fireEvent.change(screen.getByLabelText("Senha"), { target: { value: "senha fictícia" } });
    fireEvent.submit(screen.getByRole("button", { name: "Entrar" }).closest("form")!);

    expect(screen.getByRole("button", { name: /Entrando/ })).toBeDisabled();
    resolveRequest?.(new Response("{}", { status: 401 }));
    await waitFor(() => expect(screen.getByRole("button", { name: "Entrar" })).toBeEnabled());
  });
});

describe("OTP field", () => {
  it("advances focus, supports paste and moves back on backspace", () => {
    const completed = vi.fn();
    render(<OTPField onComplete={completed} />);
    const first = screen.getByLabelText("Dígito 1 de 6");
    const second = screen.getByLabelText("Dígito 2 de 6");

    fireEvent.change(first, { target: { value: "1" } });
    expect(second).toHaveFocus();
    fireEvent.keyDown(second, { key: "Backspace" });
    expect(first).toHaveFocus();

    fireEvent.paste(first, {
      clipboardData: { getData: () => "654321" },
    });
    expect(screen.getByLabelText("Dígito 6 de 6")).toHaveValue("1");
    expect(completed).toHaveBeenCalledWith("654321");
  });

  it("never writes secret values to browser storage", () => {
    const localStorageWrite = vi.spyOn(Storage.prototype, "setItem");
    render(<OTPField />);
    fireEvent.paste(screen.getByLabelText("Dígito 1 de 6"), {
      clipboardData: { getData: () => "123456" },
    });
    expect(localStorageWrite).not.toHaveBeenCalled();
  });
});

describe("safe system states", () => {
  it.each(Object.keys(systemStates) as SystemStateKind[])(
    "renders %s without internal details",
    (kind) => {
      const state = systemStates[kind];
      const { container, unmount } = render(
        <AuthShell description={state.description} title={state.title}>
          <SystemState kind={kind} />
        </AuthShell>,
      );
      expect(screen.getByRole("heading", { name: state.title })).toBeVisible();
      expect(container.textContent).not.toMatch(/SQLSTATE|https?:\/\/|request[_ -]?id|token/i);
      unmount();
    },
  );
});
