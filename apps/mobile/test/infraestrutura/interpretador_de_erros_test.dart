import 'package:flutter_test/flutter_test.dart';
import 'package:sysap/dominio/falhas/falha_de_autenticacao.dart';
import 'package:sysap/infraestrutura/api/interpretador_de_erros_da_api.dart';

void main() {
  group('Infraestrutura: InterpretadorDeErrosDaApi', () {
    const interpretador = InterpretadorDeErrosDaApi();

    test('mapeia 401 para credenciais inválidas de modo seguro', () {
      final falha = interpretador.interpretar(
        codigoHttp: 401,
        corpo: '{"error":{"code":"invalid_credentials","message":"invalid credentials"}}',
      );

      expect(
        falha.tipo,
        equals(TipoDeFalhaDeAutenticacao.credenciaisInvalidas),
      );
      expect(falha.codigoHttp, equals(401));
      expect(falha.mensagem, contains('inválida'));
    });

    test('mapeia 403 com mfa_required para mfaRequerido', () {
      final falha = interpretador.interpretar(
        codigoHttp: 403,
        corpo: '{"error":{"code":"mfa_required","message":"mfa required"}}',
      );

      expect(falha.tipo, equals(TipoDeFalhaDeAutenticacao.mfaRequerido));
      expect(falha.codigoHttp, equals(403));
    });

    test('mapeia 403 genérico para acesso negado', () {
      final falha = interpretador.interpretar(
        codigoHttp: 403,
        corpo:
            '{"error":{"code":"access_denied","message":"access is denied"}}',
      );

      expect(falha.tipo, equals(TipoDeFalhaDeAutenticacao.acessoNegado));
      expect(falha.codigoHttp, equals(403));
    });

    test('mapeia 422 para formato inválido', () {
      final falha = interpretador.interpretar(
        codigoHttp: 422,
        corpo: '{"error":{"code":"validation_failed"}}',
      );

      expect(falha.tipo, equals(TipoDeFalhaDeAutenticacao.formatoInvalido));
      expect(falha.codigoHttp, equals(422));
    });

    test('mapeia 503 para serviço indisponível', () {
      final falha = interpretador.interpretar(
        codigoHttp: 503,
        corpo: '{"error":{"code":"service_unavailable"}}',
      );

      expect(falha.tipo, equals(TipoDeFalhaDeAutenticacao.servicoIndisponivel));
      expect(falha.codigoHttp, equals(503));
    });

    test('interpretarFalhaDeRede retorna falha de rede clara', () {
      final falha = interpretador.interpretarFalhaDeRede(
        Exception('SocketException'),
      );
      expect(falha.tipo, equals(TipoDeFalhaDeAutenticacao.redeIndisponivel));
      expect(falha.mensagem, contains('conexão'));
    });
  });
}
