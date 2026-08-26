import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:sysap/aplicacao/configuracao/ambiente.dart';
import 'package:sysap/aplicacao/configuracao/configuracao_publica.dart';
import 'package:sysap/aplicacao/sessao/gerenciador_de_sessao.dart';
import 'package:sysap/apresentacao/autenticacao/tela_de_login.dart';
import 'package:sysap/apresentacao/tema/tema_sysap.dart';
import 'package:sysap/dominio/falhas/falha_de_autenticacao.dart';
import 'package:sysap/infraestrutura/api/cliente_da_api.dart';
import 'package:sysap/infraestrutura/armazenamento_seguro/armazenamento_seguro_de_sessao.dart';

class FakeClienteLogin extends ClienteDaApi {
  bool failNext = false;
  int loginCalls = 0;

  FakeClienteLogin()
    : super(
        configuracao: ConfiguracaoPublica(
          ambiente: AmbienteDeExecucao.desenvolvimento,
          urlDaApi: Uri.parse('http://127.0.0.1:8080'),
        ),
      );

  @override
  Future<RespostaHttp> post(
    String caminho, {
    Map<String, dynamic>? corpo,
    String? tokenBearer,
  }) async {
    if (caminho == '/v1/auth/login') {
      loginCalls++;
      if (failNext) {
        throw const FalhaDeAutenticacao(
          mensagem: 'Matrícula ou senha inválida.',
          tipo: TipoDeFalhaDeAutenticacao.credenciaisInvalidas,
          codigoHttp: 401,
        );
      }
      return const RespostaHttp(
        status: 200,
        corpo: '{"session_id":"s1","profile_id":"p1","organization_id":"o1","role":"owner","access_token":"at1","refresh_token":"rt1","expires_in":3600}',
      );
    }
    return const RespostaHttp(status: 200, corpo: '{}');
  }

  @override
  Future<RespostaHttp> get(String caminho, {String? tokenBearer}) async {
    if (caminho == '/v1/me') {
      return const RespostaHttp(
        status: 200,
        corpo: '{"profile":{"id":"p1","display_name":"Artur"},"memberships":[{"organization_id":"o1","role":"owner","status":"active"}]}',
      );
    }
    return const RespostaHttp(status: 200, corpo: '{}');
  }
}

void main() {
  group('Apresentação: TelaDeLogin', () {
    late FakeClienteLogin fakeApi;
    late GerenciadorDeSessao gerenciador;

    setUp(() {
      fakeApi = FakeClienteLogin();
      gerenciador = GerenciadorDeSessao(
        clienteApi: fakeApi,
        repositorioSessao: ArmazenamentoSeguroDeSessao(),
        estadoInicial: const SessaoNaoAutenticada(),
      );
    });

    Widget criarTelaDeTeste() {
      return MaterialApp(
        theme: TemaSysAP.temaEscuro,
        home: ListenableBuilder(
          listenable: gerenciador,
          builder: (context, _) {
            return TelaDeLogin(gerenciadorDeSessao: gerenciador);
          },
        ),
      );
    }

    testWidgets('renderiza campos de matrícula, senha e botão entrar', (
      tester,
    ) async {
      await tester.pumpWidget(criarTelaDeTeste());

      expect(find.byType(TextFormField), findsNWidgets(2));
      expect(find.text('Matrícula'), findsOneWidget);
      expect(find.text('Senha'), findsOneWidget);
      expect(find.text('Entrar'), findsOneWidget);
      expect(find.text('Ativar conta'), findsOneWidget);
      expect(find.text('Esqueci a senha'), findsOneWidget);
    });

    testWidgets('valida campos obrigatórios antes de submeter', (tester) async {
      await tester.pumpWidget(criarTelaDeTeste());

      await tester.tap(find.text('Entrar'));
      await tester.pumpAndSettle();

      expect(find.text('Informe sua matrícula.'), findsOneWidget);
      expect(fakeApi.loginCalls, equals(0));
    });

    testWidgets('exibe mensagem segura quando as credenciais falham', (
      tester,
    ) async {
      fakeApi.failNext = true;
      await tester.pumpWidget(criarTelaDeTeste());

      await tester.enterText(find.byType(TextFormField).at(0), '2026000001');
      await tester.enterText(
        find.byType(TextFormField).at(1),
        'senha_errada_123',
      );
      await tester.tap(find.text('Entrar'));
      await tester.pumpAndSettle();

      expect(find.text('Matrícula ou senha inválida.'), findsOneWidget);
      expect(fakeApi.loginCalls, equals(1));
    });

    testWidgets('permite alternar visibilidade da senha', (tester) async {
      await tester.pumpWidget(criarTelaDeTeste());

      final toggleButton = find.byType(IconButton).first;
      await tester.tap(toggleButton);
      await tester.pumpAndSettle();

      expect(find.byIcon(Icons.visibility_off_outlined), findsOneWidget);
    });
  });
}
