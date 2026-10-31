import type { ReactNode } from "react";

export function ProductHeader({ title, description }: { title: string; description: string }) {
  return <header className="product-header"><p className="product-demo">Demonstração • dados fictícios</p><h1>{title}</h1><p>{description}</p></header>;
}

export function Card({ children, className = "" }: { children: ReactNode; className?: string }) {
  return <section className={`product-card ${className}`}>{children}</section>;
}

export function Badge({ children, tone = "neutral" }: { children: ReactNode; tone?: "neutral" | "success" | "warning" | "error" }) {
  return <span className={`product-badge product-badge-${tone}`}>{children}</span>;
}

export function UnavailableAction({ children }: { children: ReactNode }) {
  return <button className="product-button product-button-secondary" disabled title="Disponível quando houver suporte no sistema" type="button">{children}</button>;
}

export function ContractNotice({ children }: { children: ReactNode }) {
  return <p className="product-notice">{children}</p>;
}
