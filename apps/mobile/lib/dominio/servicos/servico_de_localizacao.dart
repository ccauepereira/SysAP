import '../entidades/disponibilidade_de_recurso.dart';

/// Porta de serviço para capacidade futura de localização e GPS pós-treino.
///
/// Nesta fase, NÃO executa coleta de localização, NÃO solicita permissões em segundo plano
/// e NÃO realiza rastreamento ao vivo.
abstract class ServicoDeLocalizacao {
  /// Verifica se o dispositivo possui capacidade de GPS disponível para uso futuro.
  Future<DisponibilidadeDeRecurso> verificarCapacidadeGps();
}
