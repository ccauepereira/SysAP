import '../../../dominio/falhas/falha_de_autenticacao.dart';
import '../../../infraestrutura/api/cliente_da_api.dart';

/// Serviço de aplicação responsável pelo fluxo de ativação de conta de atleta.
class ServicoDeAtivacao {
  final ClienteDaApi clienteApi;

  const ServicoDeAtivacao({required this.clienteApi});

  /// Inicia o desafio OTP para a matrícula informada (/v1/activation/start).
  Future<void> solicitarDesafio(String matricula) async {
    final matriculaLimpa = matricula.trim();
    if (matriculaLimpa.length != 10 ||
        !RegExp(r'^[0-9]{10}$').hasMatch(matriculaLimpa)) {
      throw const FalhaDeAutenticacao(
        mensagem: 'Matrícula inválida.',
        tipo: TipoDeFalhaDeAutenticacao.formatoInvalido,
      );
    }

    await clienteApi.post(
      '/v1/activation/start',
      corpo: {'enrollment_number': matriculaLimpa},
    );
  }

  /// Verifica o código OTP e obtém a activation_proof em memória (/v1/activation/verify).
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
      '/v1/activation/verify',
      corpo: {'enrollment_number': matriculaLimpa, 'code': codigoLimpo},
    );

    final proof = resposta.corpoJson['activation_proof'] as String?;
    if (proof == null || proof.isEmpty) {
      throw const FalhaDeAutenticacao(
        mensagem: 'Não foi possível validar o código de ativação.',
        tipo: TipoDeFalhaDeAutenticacao.credenciaisInvalidas,
      );
    }

    return proof;
  }

  /// Conclui a ativação definindo a senha (/v1/activation/complete).
  Future<void> concluirAtivacao({
    required String activationProof,
    required String novaSenha,
  }) async {
    if (novaSenha.length < 15) {
      throw const FalhaDeAutenticacao(
        mensagem: 'A nova senha deve ter no mínimo 15 caracteres.',
        tipo: TipoDeFalhaDeAutenticacao.formatoInvalido,
      );
    }

    await clienteApi.post(
      '/v1/activation/complete',
      corpo: {'activation_proof': activationProof, 'password': novaSenha},
    );
  }
}
