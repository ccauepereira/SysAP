import 'dart:convert';
import 'dart:io';

import '../../aplicacao/configuracao/configuracao_publica.dart';
import '../../dominio/falhas/falha_de_autenticacao.dart';
import 'interpretador_de_erros_da_api.dart';

/// Resposta simplificada e segura retornada pelo [ClienteDaApi].
class RespostaHttp {
  final int status;
  final String corpo;
  final String? requestId;

  const RespostaHttp({
    required this.status,
    required this.corpo,
    this.requestId,
  });

  bool get ehSucesso => status >= 200 && status < 300;

  Map<String, dynamic> get corpoJson {
    if (corpo.trim().isEmpty) return {};
    try {
      final decoded = jsonDecode(corpo);
      if (decoded is Map<String, dynamic>) return decoded;
      return {};
    } catch (_) {
      return {};
    }
  }
}

/// Cliente HTTP central, seguro e testável para comunicação com a API SysAP.
///
/// Implementado diretamente sobre o [HttpClient] nativo do Dart SDK.
class ClienteDaApi {
  final ConfiguracaoPublica configuracao;
  final InterpretadorDeErrosDaApi interpretadorDeErros;
  final HttpClient _httpClient;

  ClienteDaApi({
    required this.configuracao,
    this.interpretadorDeErros = const InterpretadorDeErrosDaApi(),
    HttpClient? customHttpClient,
  }) : _httpClient = customHttpClient ?? HttpClient() {
    _httpClient.connectionTimeout = const Duration(seconds: 10);
  }

  /// Constrói a URI completa validando a origem.
  Uri _construirUri(String caminho) {
    final base = configuracao.urlDaApi;
    final caminhoLimpo = caminho.startsWith('/') ? caminho : '/$caminho';
    return base.replace(path: caminhoLimpo);
  }

  /// Executa requisição POST para a API.
  Future<RespostaHttp> post(
    String caminho, {
    Map<String, dynamic>? corpo,
    String? tokenBearer,
  }) async {
    return _executarRequisicao(
      metodo: 'POST',
      caminho: caminho,
      corpo: corpo,
      tokenBearer: tokenBearer,
    );
  }

  /// Executa requisição GET para a API.
  Future<RespostaHttp> get(String caminho, {String? tokenBearer}) async {
    return _executarRequisicao(
      metodo: 'GET',
      caminho: caminho,
      tokenBearer: tokenBearer,
    );
  }

  Future<RespostaHttp> _executarRequisicao({
    required String metodo,
    required String caminho,
    Map<String, dynamic>? corpo,
    String? tokenBearer,
  }) async {
    final uri = _construirUri(caminho);

    try {
      final HttpClientRequest request;
      if (metodo == 'POST') {
        request = await _httpClient.postUrl(uri);
      } else {
        request = await _httpClient.getUrl(uri);
      }

      request.followRedirects = false;
      request.headers.set(HttpHeaders.acceptHeader, 'application/json');

      if (tokenBearer != null && tokenBearer.isNotEmpty) {
        request.headers.set(
          HttpHeaders.authorizationHeader,
          'Bearer $tokenBearer',
        );
      }

      if (corpo != null) {
        request.headers.set(
          HttpHeaders.contentTypeHeader,
          'application/json; charset=utf-8',
        );
        final jsonBytes = utf8.encode(jsonEncode(corpo));
        request.contentLength = jsonBytes.length;
        request.add(jsonBytes);
      }

      final response = await request.close();
      final responseBody = await utf8.decodeStream(response);
      final requestId = response.headers.value('x-request-id');

      final resposta = RespostaHttp(
        status: response.statusCode,
        corpo: responseBody,
        requestId: requestId,
      );

      if (!resposta.ehSucesso) {
        throw interpretadorDeErros.interpretar(
          codigoHttp: resposta.status,
          corpo: resposta.corpo,
        );
      }

      return resposta;
    } on FalhaDeAutenticacao {
      rethrow;
    } catch (e) {
      throw interpretadorDeErros.interpretarFalhaDeRede(e);
    }
  }

  void fechar() {
    _httpClient.close(force: true);
  }
}
