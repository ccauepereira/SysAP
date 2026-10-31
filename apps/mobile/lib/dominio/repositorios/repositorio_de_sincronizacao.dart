/// Estado de uma sincronização de metadados esportivos.
enum EstadoDeSincronizacao { pendente, emAndamento, concluida, falhaSegura }

/// Metadados de registro de sincronização (sem dados brutos de saúde em aberto).
class RegistroDeSincronizacao {
  final String id;
  final String atletaId;
  final String organizacaoId;
  final String origem;
  final DateTime periodoInicio;
  final DateTime periodoFim;
  final EstadoDeSincronizacao estado;
  final DateTime criadoEm;
  final DateTime? concluidoEm;

  const RegistroDeSincronizacao({
    required this.id,
    required this.atletaId,
    required this.organizacaoId,
    required this.origem,
    required this.periodoInicio,
    required this.periodoFim,
    required this.estado,
    required this.criadoEm,
    this.concluidoEm,
  });
}

/// Contrato para rastrear operações de sincronização sem expor dados brutos.
abstract class RepositorioDeSincronizacao {
  Future<void> registrarInicio(RegistroDeSincronizacao registro);
  Future<void> registrarConclusao({
    required String id,
    required EstadoDeSincronizacao estado,
    required DateTime concluidoEm,
  });
  Future<List<RegistroDeSincronizacao>> listarPendentes({
    required String atletaId,
    required String organizacaoId,
  });
}
