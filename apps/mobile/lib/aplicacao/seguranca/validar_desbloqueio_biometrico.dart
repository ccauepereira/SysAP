import '../../dominio/servicos/autenticacao_biometrica.dart';

/// Caso de uso para validação do desbloqueio biométrico local.
class ValidarDesbloqueioBiometrico {
  final AutenticacaoBiometrica biometria;

  const ValidarDesbloqueioBiometrico({required this.biometria});

  Future<bool> executar({required String motivo}) async {
    final suporta = await biometria.dispositivoSuportaBiometria();
    if (!suporta) return false;

    return biometria.autenticarLocalmente(motivo: motivo);
  }
}
