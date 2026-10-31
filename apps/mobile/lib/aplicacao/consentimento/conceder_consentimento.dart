import '../../dominio/entidades/consentimento_de_dados.dart';
import '../../dominio/repositorios/repositorio_de_consentimentos.dart';

/// Caso de uso para registrar novo consentimento ou atualizar termo.
class ConcederConsentimento {
  final RepositorioDeConsentimentos repositorio;

  const ConcederConsentimento({required this.repositorio});

  Future<void> executar(ConsentimentoDeDados consentimento) =>
      repositorio.salvarConsentimento(consentimento);
}
