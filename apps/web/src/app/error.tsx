"use client";

import { useEffect } from "react";

export default function ErrorPage({ reset }: { readonly reset: () => void }) {
  useEffect(() => {
    // Technical details remain intentionally outside the interface.
  }, []);

  return (
    <div className="page-state" role="alert">
      <span className="page-state-mark page-state-mark-error" aria-hidden="true" />
      <h1>Não foi possível abrir o painel</h1>
      <p>Nenhum dado foi perdido. Tente carregar a página novamente.</p>
      <button className="secondary-action" onClick={reset} type="button">
        Tentar novamente
      </button>
    </div>
  );
}
