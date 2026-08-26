import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:integration_test/integration_test.dart';
import 'package:sysap/aplicacao/configuracao/ambiente.dart';
import 'package:sysap/aplicacao/configuracao/configuracao_publica.dart';
import 'package:sysap/aplicacao/inicializacao/estado_de_inicializacao.dart';
import 'package:sysap/aplicacao/sessao/gerenciador_de_sessao.dart';
import 'package:sysap/apresentacao/autenticacao/tela_de_login.dart';
import 'package:sysap/apresentacao/navegacao/navegacao_por_perfil.dart';
import 'package:sysap/infraestrutura/api/cliente_da_api.dart';
import 'package:sysap/infraestrutura/armazenamento_seguro/armazenamento_seguro_de_sessao.dart';
import 'package:sysap/main.dart';

class ControlledIntegrationApiClient extends ClienteDaApi {
  ControlledIntegrationApiClient()
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
      return const RespostaHttp(
        status: 200,
        corpo: '{"session_id":"s-int","profile_id":"p-int","organization_id":"o-int","role":"owner","access_token":"token-int","refresh_token":"ref-int","expires_in":3600}',
      );
    }
    if (caminho == '/v1/auth/logout') {
      return const RespostaHttp(status: 204, corpo: '');
    }
    return const RespostaHttp(status: 200, corpo: '{}');
  }

  @override
  Future<RespostaHttp> get(String caminho, {String? tokenBearer}) async {
    if (caminho == '/v1/me') {
      return const RespostaHttp(
        status: 200,
        corpo: '{"profile":{"id":"p-int","display_name":"Artur Head Coach"},"memberships":[{"organization_id":"o-int","role":"owner","status":"active"}]}',
      );
    }
    return const RespostaHttp(status: 200, corpo: '{}');
  }
}

void main() {
  IntegrationTestWidgetsFlutterBinding.ensureInitialized();

  testWidgets('Fluxo E2E controlado: Login -> Perfil Confirmado -> Logout', (
    tester,
  ) async {
    final configuracao = ConfiguracaoPublica(
      ambiente: AmbienteDeExecucao.desenvolvimento,
      urlDaApi: Uri.parse('http://127.0.0.1:8080'),
    );

    final fakeApi = ControlledIntegrationApiClient();
    final repositorioSessao = ArmazenamentoSeguroDeSessao();
    final gerenciadorSessao = GerenciadorDeSessao(
      clienteApi: fakeApi,
      repositorioSessao: repositorioSessao,
    );

    await tester.pumpWidget(
      AplicativoSysAP(
        estadoInicial: EstadoPreparado(configuracao),
        gerenciadorDeSessao: gerenciadorSessao,
      ),
    );
    await tester.pumpAndSettle();

    // 1. App exibe tela de login inicial
    expect(find.byType(TelaDeLogin), findsOneWidget);
    expect(find.text('Entrar'), findsOneWidget);

    // 2. Preenche matrícula e senha
    await tester.enterText(find.byType(TextFormField).at(0), '2026000001');
    await tester.enterText(
      find.byType(TextFormField).at(1),
      'senha_valida_123',
    );
    await tester.tap(find.text('Entrar'));
    await tester.pumpAndSettle();

    // 3. Após autenticação e confirmação em /v1/me, navega para Casca da Área Owner
    expect(find.byType(CascaDaAreaOwner), findsOneWidget);
    expect(find.text('Artur Head Coach'), findsOneWidget);

    // 4. Executa Logout
    await tester.tap(find.text('Encerrar Sessão'));
    await tester.pumpAndSettle();

    // 5. Retorna à Tela de Login e cofre é limpo
    expect(find.byType(TelaDeLogin), findsOneWidget);
    expect(await repositorioSessao.lerSessao(), isNull);
  });
}
