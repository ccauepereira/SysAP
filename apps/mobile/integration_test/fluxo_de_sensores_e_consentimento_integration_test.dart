import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:integration_test/integration_test.dart';
import 'package:sysap/aplicacao/configuracao/ambiente.dart';
import 'package:sysap/aplicacao/configuracao/configuracao_publica.dart';
import 'package:sysap/aplicacao/consentimento/gerenciador_de_consentimento.dart';
import 'package:sysap/aplicacao/dispositivos/sincronizador_de_dados_de_dispositivo.dart';
import 'package:sysap/aplicacao/inicializacao/estado_de_inicializacao.dart';
import 'package:sysap/aplicacao/sessao/gerenciador_de_sessao.dart';
import 'package:sysap/apresentacao/consentimento/tela_de_gerenciamento_de_consentimento.dart';
import 'package:sysap/apresentacao/navegacao/navegacao_por_perfil.dart';
import 'package:sysap/dominio/entidades/categoria_de_dado_sensivel.dart';
import 'package:sysap/dominio/entidades/consentimento_de_dados.dart';
import 'package:sysap/dominio/entidades/disponibilidade_de_recurso.dart';
import 'package:sysap/dominio/entidades/falha_de_integracao.dart';
import 'package:sysap/dominio/entidades/papel_do_usuario.dart';
import 'package:sysap/dominio/entidades/resumo_de_atividade.dart';
import 'package:sysap/dominio/repositorios/repositorio_de_dispositivos.dart';
import 'package:sysap/dominio/repositorios/repositorio_de_sincronizacao.dart';
import 'package:sysap/dominio/servicos/detector_de_conectividade.dart';
import 'package:sysap/dominio/servicos/fonte_de_dados_de_saude.dart';
import 'package:sysap/infraestrutura/api/cliente_da_api.dart';
import 'package:sysap/infraestrutura/armazenamento_seguro/armazenamento_seguro_de_sessao.dart';
import 'package:sysap/infraestrutura/consentimento/repositorio_de_consentimentos_local.dart';
import 'package:sysap/main.dart';

class AthleteControlledApiClient extends ClienteDaApi {
  AthleteControlledApiClient()
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
        corpo: '{"session_id":"s-atleta","profile_id":"p-atleta","organization_id":"org-1","role":"athlete","access_token":"token-atleta","refresh_token":"ref-atleta","expires_in":3600}',
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
        corpo: '{"profile":{"id":"p-atleta","display_name":"Gabriel Atleta"},"memberships":[{"organization_id":"org-1","role":"athlete","status":"active"}]}',
      );
    }
    return const RespostaHttp(status: 200, corpo: '{}');
  }
}

class FakeIntegrationFonteSaude implements FonteDeDadosDeSaude {
  @override
  String get nomeDaPlataforma => 'health_connect';

  @override
  Future<DisponibilidadeDeRecurso> verificarDisponibilidade() async =>
      DisponibilidadeDeRecurso.disponivel;

  @override
  Future<bool> solicitarPermissoes(
    Set<CategoriaDeDadoSensivel> categorias,
  ) async => true;

  @override
  Future<ResumoDeAtividade?> lerResumoDeTreino({
    required DateTime inicio,
    required DateTime fim,
    required Set<CategoriaDeDadoSensivel> categoriasAutorizadas,
  }) async {
    return ResumoDeAtividade(
      inicio: inicio,
      fim: fim,
      duracao: fim.difference(inicio),
      passos: 8400,
      distanciaMetros: 6800.0,
      origem: nomeDaPlataforma,
    );
  }
}

class FakeIntegrationRepositorioDispositivos
    implements RepositorioDeDispositivos {
  DateTime? ultimaSincronizacao;

  @override
  Future<List<InformacaoDeDispositivo>> listarFontesConectadas() async => [];

  @override
  Future<void> atualizarUltimaSincronizacao({
    required String identificador,
    required DateTime sincronizadoEm,
  }) async {
    ultimaSincronizacao = sincronizadoEm;
  }
}

class FakeIntegrationRepositorioSincronizacao
    implements RepositorioDeSincronizacao {
  final Map<String, EstadoDeSincronizacao> status = {};

  @override
  Future<void> registrarInicio(RegistroDeSincronizacao registro) async {
    status[registro.id] = registro.estado;
  }

  @override
  Future<void> registrarConclusao({
    required String id,
    required EstadoDeSincronizacao estado,
    required DateTime concluidoEm,
  }) async {
    status[id] = estado;
  }

  @override
  Future<List<RegistroDeSincronizacao>> listarPendentes({
    required String atletaId,
    required String organizacaoId,
  }) async => [];
}

class FakeIntegrationDetectorConectividade implements DetectorDeConectividade {
  @override
  Future<EstadoDeConectividade> verificarConectividade() async =>
      EstadoDeConectividade.online;

  @override
  Stream<EstadoDeConectividade> observarConectividade() =>
      Stream.value(EstadoDeConectividade.online);
}

