export type Metric = {
  readonly label: string;
  readonly value: string;
  readonly comparison: string;
  readonly icon: "athletes" | "attendance" | "readiness" | "load";
};

export type AthleteStatus = "Ótimo" | "Bom" | "Atenção" | "Revisar";

export type DemonstrationAthlete = {
  readonly id: string;
  readonly initials: string;
  readonly name: string;
  readonly position: string;
  readonly attendance: string;
  readonly readiness: string;
  readonly weeklyLoad: string;
  readonly status: AthleteStatus;
};

export type DemonstrationAlert = {
  readonly id: string;
  readonly title: string;
  readonly guidance: string;
  readonly level: "critical" | "warning" | "neutral";
};

export const metrics: readonly Metric[] = [
  {
    label: "Atletas ativos",
    value: "48",
    comparison: "+8% vs. semana anterior — demonstração",
    icon: "athletes",
  },
  {
    label: "Presenças",
    value: "92%",
    comparison: "+5% vs. semana anterior — demonstração",
    icon: "attendance",
  },
  {
    label: "Prontidão média",
    value: "7,6",
    comparison: "+0,8 vs. semana anterior — demonstração",
    icon: "readiness",
  },
  {
    label: "Carga semanal",
    value: "1.245 UA",
    comparison: "+12% vs. semana anterior — demonstração",
    icon: "load",
  },
];

export const athletes: readonly DemonstrationAthlete[] = [
  {
    id: "demo-a",
    initials: "AE",
    name: "Atleta Exemplo A",
    position: "Meia",
    attendance: "100%",
    readiness: "8,5",
    weeklyLoad: "980 UA",
    status: "Ótimo",
  },
  {
    id: "demo-b",
    initials: "BE",
    name: "Atleta Exemplo B",
    position: "Atacante",
    attendance: "100%",
    readiness: "7,2",
    weeklyLoad: "875 UA",
    status: "Bom",
  },
  {
    id: "demo-c",
    initials: "CE",
    name: "Atleta Exemplo C",
    position: "Volante",
    attendance: "92%",
    readiness: "6,8",
    weeklyLoad: "810 UA",
    status: "Atenção",
  },
  {
    id: "demo-d",
    initials: "DE",
    name: "Atleta Exemplo D",
    position: "Zagueiro",
    attendance: "88%",
    readiness: "5,6",
    weeklyLoad: "690 UA",
    status: "Revisar",
  },
];

export const alerts: readonly DemonstrationAlert[] = [
  {
    id: "readiness-threshold",
    title: "3 atletas com prontidão abaixo do limiar configurado",
    guidance: "Revisão do treinador recomendada.",
    level: "critical",
  },
  {
    id: "load-range",
    title: "2 atletas com carga acima da faixa configurada",
    guidance: "Verifique os dados antes do próximo treino.",
    level: "warning",
  },
  {
    id: "attendance-review",
    title: "5 ausências aguardando justificativa",
    guidance: "Revise os registros demonstrativos pendentes.",
    level: "neutral",
  },
];

export const readinessDistribution = [
  { label: "Baixa", range: "≤ 5,9", value: 8, level: "critical" },
  { label: "Moderada", range: "6,0–6,9", value: 25, level: "warning" },
  { label: "Boa", range: "7,0–8,4", value: 52, level: "good" },
  { label: "Ótima", range: "≥ 8,5", value: 15, level: "excellent" },
] as const;

export const readinessHistory = [
  { label: "09/abr", value: 6.2 },
  { label: "16/abr", value: 6.5 },
  { label: "23/abr", value: 6.7 },
  { label: "30/abr", value: 6.9 },
  { label: "07/mai", value: 7.1 },
  { label: "14/mai", value: 7.2 },
  { label: "21/mai", value: 7.4 },
  { label: "04/jun", value: 7.6 },
] as const;
