import '../../dominio/servicos/detector_de_conectividade.dart';

/// Caso de uso para verificar o estado atual de conectividade do dispositivo.
class VerificarConectividade {
  final DetectorDeConectividade detector;

  const VerificarConectividade({required this.detector});

  Future<EstadoDeConectividade> executar() => detector.verificarConectividade();
}
