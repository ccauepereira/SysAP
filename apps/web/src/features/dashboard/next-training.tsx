import { CalendarDays, Clock3, MapPin } from "lucide-react";

export function NextTraining() {
  return (
    <section aria-labelledby="training-heading" className="panel training-panel">
      <div className="section-heading">
        <div>
          <CalendarDays aria-hidden="true" size={19} />
          <h2 id="training-heading">Próximo treino</h2>
        </div>
        <span aria-disabled="true" className="future-link">
          Ver calendário <span className="visually-hidden">— em breve</span>
        </span>
      </div>
      <div className="training-summary">
        <time dateTime="2025-06-05T07:00:00-03:00">
          <small>QUI</small>
          <strong>05</strong>
          <small>JUN</small>
        </time>
        <div>
          <h3>Treino demonstrativo de força</h3>
          <p>
            <span>Turma exemplo</span>
            <span><Clock3 aria-hidden="true" size={14} /> 07:00–08:30</span>
            <span><MapPin aria-hidden="true" size={14} /> Academia</span>
          </p>
          <div className="training-tags">
            <span>Força</span>
            <span>90 min</span>
          </div>
        </div>
      </div>
      <p className="panel-note">Agenda demonstrativa — nenhuma ação será salva nesta fase.</p>
    </section>
  );
}
