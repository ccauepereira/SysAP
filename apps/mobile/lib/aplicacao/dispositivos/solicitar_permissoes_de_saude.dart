import '../../dominio/entidades/categoria_de_dado_sensivel.dart';
import '../../dominio/entidades/falha_de_integracao.dart';
import '../../dominio/servicos/fonte_de_dados_de_saude.dart';
import '../consentimento/gerenciador_de_consentimento.dart';

/// Caso de uso para solicitar permissões de sensores ao sistema operacional.
///
/// Exige que o consentimento no SysAP tenha sido concedido previamente.
class SolicitarPermissoesDeSaude {
  final FonteDeDadosDeSaude fonte;
  final GerenciadorDeConsentimento gerenciadorConsentimento;

  const SolicitarPermissoesDeSaude({
    required this.fonte,
    required this.gerenciadorConsentimento,
  });

  Future<bool> executar({
    required String atletaId,
    required String organizacaoId,
    required Set<CategoriaDeDadoSensivel> categorias,
  }) async {
    for (final cat in categorias) {
      final ativo = await gerenciadorConsentimento.estaConsentido(
        atletaId: atletaId,
        organizacaoId: organizacaoId,
        categoria: cat,
      );

      if (!ativo) {
        throw FalhaDeConsentimentoAusente(
          'Consentimento SysAP não encontrado para ${cat.tituloEmPortugues}. A permissão de sistema não será solicitada sem consentimento prévio.',
        );
      }
    }

    return fonte.solicitarPermissoes(categorias);
  }
}