void main() {
  IntegrationTestWidgetsFlutterBinding.ensureInitialized();

  testWidgets(
    'Fluxo Integrado: Login Atleta -> Gestão de Consentimento -> Sincronização -> Revogação -> Isolamento Owner -> Logout',
    (tester) async {
      final configuracao = ConfiguracaoPublica(
        ambiente: AmbienteDeExecucao.desenvolvimento,
        urlDaApi: Uri.parse('http://127.0.0.1:8080'),
      );

      final fakeApi = AthleteControlledApiClient();
      final repositorioSessao = ArmazenamentoSeguroDeSessao();
      final gerenciadorSessao = GerenciadorDeSessao(
        clienteApi: fakeApi,
        repositorioSessao: repositorioSessao,
      );

      // 1. Inicializa app e efetua login
      await tester.pumpWidget(
        AplicativoSysAP(
          estadoInicial: EstadoPreparado(configuracao),
          gerenciadorDeSessao: gerenciadorSessao,
        ),
      );
      await tester.pumpAndSettle();

      await tester.enterText(find.byType(TextFormField).at(0), '2026000002');
      await tester.enterText(
        find.byType(TextFormField).at(1),
        'senha_atleta_123',
      );
      await tester.tap(find.text('Entrar'));
      await tester.pumpAndSettle();

      // 2. Confirma navegação para Área do Atleta
      expect(find.byType(CascaDaAreaAthlete), findsOneWidget);
      expect(find.text('Gabriel Atleta'), findsOneWidget);

      // 3. Abre a tela de Gerenciamento de Consentimento
      await tester.tap(find.text('Gerenciar Consentimento e Sensores'));
      await tester.pumpAndSettle();
      expect(find.byType(TelaDeGerenciamentoDeConsentimento), findsOneWidget);

      // 4. Concede consentimento para Dados Esportivos Básicos
      final switchDados = find.byType(Switch).first;
      await tester.tap(switchDados);
      await tester.pumpAndSettle();

      // 5. Testa o sincronizador de dados com o consentimento ativo
      final repoConsentimento = RepositorioDeConsentimentosLocal();
      final gerenciadorConsentimento = GerenciadorDeConsentimento(
        repositorio: repoConsentimento,
      );
      final repoDispositivos = FakeIntegrationRepositorioDispositivos();
      final repoSincronizacao = FakeIntegrationRepositorioSincronizacao();
      final detectorConectividade = FakeIntegrationDetectorConectividade();
      final fonteSaude = FakeIntegrationFonteSaude();

      final sincronizador = SincronizadorDeDadosDeDispositivo(
        fonteSaude: fonteSaude,
        gerenciadorConsentimento: gerenciadorConsentimento,
        repositorioDispositivos: repoDispositivos,
        repositorioSincronizacao: repoSincronizacao,
        detectorConectividade: detectorConectividade,
      );

      // Registra consentimento no repositório do fluxo
      await repoConsentimento.salvarConsentimento(
        ConsentimentoDeDados.criarNovo(
          id: 'cons-integ',
          atletaId: 'p-atleta',
          organizacaoId: 'org-1',
          categoria: CategoriaDeDadoSensivel.dadosEsportivosBasicos,
          finalidade: 'Passos e distância',
          versaoDoTermo: '1.0',
          dataDeConcessao: DateTime.now().toUtc(),
        ),
      );

      final inicio = DateTime.now().toUtc().subtract(const Duration(hours: 1));
      final fim = DateTime.now().toUtc();

      final resumo = await sincronizador.sincronizarTreino(
        atletaId: 'p-atleta',
        organizacaoId: 'org-1',
        sincronizacaoId: 'sync-int-1',
        inicio: inicio,
        fim: fim,
      );

      expect(resumo, isNotNull);
      expect(resumo!.passos, equals(8400));
      expect(resumo.distanciaMetros, equals(6800.0));
      expect(
        repoSincronizacao.status['sync-int-1'],
        equals(EstadoDeSincronizacao.concluida),
      );

      // 6. Revoga o consentimento
      await repoConsentimento.revogarConsentimento(
        atletaId: 'p-atleta',
        organizacaoId: 'org-1',
        categoria: CategoriaDeDadoSensivel.dadosEsportivosBasicos,
        dataDaRevogacao: DateTime.now().toUtc(),
      );

      // 7. Confirma que nova sincronização é bloqueada imediatamente
      expect(
        () => sincronizador.sincronizarTreino(
          atletaId: 'p-atleta',
          organizacaoId: 'org-1',
          sincronizacaoId: 'sync-int-2',
          inicio: inicio,
          fim: fim,
        ),
        throwsA(isA<FalhaDeConsentimentoAusente>()),
      );

      // 8. Confirma que Owner não consegue acessar os dados sem consentimento
      expect(
        () => gerenciadorConsentimento.validarAutorizacaoDeAcesso(
          solicitanteId: 'owner-id',
          solicitantePapel: PapelDoUsuario.owner,
          solicitanteOrganizacaoId: 'org-1',
          solicitanteVinculoAtivo: true,
          alvoAtletaId: 'p-atleta',
          alvoOrganizacaoId: 'org-1',
          categoria: CategoriaDeDadoSensivel.dadosEsportivosBasicos,
        ),
        throwsA(isA<FalhaDeAutorizacaoOwner>()),
      );

      // 9. Volta da tela de consentimento e encerra sessão
      Navigator.of(
        tester.element(find.byType(TelaDeGerenciamentoDeConsentimento)),
      ).pop();
      await tester.pumpAndSettle();

      await tester.tap(find.text('Encerrar Sessão'));
      await tester.pumpAndSettle();

      // 10. Sessão limpa com segurança
      expect(await repositorioSessao.lerSessao(), isNull);
    },
  );
}
