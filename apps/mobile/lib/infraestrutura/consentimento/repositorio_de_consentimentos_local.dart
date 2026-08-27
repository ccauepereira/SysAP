import '../../dominio/entidades/categoria_de_dado_sensivel.dart';
import '../../dominio/entidades/consentimento_de_dados.dart';
import '../../dominio/repositorios/repositorio_de_consentimentos.dart';

/// Implementação local/stub do repositório de consentimentos para a Fase 3.5.
///
/// Mantém os consentimentos em memória/armazenamento local protegido durante a fundação da Fase 3.5,
/// até a conexão completa com os endpoints remotos da API Go na Fase 4.
class RepositorioDeConsentimentosLocal implements RepositorioDeConsentimentos {
  final Map<String, ConsentimentoDeDados> _armazenamento = {};

  String _gerarChave(
    String atletaId,
    String organizacaoId,
    CategoriaDeDadoSensivel categoria,
  ) {
    return '$organizacaoId:$atletaId:${categoria.name}';
  }

  @override
  Future<ConsentimentoDeDados?> consultarPorCategoria({
    required String atletaId,
    required String organizacaoId,
    required CategoriaDeDadoSensivel categoria,
  }) async {
    final chave = _gerarChave(atletaId, organizacaoId, categoria);
    return _armazenamento[chave];
  }

  @override
  Future<List<ConsentimentoDeDados>> consultarTodos({
    required String atletaId,
    required String organizacaoId,
  }) async {
    final prefixo = '$organizacaoId:$atletaId:';
    return _armazenamento.entries
        .where((e) => e.key.startsWith(prefixo))
        .map((e) => e.value)
        .toList();
  }

  @override
  Future<void> salvarConsentimento(ConsentimentoDeDados consentimento) async {
    final chave = _gerarChave(
      consentimento.atletaId,
      consentimento.organizacaoId,
      consentimento.categoria,
    );
    _armazenamento[chave] = consentimento;
  }

  @override
  Future<void> revogarConsentimento({
    required String atletaId,
    required String organizacaoId,
    required CategoriaDeDadoSensivel categoria,
    required DateTime dataDaRevogacao,
  }) async {
    final chave = _gerarChave(atletaId, organizacaoId, categoria);
    final existente = _armazenamento[chave];
    if (existente != null) {
      _armazenamento[chave] = existente.revogar(
        dataDaRevogacao: dataDaRevogacao,
      );
    }
  }
}
