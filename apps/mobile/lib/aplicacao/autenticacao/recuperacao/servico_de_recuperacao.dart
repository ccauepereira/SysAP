import '../../../dominio/falhas/falha_de_autenticacao.dart';
import '../../../infraestrutura/api/cliente_da_api.dart';

/// Serviço de aplicação responsável pela recuperação de acesso por e-mail/matrícula.
class ServicoDeRecuperacao {
  final ClienteDaApi clienteApi;

  const ServicoDeRecuperacao({required this.clienteApi});

  /// Solicita recuperação de senha (/v1/auth/password-recovery/start).
  Future<void> solicitarRecuperacao(String matricula) async {
    final matriculaLimpa = matricula.trim();
    if (matriculaLimpa.length != 10 ||
        !RegExp(r'^[0-9]{10}$').hasMatch(matriculaLimpa)) {
      throw const FalhaDeAutenticacao(
        mensagem: 'Matrícula inválida.',
        tipo: TipoDeFalhaDeAutenticacao.formatoInvalido,
      );
    }

    await clienteApi.post(
      '/v1/auth/password-recovery/start',
      corpo: {'enrollment_number': matriculaLimpa},
    );
  }

  /// Valida o código OTP recebido e obtém a proof em memória (/v1/auth/password-recovery/verify).
  Future<String> verificarCodigo({
    required String matricula,
    required String codigo,
  }) async {
    final matriculaLimpa = matricula.trim();
    final codigoLimpo = codigo.trim();

    if (codigoLimpo.length != 6 ||
        !RegExp(r'^[0-9]{6}$').hasMatch(codigoLimpo)) {
      throw const FalhaDeAutenticacao(
        mensagem: 'O código OTP deve conter 6 dígitos numéricos.',
        tipo: TipoDeFalhaDeAutenticacao.formatoInvalido,
      );
    }

    final resposta = await clienteApi.post(
      '/v1/auth/password-recovery/verify',
      corpo: {'enrollment_number': matriculaLimpa, 'code': codigoLimpo},
    );

    final proof = resposta.corpoJson['proof'] as String?;
    if (proof == null || proof.isEmpty) {
      throw const FalhaDeAutenticacao(
        mensagem: 'Código de recuperação inválido ou expirado.',
        tipo: TipoDeFalhaDeAutenticacao.credenciaisInvalidas,
      );
    }

    return proof;
  }

  /// Conclui a recuperação redefinindo a senha (/v1/auth/password-recovery/complete).
  Future<void> concluirRecuperacao({
    required String proof,
    required String novaSenha,
  }) async {
    if (novaSenha.length < 15) {
      throw const FalhaDeAutenticacao(
        mensagem: 'A nova senha deve ter no mínimo 15 caracteres.',
        tipo: TipoDeFalhaDeAutenticacao.formatoInvalido,
      );
    }

    await clienteApi.post(
      '/v1/auth/password-recovery/complete',
      corpo: {'proof': proof, 'password': novaSenha},
    );
  }
}
