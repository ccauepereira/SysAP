/// Estados de conectividade com a rede.
enum EstadoDeConectividade { online, offline, desconhecido }

/// Porta de serviço para monitorar a conectividade de rede do aparelho.
abstract class DetectorDeConectividade {
  /// Consulta o estado atual de conectividade.
  Future<EstadoDeConectividade> verificarConectividade();

  /// Stream reativo para observar alterações na conectividade.
  Stream<EstadoDeConectividade> observarConectividade();
}
