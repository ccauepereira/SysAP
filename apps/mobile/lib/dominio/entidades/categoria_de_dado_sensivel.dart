/// Categorias de dados sensíveis suportadas para consentimento no SysAP.
///
/// Dados de saúde, sensores esportivos e localização pós-treino são tratados como
/// dados sensíveis e exigem consentimento explícito, específico e revogável.
enum CategoriaDeDadoSensivel {
  /// Métricas básicas esportivas: duração, distância estimada e passos totais.
  dadosEsportivosBasicos(
    codigo: 'dados_esportivos_basicos',
    tituloEmPortugues: 'Dados Esportivos Básicos',
    descricaoFinalidade: 'Leitura de passos, duração de sessão e distância percorrida para registro de carga.',
    exigeConsentimentoSensivel: true,
  ),

  /// Frequência cardíaca durante atividades esportivas autorizadas.
  frequenciaCardiaca(
    codigo: 'frequencia_cardiaca',
    tituloEmPortugues: 'Frequência Cardíaca em Treino',
    descricaoFinalidade: 'Leitura de frequência cardíaca média e máxima durante o treino para cálculo determinístico de intensidade.',
    exigeConsentimentoSensivel: true,
  ),

  /// Localização e GPS pós-treino (importação/sincronização de rota após a sessão).
  localizacaoGpsPosTreino(
    codigo: 'localizacao_gps_pos_treino',
    tituloEmPortugues: 'Localização e GPS Pós-Treino',
    descricaoFinalidade: 'Leitura de coordenadas pós-sessão para mapeamento de deslocamento e mapa de calor no campo.',
    exigeConsentimentoSensivel: true,
  ),

  /// Compartilhamento de resumos esportivos com o treinador e a comissão técnica.
  compartilhamentoEquipeTecnica(
    codigo: 'compartilhamento_equipe_tecnica',
    tituloEmPortugues: 'Compartilhamento com Equipe Técnica',
    descricaoFinalidade: 'Disponibilização de resumos agregados de treino para visualização do treinador na plataforma.',
    exigeConsentimentoSensivel: true,
  );

  final String codigo;
  final String tituloEmPortugues;
  final String descricaoFinalidade;
  final bool exigeConsentimentoSensivel;

  const CategoriaDeDadoSensivel({
    required this.codigo,
    required this.tituloEmPortugues,
    required this.descricaoFinalidade,
    required this.exigeConsentimentoSensivel,
  });

  static CategoriaDeDadoSensivel? porCodigo(String codigo) {
    for (final categoria in CategoriaDeDadoSensivel.values) {
      if (categoria.codigo == codigo) {
        return categoria;
      }
    }
    return null;
  }
}
