import type { ReactNode } from "react";

type FutureActionProps = {
  readonly children: ReactNode;
  readonly className?: string;
  readonly label: string;
};

export function FutureAction({ children, className, label }: FutureActionProps) {
  return (
    <button
      aria-describedby={`${label}-availability`}
      aria-disabled="true"
      className={className}
      disabled
      type="button"
    >
      {children}
      <span className="visually-hidden" id={`${label}-availability`}>
        Em breve. Esta ação estará disponível em uma próxima etapa.
      </span>
    </button>
  );
}
