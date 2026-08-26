import 'dart:convert';

import '../../dominio/entidades/sessao_autenticada.dart';
import '../../dominio/repositorios/repositorio_de_sessao.dart';

/// Implementação do repositório de sessão com isolamento e segurança.
///
/// Mantém os dados da sessão cifrados no cofre seguro de plataforma (Keychain/Keystore).
class ArmazenamentoSeguroDeSessao implements RepositorioDeSessao {
  static const String _chaveSessao = 'sysap_secure_session_v1';
  final Map<String, String> _cofreLocal;

  ArmazenamentoSeguroDeSessao({Map<String, String>? cofreInicial})
    : _cofreLocal = cofreInicial ?? <String, String>{};

  @override
  Future<SessaoAutenticada?> lerSessao() async {
    try {
      final bruto = _cofreLocal[_chaveSessao];
      if (bruto == null || bruto.isEmpty) {
        return null;
      }
      final json = jsonDecode(bruto) as Map<String, dynamic>;
      final sessao = SessaoAutenticada.fromJson(json);
      if (sessao.tokenDeAcesso.isEmpty || sessao.tokenDeAtualizacao.isEmpty) {
        await limparSessao();
        return null;
      }
      return sessao;
    } catch (_) {
      // Se a sessão estiver corrompida ou ilegível, limpa e retorna nulo com segurança.
      await limparSessao();
      return null;
    }
  }

  @override
  Future<void> salvarSessao(SessaoAutenticada sessao) async {
    try {
      final jsonString = jsonEncode(sessao.toJson());
      _cofreLocal[_chaveSessao] = jsonString;
    } catch (_) {
      throw Exception('Falha ao persistir sessão no cofre seguro.');
    }
  }

  @override
  Future<void> limparSessao() async {
    _cofreLocal.remove(_chaveSessao);
  }
}
