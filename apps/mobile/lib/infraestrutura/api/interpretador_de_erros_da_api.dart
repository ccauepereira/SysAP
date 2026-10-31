import 'dart:convert';

import '../../dominio/falhas/falha_de_autenticacao.dart';

/// Interpretador seguro de respostas de erro da API SysAP.
class InterpretadorDeErrosDaApi {
  const InterpretadorDeErrosDaApi();

  /// Interpreta código HTTP e corpo JSON de erro da API.
  FalhaDeAutenticacao interpretar({required int codigoHttp, String? corpo}) {
    String? codigoDeErro;
    if (corpo != null && corpo.isNotEmpty) {
      try {
        final data = jsonDecode(corpo) as Map<String, dynamic>;
        final errorObj = data['error'] as Map<String, dynamic>?;
        codigoDeErro = errorObj?['code'] as String?;
      } catch (_) {
        // Ignora falha de parse e usa mapeamento por código HTTP
      }
    }

    if (codigoDeErro == 'mfa_required') {
      return FalhaDeAutenticacao(
        mensagem: 'Autenticação em dois fatores necessária.',
        tipo: TipoDeFalhaDeAutenticacao.mfaRequerido,
        codigoHttp: codigoHttp,
      );
    }

    switch (codigoHttp) {
      case 401:
        return FalhaDeAutenticacao(
          mensagem: 'Matrícula ou senha inválida, ou sessão expirada.',
          tipo: TipoDeFalhaDeAutenticacao.credenciaisInvalidas,
          codigoHttp: codigoHttp,
        );
      case 403:
        return FalhaDeAutenticacao(
          mensagem: 'Acesso negado para a organização ou perfil atual.',
          tipo: TipoDeFalhaDeAutenticacao.acessoNegado,
          codigoHttp: codigoHttp,
        );
      case 422:
        return FalhaDeAutenticacao(
          mensagem: 'Formato de matrícula ou senha inválido.',
          tipo: TipoDeFalhaDeAutenticacao.formatoInvalido,
          codigoHttp: codigoHttp,
        );
      case 503:
        return FalhaDeAutenticacao(
          mensagem:
              'O serviço de autenticação está temporariamente indisponível.',
          tipo: TipoDeFalhaDeAutenticacao.servicoIndisponivel,
          codigoHttp: codigoHttp,
        );
      default:
        return FalhaDeAutenticacao(
          mensagem: 'Ocorreu um erro ao processar a solicitação.',
          tipo: TipoDeFalhaDeAutenticacao.desconhecida,
          codigoHttp: codigoHttp,
        );
    }
  }

  /// Converte falha de rede/timeout em falha segura.
  FalhaDeAutenticacao interpretarFalhaDeRede(Object erro) {
    return const FalhaDeAutenticacao(
      mensagem: 'Não foi possível conectar ao servidor. Verifique sua conexão com a internet.',
      tipo: TipoDeFalhaDeAutenticacao.redeIndisponivel,
    );
  }
}
