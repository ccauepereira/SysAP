import Link from "next/link";
import { Card } from "./product-ui";

export function RealHome() {
  return <div className="product-page"><header className="product-header"><h1>Início</h1><p>Acompanhe sua academia pelo painel.</p></header>
    <Card><h2>Atletas e turmas</h2><p>Crie uma turma e cadastre atletas para começar.</p><Link className="product-button product-button-primary" href="/atletas">Ir para atletas</Link></Card>
  </div>;
}

export function RealAthletes({ canInvite }: { canInvite: boolean }) {
  return <div className="product-page"><header className="product-header"><h1>Atletas</h1><p>Cadastros e situação de acesso.</p></header>
    {canInvite && <Link className="product-button product-button-primary" href="/admin/atletas/novo">Cadastrar atleta ou criar turma</Link>}
    <Card><h2>Consulta indisponível</h2><p>A listagem de atletas ainda não está disponível. Os dados existentes não são exibidos nesta tela.</p></Card>
  </div>;
}

export function RealUnavailablePage({ title }: { title: string }) {
  return <div className="product-page"><header className="product-header"><h1>{title}</h1></header><Card><h2>Consulta indisponível</h2><p>Esta área ainda não está conectada aos registros da academia.</p></Card></div>;
}
