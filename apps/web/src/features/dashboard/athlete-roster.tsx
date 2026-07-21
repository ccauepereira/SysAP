import { athletes, type AthleteStatus } from "./dashboard-data";

function statusClassName(status: AthleteStatus): string {
  if (status === "Ótimo" || status === "Bom") return "status-good";
  if (status === "Atenção") return "status-warning";
  return "status-critical";
}

function AthleteIdentity({ athlete }: { readonly athlete: (typeof athletes)[number] }) {
  return (
    <div className="athlete-identity">
      <span aria-hidden="true" className="avatar">
        {athlete.initials}
      </span>
      <span>
        <strong>{athlete.name}</strong>
        <small>{athlete.position}</small>
      </span>
    </div>
  );
}

export function AthleteRoster() {
  return (
    <section aria-labelledby="athletes-heading" className="panel athletes-panel">
      <div className="section-heading">
        <h2 id="athletes-heading">Atletas da turma</h2>
        <span aria-disabled="true" className="future-link">
          Ver todos <span className="visually-hidden">— em breve</span>
        </span>
      </div>

      <div className="athlete-table-wrapper">
        <table>
          <caption className="visually-hidden">
            Atletas demonstrativos e indicadores da semana
          </caption>
          <thead>
            <tr>
              <th scope="col">Atleta</th>
              <th scope="col">Presença</th>
              <th scope="col">Prontidão</th>
              <th scope="col">Carga semanal</th>
              <th scope="col">Status</th>
            </tr>
          </thead>
          <tbody>
            {athletes.map((athlete) => (
              <tr key={athlete.id}>
                <th scope="row">
                  <AthleteIdentity athlete={athlete} />
                </th>
                <td>{athlete.attendance}</td>
                <td>{athlete.readiness}</td>
                <td>{athlete.weeklyLoad}</td>
                <td>
                  <span className={`status-badge ${statusClassName(athlete.status)}`}>
                    {athlete.status}
                  </span>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>

      <ul aria-label="Lista mobile de atletas demonstrativos" className="athlete-mobile-list">
        {athletes.map((athlete) => (
          <li key={athlete.id}>
            <AthleteIdentity athlete={athlete} />
            <dl>
              <div>
                <dt>Presença</dt>
                <dd>{athlete.attendance}</dd>
              </div>
              <div>
                <dt>Prontidão</dt>
                <dd>{athlete.readiness}</dd>
              </div>
            </dl>
            <span className={`status-badge ${statusClassName(athlete.status)}`}>
              {athlete.status}
            </span>
          </li>
        ))}
      </ul>
    </section>
  );
}
