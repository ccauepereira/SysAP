import { readinessDistribution } from "./dashboard-data";

export function ReadinessDistribution() {
  return (
    <section aria-labelledby="distribution-heading" className="panel distribution-panel">
      <h2 id="distribution-heading">Distribuição de prontidão</h2>
      <div
        aria-label="Distribuição demonstrativa: 8% baixa, 25% moderada, 52% boa e 15% ótima"
        className="distribution-bar"
        role="img"
      >
        {readinessDistribution.map((item) => (
          <span
            className={`distribution-${item.level}`}
            key={item.label}
            style={{ width: `${item.value}%` }}
          />
        ))}
      </div>
      <dl className="distribution-legend">
        {readinessDistribution.map((item) => (
          <div key={item.label}>
            <dt>{item.value}%</dt>
            <dd>
              {item.label}
              <small>({item.range})</small>
            </dd>
          </div>
        ))}
      </dl>
      <p className="panel-note">Base demonstrativa: 48 atletas</p>
    </section>
  );
}
