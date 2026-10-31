import '../../dominio/entidades/categoria_de_dado_sensivel.dart';
import '../../dominio/entidades/disponibilidade_de_recurso.dart';
import '../../dominio/entidades/falha_de_integracao.dart';
import '../../dominio/entidades/resumo_de_atividade.dart';
import '../../dominio/repositorios/repositorio_de_dispositivos.dart';
import '../../dominio/repositorios/repositorio_de_sincronizacao.dart';
import '../../dominio/servicos/detector_de_conectividade.dart';
import '../../dominio/servicos/fonte_de_dados_de_saude.dart';
import '../consentimento/gerenciador_de_consentimento.dart';

/// Orquestrador de sincronização de dados de dispositivos e sensores de saúde.
///
/// Implementa as salvaguardas da Fase 3.5:
/// 1. Nunca tenta sincronizar se o dispositivo estiver offline (evita retry cego);
/// 2. Valida consentimento explícito no SysAP antes de ler da fonte;
/// 3. Confirma disponibilidade e permissão na fonte de saúde nativa;
/// 4. Filtra rigorosamente o resumo lido, omitindo métricas não consentidas;
/// 5. Atualiza o status nos repositórios e registra falha segura em caso de erro.
class SincronizadorDeDadosDeDispositivo {
  final FonteDeDadosDeSaude fonteSaude;
  final GerenciadorDeConsentimento gerenciadorConsentimento;
  final RepositorioDeDispositivos repositorioDispositivos;
  final RepositorioDeSincronizacao repositorioSincronizacao;
  final DetectorDeConectividade detectorConectividade;

  const SincronizadorDeDadosDeDispositivo({
    required this.fonteSaude,
    required this.gerenciadorConsentimento,
    required this.repositorioDispositivos,
    required this.repositorioSincronizacao,
    required this.detectorConectividade,
  });

  /// Executa o fluxo seguro de sincronização de um treino.
  Future<ResumoDeAtividade?> sincronizarTreino({
    required String atletaId,
    required String organizacaoId,
    required String sincronizacaoId,
    required DateTime inicio,
    required DateTime fim,
  }) async {
    // 1. Checagem de conectividade
    final conectividade = await detectorConectividade.verificarConectividade();
    if (conectividade == EstadoDeConectividade.offline) {
      throw const FalhaDeConectividade(
        'Dispositivo sem conexão de rede. A sincronização foi suspensa para evitar retries cegos.',
      );
    }

    // 2. Mapeamento de consentimentos vigentes
    final consentiuDadosEsportivos = await gerenciadorConsentimento
        .estaConsentido(
          atletaId: atletaId,
          organizacaoId: organizacaoId,
          categoria: CategoriaDeDadoSensivel.dadosEsportivosBasicos,
        );

    final consentiuFC = await gerenciadorConsentimento.estaConsentido(
      atletaId: atletaId,
      organizacaoId: organizacaoId,
      categoria: CategoriaDeDadoSensivel.frequenciaCardiaca,
    );

    // Se o atleta não consentiu com nenhuma categoria de saúde, a sincronização é bloqueada
    if (!consentiuDadosEsportivos && !consentiuFC) {
      throw const FalhaDeConsentimentoAusente(
        'Nenhum consentimento de saúde ou atividade física está ativo para esta sincronização.',
      );
    }

    // 3. Checagem de disponibilidade na plataforma nativa
    final disponibilidade = await fonteSaude.verificarDisponibilidade();
    if (disponibilidade == DisponibilidadeDeRecurso.permissaoNegada) {
      throw const FalhaDePermissaoDeSistema(
        'Permissão de leitura negada no Health Connect / Apple HealthKit.',
      );
    }
    if (disponibilidade != DisponibilidadeDeRecurso.disponivel) {
      throw FalhaDeFonteIndisponivel(
        'Fonte de saúde não está disponível no momento (${disponibilidade.rotuloEmPortugues}).',
      );
    }

    // 4. Registro de início no repositório de sincronização
    final agora = DateTime.now().toUtc();
    final registro = RegistroDeSincronizacao(
      id: sincronizacaoId,
      atletaId: atletaId,
      organizacaoId: organizacaoId,
      origem: fonteSaude.nomeDaPlataforma,
      periodoInicio: inicio,
      periodoFim: fim,
      estado: EstadoDeSincronizacao.emAndamento,
      criadoEm: agora,
    );
    await repositorioSincronizacao.registrarInicio(registro);

    try {
      final categoriasAutorizadas = <CategoriaDeDadoSensivel>{
        if (consentiuDadosEsportivos)
          CategoriaDeDadoSensivel.dadosEsportivosBasicos,
        if (consentiuFC) CategoriaDeDadoSensivel.frequenciaCardiaca,
      };

      // 5. Leitura da fonte nativa com categorias estritamente autorizadas
      final resumoBruto = await fonteSaude.lerResumoDeTreino(
        inicio: inicio,
        fim: fim,
        categoriasAutorizadas: categoriasAutorizadas,
      );

      if (resumoBruto == null) {
        await repositorioSincronizacao.registrarConclusao(
          id: sincronizacaoId,
          estado: EstadoDeSincronizacao.concluida,
          concluidoEm: DateTime.now().toUtc(),
        );
        return null;
      }

      // 6. Filtragem rigorosa final por consentimento (zero vazamento de FC sem autorização)
      final resumoFiltrado = resumoBruto.filtrarPorConsentimento(
        consentiuDadosEsportivos: consentiuDadosEsportivos,
        consentiuFrequenciaCardiaca: consentiuFC,
      );

      // 7. Atualização de sucesso
      await repositorioSincronizacao.registrarConclusao(
        id: sincronizacaoId,
        estado: EstadoDeSincronizacao.concluida,
        concluidoEm: DateTime.now().toUtc(),
      );

      await repositorioDispositivos.atualizarUltimaSincronizacao(
        identificador: fonteSaude.nomeDaPlataforma,
        sincronizadoEm: DateTime.now().toUtc(),
      );

      return resumoFiltrado;
    } catch (e) {
      await repositorioSincronizacao.registrarConclusao(
        id: sincronizacaoId,
        estado: EstadoDeSincronizacao.falhaSegura,
        concluidoEm: DateTime.now().toUtc(),
      );
      rethrow;
    }
  }
}
