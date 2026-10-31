import '../entidades/categoria_de_dado_sensivel.dart';
import '../entidades/consentimento_de_dados.dart';

/// Contrato para persistência e consulta de consentimentos do atleta.
abstract class RepositorioDeConsentimentos {
  /// Consulta todos os consentimentos registrados para o atleta na organização.
  Future<List<ConsentimentoDeDados>> consultarTodos({
    required String atletaId,
    required String organizacaoId,
  });

  /// Consulta o consentimento específico para uma categoria.
  Future<ConsentimentoDeDados?> consultarPorCategoria({
    required String atletaId,
    required String organizacaoId,
    required CategoriaDeDadoSensivel categoria,
  });

  /// Salva ou atualiza um registro de consentimento.
  Future<void> salvarConsentimento(ConsentimentoDeDados consentimento);

  /// Revoga um consentimento existente.
  Future<void> revogarConsentimento({
    required String atletaId,
    required String organizacaoId,
    required CategoriaDeDadoSensivel categoria,
    required DateTime dataDaRevogacao,
  });
}
