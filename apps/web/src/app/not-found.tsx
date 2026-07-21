import Link from "next/link";

export default function NotFound() {
  return (
    <div className="page-state">
      <span className="page-state-mark" aria-hidden="true" />
      <h1>Página não encontrada</h1>
      <p>Esta área ainda não faz parte da fundação do SysAP.</p>
      <Link className="secondary-action" href="/">
        Voltar ao início
      </Link>
    </div>
  );
}
