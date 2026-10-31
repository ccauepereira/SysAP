import '../entidades/categoria_de_dado_sensivel.dart';
import '../entidades/disponibilidade_de_recurso.dart';
import '../entidades/resumo_de_atividade.dart';

/// Porta de serviço para consultar e ler dados de saúde das plataformas nativas
/// (Health Connect no Android ou HealthKit no iOS).
abstract class FonteDeDadosDeSaude {
  /// Nome identificador da plataforma subjacente ('health_connect', 'healthkit').
  String get nomeDaPlataforma;

  /// Verifica se o serviço de saúde da plataforma está disponível no aparelho.
  Future<DisponibilidadeDeRecurso> verificarDisponibilidade();

  /// Solicita ao sistema operacional as permissões necessárias para as categorias aprovadas.
  Future<bool> solicitarPermissoes(Set<CategoriaDeDadoSensivel> categorias);

  /// Lê o resumo agregado de atividades para o período especificado, respeitando
  /// estritamente as categorias autorizadas.
  Future<ResumoDeAtividade?> lerResumoDeTreino({
    required DateTime inicio,
    required DateTime fim,
    required Set<CategoriaDeDadoSensivel> categoriasAutorizadas,
  });
}
