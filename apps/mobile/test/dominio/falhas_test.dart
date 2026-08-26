import 'package:flutter_test/flutter_test.dart';
import 'package:sysap/dominio/falhas/falha_de_configuracao.dart';
import 'package:sysap/dominio/falhas/falha_do_sysap.dart';

void main() {
  group('Domínio: Falhas', () {
    test(
      'FalhaDeConfiguracao implementa FalhaDoSysAP e preserva mensagem e tipo',
      () {
        const falha = FalhaDeConfiguracao(
          mensagem: 'Erro de validação',
          tipo: TipoDeFalhaDeConfiguracao.urlInvalida,
        );

        expect(falha, isA<FalhaDoSysAP>());
        expect(falha.mensagem, equals('Erro de validação'));
        expect(falha.tipo, equals(TipoDeFalhaDeConfiguracao.urlInvalida));
        expect(falha.toString(), contains('urlInvalida'));
        expect(falha.toString(), contains('Erro de validação'));
      },
    );
  });
}
