/// Tipo de biometria suportada pelo hardware local.
enum TipoDeBiometria { impressaoDigital, reconhecimentoFacial, nenhuma }

/// Porta de serviço para autenticação biométrica local (desbloqueio do aplicativo).
///
/// A biometria atua exclusivamente como camada local de conveniência/segurança
/// para desbloquear o app após inatividade, NUNCA substituindo o login na API.
abstract class AutenticacaoBiometrica {
  /// Confirma se o hardware e o sistema operacional possuem biometria configurada.
  Future<bool> dispositivoSuportaBiometria();

  /// Identifica os tipos de biometria cadastrados.
  Future<List<TipoDeBiometria>> tiposDisponiveis();

  /// Solicita ao usuário a confirmação biométrica local.
  Future<bool> autenticarLocalmente({required String motivo});
}
