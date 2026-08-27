import '../../dominio/entidades/disponibilidade_de_recurso.dart';
import '../../dominio/servicos/fonte_de_dados_de_saude.dart';

/// Caso de uso para verificar a prontidão do Health Connect ou HealthKit.
class VerificarDisponibilidadeDeSaude {
  final FonteDeDadosDeSaude fonte;

  const VerificarDisponibilidadeDeSaude({required this.fonte});

  Future<DisponibilidadeDeRecurso> executar() =>
      fonte.verificarDisponibilidade();
}
