/// Resumo de atividade esportiva sincronizada de forma autorizada.
///
/// Contém estritamente métricas agregadas aprovadas para o MVP.
/// Não contém rotas brutas, sono, calorias nem diagnósticos de saúde.
class ResumoDeAtividade {
  final DateTime inicio;
  final DateTime fim;
  final Duration duracao;
  final int? passos;
  final double? distanciaMetros;
  final int? frequenciaCardiacaMediaBpm;
  final int? frequenciaCardiacaMaximaBpm;
  final String origem;

  ResumoDeAtividade({
    required this.inicio,
    required this.fim,
    required this.duracao,
    this.passos,
    this.distanciaMetros,
    this.frequenciaCardiacaMediaBpm,
    this.frequenciaCardiacaMaximaBpm,
    required this.origem,
  }) {
    if (fim.isBefore(inicio)) {
      throw ArgumentError(
        'A data de fim não pode ser anterior à data de início.',
      );
    }
    if (duracao.isNegative) {
      throw ArgumentError('A duração da atividade não pode ser negativa.');
    }
    if (passos != null && passos! < 0) {
      throw ArgumentError('O total de passos não pode ser negativo.');
    }
    if (distanciaMetros != null && distanciaMetros! < 0) {
      throw ArgumentError('A distância em metros não pode ser negativa.');
    }
    if (frequenciaCardiacaMediaBpm != null &&
        frequenciaCardiacaMediaBpm! <= 0) {
      throw ArgumentError('A frequência cardíaca média deve ser positiva.');
    }
    if (frequenciaCardiacaMaximaBpm != null &&
        frequenciaCardiacaMaximaBpm! <= 0) {
      throw ArgumentError('A frequência cardíaca máxima deve ser positiva.');
    }
  }

  /// Retorna uma cópia filtrada garantindo que campos sem consentimento sejam omitidos.
  ResumoDeAtividade filtrarPorConsentimento({
    required bool consentiuDadosEsportivos,
    required bool consentiuFrequenciaCardiaca,
  }) {
    return ResumoDeAtividade(
      inicio: inicio,
      fim: fim,
      duracao: duracao,
      passos: consentiuDadosEsportivos ? passos : null,
      distanciaMetros: consentiuDadosEsportivos ? distanciaMetros : null,
      frequenciaCardiacaMediaBpm: consentiuFrequenciaCardiaca
          ? frequenciaCardiacaMediaBpm
          : null,
      frequenciaCardiacaMaximaBpm: consentiuFrequenciaCardiaca
          ? frequenciaCardiacaMaximaBpm
          : null,
      origem: origem,
    );
  }

  @override
  bool operator ==(Object other) =>
      identical(this, other) ||
      other is ResumoDeAtividade &&
          runtimeType == other.runtimeType &&
          inicio == other.inicio &&
          fim == other.fim &&
          duracao == other.duracao &&
          passos == other.passos &&
          distanciaMetros == other.distanciaMetros &&
          frequenciaCardiacaMediaBpm == other.frequenciaCardiacaMediaBpm &&
          frequenciaCardiacaMaximaBpm == other.frequenciaCardiacaMaximaBpm &&
          origem == other.origem;

  @override
  int get hashCode =>
      inicio.hashCode ^
      fim.hashCode ^
      duracao.hashCode ^
      passos.hashCode ^
      distanciaMetros.hashCode ^
      frequenciaCardiacaMediaBpm.hashCode ^
      frequenciaCardiacaMaximaBpm.hashCode ^
      origem.hashCode;
}
