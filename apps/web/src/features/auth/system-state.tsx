import Link from "next/link";
import {
  CircleAlert,
  Clock3,
  LoaderCircle,
  ServerOff,
  ShieldAlert,
  UserRoundX,
  type LucideIcon,
} from "lucide-react";
import { authStyles as styles } from "./auth-shell";

export type SystemStateKind =
  | "carregando"
  | "sessao-expirada"
  | "conta-suspensa"
  | "servico-indisponivel"
  | "mfa-obrigatorio";

type StateDefinition = {
  readonly action: string;
  readonly description: string;
  readonly href: string;
  readonly icon: LucideIcon;
  readonly title: string;
  readonly danger?: boolean;
};

export const systemStates: Record<SystemStateKind, StateDefinition> = {
  carregando: {
    action: "Voltar ao login",
    description: "Aguarde enquanto verificamos suas informações.",
    href: "/login",
    icon: LoaderCircle,
    title: "Carregando",
  },
  "sessao-expirada": {
    action: "Entrar novamente",
    description: "Sua sessão expirou por inatividade. Entre novamente para continuar.",
    href: "/login",
    icon: Clock3,
    title: "Sessão expirada",
  },
  "conta-suspensa": {
    action: "Voltar ao login",
    description: "Esta conta está suspensa. Fale com a equipe AP para receber orientação.",
    href: "/login",
    icon: UserRoundX,
    title: "Conta suspensa",
    danger: true,
  },
  "servico-indisponivel": {
    action: "Tentar novamente",
    description: "Nosso serviço está temporariamente indisponível. Tente novamente em instantes.",
    href: "/login",
    icon: ServerOff,
    title: "Serviço indisponível",
  },
  "mfa-obrigatorio": {
    action: "Confirmar identidade",
    description: "Confirme sua identidade antes de acessar as ferramentas da equipe AP.",
    href: "/verificar-identidade",
    icon: ShieldAlert,
    title: "Confirmação necessária",
  },
};

export function SystemState({ kind }: Readonly<{ kind: SystemStateKind }>) {
  const state = systemStates[kind];
  const Icon = state?.icon ?? CircleAlert;

  if (!state) return null;

  return (
    <div
      aria-busy={kind === "carregando" ? "true" : undefined}
      className={styles.state}
      role={state.danger ? "alert" : "status"}
    >
      <span
        className={`${styles.stateIcon} ${state.danger ? styles.stateIconDanger : ""}`}
      >
        <Icon
          aria-hidden="true"
          className={kind === "carregando" ? styles.spinner : undefined}
          size={30}
        />
      </span>
      <Link className={styles.primary} href={state.href}>
        {state.action}
      </Link>
    </div>
  );
}
