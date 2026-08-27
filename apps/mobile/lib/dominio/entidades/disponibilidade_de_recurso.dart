/// Representação do estado de prontidão e disponibilidade de um recurso do dispositivo
/// (como Health Connect, HealthKit, biometria ou GPS).
enum DisponibilidadeDeRecurso {
  /// O recurso está presente, suportado e apto para uso após consentimento.
  disponivel(codigo: 'disponivel', rotuloEmPortugues: 'Disponível'),

  /// O recurso não é suportado pelo hardware ou sistema operacional.
  indisponivel(
    codigo: 'indisponivel',
    rotuloEmPortugues: 'Indisponível neste dispositivo',
  ),

  /// O recurso ainda não teve permissões solicitadas no sistema.
  naoSolicitado(codigo: 'nao_solicitado', rotuloEmPortugues: 'Não solicitado'),

  /// A permissão foi solicitada ao sistema operacional e foi recusada pelo usuário.
  permissaoNegada(
    codigo: 'permissao_negada',
    rotuloEmPortugues: 'Permissão negada pelo sistema',
  ),

  /// O serviço existe mas necessita de atualização no sistema (ex: Health Connect no Android).
  requerAtualizacao(
    codigo: 'requer_atualizacao',
    rotuloEmPortugues: 'Requer atualização do aplicativo de saúde',
  ),

  /// Não existem amostras de dados suficientes para processamento confiável.
  dadosInsuficientes(
    codigo: 'dados_insuficientes',
    rotuloEmPortugues: 'Dados insuficientes',
  );

  final String codigo;
  final String rotuloEmPortugues;

  const DisponibilidadeDeRecurso({
    required this.codigo,
    required this.rotuloEmPortugues,
  });

  bool get estaOperacional => this == DisponibilidadeDeRecurso.disponivel;
}
