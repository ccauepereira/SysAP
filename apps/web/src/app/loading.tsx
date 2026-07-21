export default function Loading() {
  return (
    <div aria-busy="true" aria-live="polite" className="page-state">
      <span className="page-state-mark" aria-hidden="true" />
      <h1>Preparando o painel</h1>
      <p>Carregando a fundação demonstrativa do SysAP.</p>
    </div>
  );
}
