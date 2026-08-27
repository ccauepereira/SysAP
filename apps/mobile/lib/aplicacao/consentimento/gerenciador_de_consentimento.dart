import '../../dominio/entidades/categoria_de_dado_sensivel.dart';
import '../../dominio/entidades/falha_de_integracao.dart';
import '../../dominio/entidades/papel_do_usuario.dart';
import '../../dominio/repositorios/repositorio_de_consentimentos.dart';

/// Gerenciador central de consentimento do SysAP.
///
/// Garante que nenhuma leitura, sincronização ou compartilhamento de dados sensíveis
/// seja executada sem consentimento explícito e vigente do atleta.
class GerenciadorDeConsentimento {
  final RepositorioDeConsentimentos repositorio;

  const GerenciadorDeConsentimento({required this.repositorio});

  /// Valida se o atleta possui consentimento vigente para uma categoria específica.
  Future<bool> estaConsentido({
    required String atletaId,
    required String organizacaoId,
    required CategoriaDeDadoSensivel categoria,
  }) async {
    final consentimento = await repositorio.consultarPorCategoria(
      atletaId: atletaId,
      organizacaoId: organizacaoId,
      categoria: categoria,
    );
    return consentimento?.estaVigente ?? false;
  }

  /// Lança FalhaDeConsentimentoAusente caso o consentimento não esteja ativo.
  Future<void> exigirConsentimentoVigente({
    required String atletaId,
    required String organizacaoId,
    required CategoriaDeDadoSensivel categoria,
  }) async {
    final ativo = await estaConsentido(
      atletaId: atletaId,
      organizacaoId: organizacaoId,
      categoria: categoria,
    );

    if (!ativo) {
      throw FalhaDeConsentimentoAusente(
        'Consentimento ausente ou revogado para a categoria: ${categoria.tituloEmPortugues}',
      );
    }
  }

  /// Valida autorização de acesso a dados de saúde/sensores respeitando papéis e regras de organização.
  ///
  /// Regra estrita: O papel Owner NÃO tem acesso irrestrito a dados individuais de saúde/GPS.
  /// Para ter acesso, é obrigatório que:
  /// 1. O solicitante possua papel 'owner' e vínculo ativo na mesma organização;
  /// 2. O atleta tenha concedido explicitamente consentimento na categoria 'compartilhamentoEquipeTecnica';
  /// 3. O atleta tenha consentimento ativo na categoria solicitada.
  Future<void> validarAutorizacaoDeAcesso({
    required String solicitanteId,
    required PapelDoUsuario solicitantePapel,
    required String solicitanteOrganizacaoId,
    required bool solicitanteVinculoAtivo,
    required String alvoAtletaId,
    required String alvoOrganizacaoId,
    required CategoriaDeDadoSensivel categoria,
  }) async {
    // 1. Isolamento multi-tenant estrito
    if (solicitanteOrganizacaoId != alvoOrganizacaoId ||
        !solicitanteVinculoAtivo) {
      throw const FalhaDeAutorizacaoOwner(
        'Acesso negado: solicitante não pertence à mesma organização ativa do atleta.',
      );
    }

    // 2. Se o solicitante for o próprio atleta
    if (solicitantePapel == PapelDoUsuario.athlete) {
      if (solicitanteId != alvoAtletaId) {
        throw const FalhaDeAutorizacaoOwner(
          'Acesso negado: atleta não pode acessar dados de outro atleta.',
        );
      }
      await exigirConsentimentoVigente(
        atletaId: alvoAtletaId,
        organizacaoId: alvoOrganizacaoId,
        categoria: categoria,
      );
      return;
    }

    // 3. Se o solicitante for Owner (Gestão)
    if (solicitantePapel == PapelDoUsuario.owner) {
      // Exige consentimento explícito do atleta para compartilhamento com a equipe técnica
      final consentiuCompartilhamento = await estaConsentido(
        atletaId: alvoAtletaId,
        organizacaoId: alvoOrganizacaoId,
        categoria: CategoriaDeDadoSensivel.compartilhamentoEquipeTecnica,
      );

      if (!consentiuCompartilhamento) {
        throw const FalhaDeAutorizacaoOwner(
          'Acesso restrito: o atleta não autorizou o compartilhamento de dados com a gestão/comissão técnica.',
        );
      }

      // Exige consentimento específico da categoria
      final consentiuCategoria = await estaConsentido(
        atletaId: alvoAtletaId,
        organizacaoId: alvoOrganizacaoId,
        categoria: categoria,
      );

      if (!consentiuCategoria) {
        throw FalhaDeConsentimentoAusente(
          'Acesso negado: o atleta não concedeu consentimento para ${categoria.tituloEmPortugues}.',
        );
      }

      return;
    }

    // Outros papéis não possuem acesso autorizado a dados sensíveis de saúde
    throw const FalhaDeAutorizacaoOwner(
      'Acesso negado: papel do solicitante não autorizado para leitura de dados de sensores.',
    );
  }
}
