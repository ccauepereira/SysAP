import 'package:flutter_test/flutter_test.dart';
import 'package:integration_test/integration_test.dart';
import 'package:sysap/aplicacao/configuracao/configuracao_publica.dart';
import 'package:sysap/aplicacao/inicializacao/estado_de_inicializacao.dart';
import 'package:sysap/main.dart';

void main() {
  IntegrationTestWidgetsFlutterBinding.ensureInitialized();

  group('Integração: Fluxo de Inicialização', () {
    testWidgets(
      'inicia o aplicativo com configuração controlada e exibe estado preparado',
      (tester) async {
        final config = ConfiguracaoPublica.validar(
          urlBruta: 'http://127.0.0.1:8080',
          ambienteBruto: 'desenvolvimento',
        );

        await tester.pumpWidget(
          AplicativoSysAP(estadoInicial: EstadoPreparado(config)),
        );
        await tester.pumpAndSettle();

        expect(find.text('SysAP'), findsOneWidget);
        expect(find.text('SysAP Preparado'), findsOneWidget);
        expect(find.textContaining('Desenvolvimento'), findsOneWidget);
      },
    );
  });
}
