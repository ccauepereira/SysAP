import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:sysap/aplicacao/inicializacao/estado_de_inicializacao.dart';
import 'package:sysap/apresentacao/inicializacao/tela_de_inicializacao.dart';
import 'package:sysap/apresentacao/tema/tema_sysap.dart';

void main() {
  group('Apresentação: TemaSysAP', () {
    test('tema claro e tema escuro utilizam Material 3', () {
      expect(TemaSysAP.temaClaro.useMaterial3, isTrue);
      expect(TemaSysAP.temaEscuro.useMaterial3, isTrue);
      expect(
        TemaSysAP.temaEscuro.scaffoldBackgroundColor,
        equals(TemaSysAP.corFundoObsidiana),
      );
      expect(TemaSysAP.corDouradoPrincipal, equals(const Color(0xFFD4AE29)));
    });

    testWidgets('o aplicativo aplica tema base sem quebrar a inicialização', (
      tester,
    ) async {
      await tester.pumpWidget(
        MaterialApp(
          theme: TemaSysAP.temaClaro,
          darkTheme: TemaSysAP.temaEscuro,
          themeMode: ThemeMode.dark,
          home: const TelaDeInicializacao(estado: EstadoInicializando()),
        ),
      );

      expect(find.byType(Scaffold), findsOneWidget);
      expect(find.text('SysAP'), findsOneWidget);
    });
  });
}
