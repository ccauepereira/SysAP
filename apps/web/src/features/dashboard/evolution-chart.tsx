import { Info } from "lucide-react";
import { readinessHistory } from "./dashboard-data";

const width = 720;
const height = 220;
const left = 42;
const right = 16;
const top = 24;
const bottom = 38;
const plotWidth = width - left - right;
const plotHeight = height - top - bottom;
const minimum = 5;
const maximum = 8;

const points = readinessHistory.map((item, index) => {
  const x = left + (index / (readinessHistory.length - 1)) * plotWidth;
  const y = top + ((maximum - item.value) / (maximum - minimum)) * plotHeight;
  return { ...item, x, y };
});

const polyline = points.map(({ x, y }) => `${x},${y}`).join(" ");

export function EvolutionChart() {
  return (
    <section aria-labelledby="evolution-heading" className="panel evolution-panel">
      <div className="section-heading">
        <div>
          <h2 id="evolution-heading">Evolução da turma</h2>
          <Info aria-label="Série exclusivamente demonstrativa" size={16} />
        </div>
        <span className="chart-period">Últimas 8 semanas</span>
      </div>
      <figure>
        <svg
          aria-labelledby="chart-title chart-description"
          className="evolution-chart"
          role="img"
          viewBox={`0 0 ${width} ${height}`}
        >
          <title id="chart-title">Evolução demonstrativa da prontidão média</title>
          <desc id="chart-description">
            A prontidão média demonstrativa sobe gradualmente de 6,2 em 9 de abril para 7,6 em 4 de junho.
          </desc>
          {[5, 6, 7, 8].map((tick) => {
            const y = top + ((maximum - tick) / (maximum - minimum)) * plotHeight;
            return (
              <g key={tick}>
                <line className="chart-grid-line" x1={left} x2={width - right} y1={y} y2={y} />
                <text className="chart-axis-label" x={4} y={y + 4}>{tick}</text>
              </g>
            );
          })}
          <polyline className="chart-line" fill="none" points={polyline} />
          {points.map((point) => (
            <g key={point.label}>
              <circle className="chart-point" cx={point.x} cy={point.y} r={4} />
              <text className="chart-value" textAnchor="middle" x={point.x} y={point.y - 11}>
                {point.value.toLocaleString("pt-BR")}
              </text>
              <text className="chart-axis-label" textAnchor="middle" x={point.x} y={height - 9}>
                {point.label}
              </text>
            </g>
          ))}
        </svg>
        <figcaption>
          Série demonstrativa de prontidão média: 6,2; 6,5; 6,7; 6,9; 7,1; 7,2; 7,4; 7,6.
        </figcaption>
      </figure>
    </section>
  );
}
