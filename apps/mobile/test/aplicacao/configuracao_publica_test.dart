import 'package:flutter_test/flutter_test.dart';
import 'package:sysap/aplicacao/configuracao/ambiente.dart';
import 'package:sysap/aplicacao/configuracao/configuracao_publica.dart';
import 'package:sysap/dominio/falhas/falha_de_configuracao.dart';

void main() {
  group('Aplicação: ConfiguracaoPublica', () {
    test(
      '1. aceita configuração válida de desenvolvimento local com HTTP e porta',
      () {
        final config = ConfiguracaoPublica.validar(
          urlBruta: 'http://127.0.0.1:8080',
          ambienteBruto: 'desenvolvimento',
        );

        expect(config.ambiente, equals(AmbienteDeExecucao.desenvolvimento));
        expect(config.urlDaApi.scheme, equals('http'));
        expect(config.urlDaApi.host, equals('127.0.0.1'));
        expect(config.urlDaApi.port, equals(8080));
      },
    );

    test('2. aceita configuração válida de produção com HTTPS', () {
      final config = ConfiguracaoPublica.validar(
        urlBruta: 'https://api.sysap.com.br',
        ambienteBruto: 'producao',
      );

      expect(config.ambiente, equals(AmbienteDeExecucao.producao));
      expect(config.urlDaApi.scheme, equals('https'));
      expect(config.urlDaApi.host, equals('api.sysap.com.br'));
    });

    test('3. rejeita URL ausente ou em branco', () {
      expect(
        () => ConfiguracaoPublica.validar(
          urlBruta: null,
          ambienteBruto: 'desenvolvimento',
        ),
        throwsA(
          isA<FalhaDeConfiguracao>().having(
            (f) => f.tipo,
            'tipo',
            equals(TipoDeFalhaDeConfiguracao.urlAusente),
          ),
        ),
      );

      expect(
        () => ConfiguracaoPublica.validar(
          urlBruta: '   ',
          ambienteBruto: 'desenvolvimento',
        ),
        throwsA(
          isA<FalhaDeConfiguracao>().having(
            (f) => f.tipo,
            'tipo',
            equals(TipoDeFalhaDeConfiguracao.urlAusente),
          ),
        ),
      );
    });

    test('4. rejeita URL malformada, sem host ou com esquema inválido', () {
      expect(
        () => ConfiguracaoPublica.validar(
          urlBruta: 'ftp://api.sysap.com.br',
          ambienteBruto: 'desenvolvimento',
        ),
        throwsA(
          isA<FalhaDeConfiguracao>().having(
            (f) => f.tipo,
            'tipo',
            equals(TipoDeFalhaDeConfiguracao.urlInvalida),
          ),
        ),
      );

      expect(
        () => ConfiguracaoPublica.validar(
          urlBruta: 'not-a-valid-url',
          ambienteBruto: 'desenvolvimento',
        ),
        throwsA(
          isA<FalhaDeConfiguracao>().having(
            (f) => f.tipo,
            'tipo',
            equals(TipoDeFalhaDeConfiguracao.urlInvalida),
          ),
        ),
      );
    });

    test('5. rejeita URL de produção com HTTP inseguro', () {
      expect(
        () => ConfiguracaoPublica.validar(
          urlBruta: 'http://api.sysap.com.br',
          ambienteBruto: 'producao',
        ),
        throwsA(
          isA<FalhaDeConfiguracao>().having(
            (f) => f.tipo,
            'tipo',
            equals(TipoDeFalhaDeConfiguracao.esquemaInseguroEmProducao),
          ),
        ),
      );
    });

    test('6. rejeita credenciais embutidas na URL da API', () {
      expect(
        () => ConfiguracaoPublica.validar(
          urlBruta: 'https://usuario:senha123@api.sysap.com.br',
          ambienteBruto: 'producao',
        ),
        throwsA(
          isA<FalhaDeConfiguracao>().having(
            (f) => f.tipo,
            'tipo',
            equals(TipoDeFalhaDeConfiguracao.credenciaisEmbutidasProibidas),
          ),
        ),
      );
    });

    test('7. garante ausência de segredos e representação textual segura', () {
      final config = ConfiguracaoPublica.validar(
        urlBruta: 'https://api.sysap.com.br/v1',
        ambienteBruto: 'producao',
      );

      final texto = config.toString();
      expect(texto, contains('Produção'));
      expect(texto, contains('https://api.sysap.com.br'));
      expect(texto, isNot(contains('secret')));
      expect(texto, isNot(contains('token')));
      expect(texto, isNot(contains('password')));
    });
  });
}
