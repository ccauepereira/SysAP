import '../../dominio/entidades/consentimento_de_dados.dart';
import '../../dominio/repositorios/repositorio_de_consentimentos.dart';

/// Caso de uso para consultar todos os consentimentos de um atleta.
class ConsultarConsentimentosAtuais {
  final RepositorioDeConsentimentos repositorio;

  const ConsultarConsentimentosAtuais({required this.repositorio});

  Future<List<ConsentimentoDeDados>> executar({
    required String atletaId,
    required String organizacaoId,
  }) => repositorio.consultarTodos(
    atletaId: atletaId,
    organizacaoId: organizacaoId,
  );
}
