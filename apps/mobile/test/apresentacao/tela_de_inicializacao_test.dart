import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:sysap/aplicacao/configuracao/configuracao_publica.dart';
import 'package:sysap/aplicacao/inicializacao/estado_de_inicializacao.dart';
import 'package:sysap/apresentacao/inicializacao/tela_de_inicializacao.dart';
import 'package:sysap/apresentacao/tema/tema_sysap.dart';
import 'package:sysap/dominio/falhas/falha_de_configuracao.dart';

Widget _envolverWidget(Widget filho) {
  return MaterialApp(
    theme: TemaSysAP.temaClaro,
    darkTheme: TemaSysAP.temaEscuro,
    home: filho,
  );
}

void main() {
  group('Apresentação: TelaDeInicializacao', () {
    testWidgets('1. inicialização exibe estado neutro de carregamento', (
      tester,
    ) async {
      await tester.pumpWidget(
        _envolverWidget(
          const TelaDeInicializacao(estado: EstadoInicializando()),
        ),
      );

      expect(find.byType(CircularProgressIndicator), findsOneWidget);
      expect(find.text('Inicializando o aplicativo...'), findsOneWidget);
    });

    testWidgets(
      '2. configuração pendente exibe mensagem clara sem dados sensíveis',
      (tester) async {
        await tester.pumpWidget(
          _envolverWidget(
            const TelaDeInicializacao(
              estado: EstadoConfiguracaoPendente(
                'Configuração pública pendente para desenvolvimento.',
              ),
            ),
          ),
        );

        expect(find.text('Configuração Pendente'), findsOneWidget);
        expect(
          find.text('Configuração pública pendente para desenvolvimento.'),
          findsOneWidget,
        );
        expect(find.textContaining('token'), findsNothing);
        expect(find.textContaining('password'), findsNothing);
      },
    );

    testWidgets('3. configuração inválida apresenta mensagem segura de falha', (
      tester,
    ) async {
      await tester.pumpWidget(
        _envolverWidget(
          const TelaDeInicializacao(
            estado: EstadoFalha(
              FalhaDeConfiguracao(
                mensagem: 'URL da API é inválida.',
                tipo: TipoDeFalhaDeConfiguracao.urlInvalida,
              ),
            ),
          ),
        ),
      );

      expect(find.text('Falha na Inicialização'), findsOneWidget);
      expect(find.text('URL da API é inválida.'), findsOneWidget);
    });

    testWidgets('4. estado preparado exibe ambiente ativo com segurança', (
      tester,
    ) async {
      final config = ConfiguracaoPublica.validar(
        urlBruta: 'http://127.0.0.1:8080',
        ambienteBruto: 'desenvolvimento',
      );

      await tester.pumpWidget(
        _envolverWidget(TelaDeInicializacao(estado: EstadoPreparado(config))),
      );

      expect(find.text('SysAP Preparado'), findsOneWidget);
      expect(find.textContaining('Desenvolvimento'), findsOneWidget);
    });
  });
}
