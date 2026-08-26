import 'package:flutter_test/flutter_test.dart';
import 'package:sysap/aplicacao/inicializacao/estado_de_inicializacao.dart';
import 'package:sysap/aplicacao/inicializacao/orquestrador_de_inicializacao.dart';
import 'package:sysap/dominio/falhas/falha_de_configuracao.dart';

void main() {
  group('Aplicação: OrquestradorDeInicializacao', () {
    const orquestrador = OrquestradorDeInicializacao();

    test('retorna EstadoPreparado quando recebe configuração válida', () {
      final estado = orquestrador.inicializar(
        urlBruta: 'https://api.sysap.com.br',
        ambienteBruto: 'producao',
        usarVariaveisDeBuild: false,
      );

      expect(estado, isA<EstadoPreparado>());
      final preparado = estado as EstadoPreparado;
      expect(preparado.configuracao.urlDaApi.host, equals('api.sysap.com.br'));
      expect(preparado.configuracao.ambiente.ehProducao, isTrue);
    });

    test('retorna EstadoFalha quando os parâmetros são inválidos', () {
      final estado = orquestrador.inicializar(
        urlBruta: 'http://inseguro.sysap.com.br',
        ambienteBruto: 'producao',
        usarVariaveisDeBuild: false,
      );

      expect(estado, isA<EstadoFalha>());
      final falha = estado as EstadoFalha;
      expect(
        falha.falha.tipo,
        equals(TipoDeFalhaDeConfiguracao.esquemaInseguroEmProducao),
      );
    });

    test(
      'retorna EstadoConfiguracaoPendente quando não há variáveis injetadas',
      () {
        final estado = orquestrador.inicializar(usarVariaveisDeBuild: true);

        expect(estado, isA<EstadoConfiguracaoPendente>());
      },
    );
  });
}
