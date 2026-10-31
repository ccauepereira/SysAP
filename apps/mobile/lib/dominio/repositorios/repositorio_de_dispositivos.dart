import '../entidades/disponibilidade_de_recurso.dart';

/// Informações sobre uma fonte ou dispositivo esportivo conectado.
class InformacaoDeDispositivo {
  final String identificador;
  final String nomeExibicao;
  final String plataforma; // 'health_connect', 'healthkit', etc.
  final DisponibilidadeDeRecurso status;
  final DateTime? ultimaSincronizacaoEm;

  const InformacaoDeDispositivo({
    required this.identificador,
    required this.nomeExibicao,
    required this.plataforma,
    required this.status,
    this.ultimaSincronizacaoEm,
  });
}

/// Contrato para consultar e gerenciar fontes/dispositivos conectáveis.
abstract class RepositorioDeDispositivos {
  /// Lista as fontes e sensores disponíveis para a plataforma atual.
  Future<List<InformacaoDeDispositivo>> listarFontesConectadas();

  /// Registra ou atualiza metadados da última sincronização bem-sucedida de uma fonte.
  Future<void> atualizarUltimaSincronizacao({
    required String identificador,
    required DateTime sincronizadoEm,
  });
}
