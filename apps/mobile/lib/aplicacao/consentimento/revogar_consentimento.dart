import '../../dominio/entidades/categoria_de_dado_sensivel.dart';
import '../../dominio/repositorios/repositorio_de_consentimentos.dart';

/// Caso de uso para revogação imediata de um consentimento concedido anteriormente.
class RevogarConsentimento {
  final RepositorioDeConsentimentos repositorio;

  const RevogarConsentimento({required this.repositorio});

  Future<void> executar({
    required String atletaId,
    required String organizacaoId,
    required CategoriaDeDadoSensivel categoria,
    required DateTime dataDaRevogacao,
  }) => repositorio.revogarConsentimento(
    atletaId: atletaId,
    organizacaoId: organizacaoId,
    categoria: categoria,
    dataDaRevogacao: dataDaRevogacao,
  );
}
