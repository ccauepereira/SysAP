import '../entidades/sessao_autenticada.dart';

/// Contrato de repositório para persistência segura de sessão autenticada.
abstract class RepositorioDeSessao {
  /// Lê a sessão armazenada com segurança. Retorna null se inexistente ou corrompida.
  Future<SessaoAutenticada?> lerSessao();

  /// Grava a sessão no cofre seguro do dispositivo (Keychain no iOS / Keystore no Android).
  Future<void> salvarSessao(SessaoAutenticada sessao);

  /// Remove integralmente a sessão do dispositivo (logout / revogação).
  Future<void> limparSessao();
}
