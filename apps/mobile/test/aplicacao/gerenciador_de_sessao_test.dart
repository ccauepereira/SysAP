import 'dart:convert';

import 'package:flutter_test/flutter_test.dart';
import 'package:sysap/aplicacao/configuracao/ambiente.dart';
import 'package:sysap/aplicacao/configuracao/configuracao_publica.dart';
import 'package:sysap/aplicacao/sessao/gerenciador_de_sessao.dart';
import 'package:sysap/dominio/entidades/papel_do_usuario.dart';
import 'package:sysap/dominio/entidades/sessao_autenticada.dart';
import 'package:sysap/dominio/falhas/falha_de_autenticacao.dart';
import 'package:sysap/infraestrutura/api/cliente_da_api.dart';
import 'package:sysap/infraestrutura/armazenamento_seguro/armazenamento_seguro_de_sessao.dart';

/// Double controlado de teste para o ClienteDaApi sem requisições reais de rede.
class FakeClienteDaApi extends ClienteDaApi {
  Map<String, dynamic>? respostaLogin;
  Map<String, dynamic>? respostaMe;
  Map<String, dynamic>? respostaRefresh;
  bool simularErro401 = false;
  int chamadasRefresh = 0;
  int chamadasLogout = 0;

  FakeClienteDaApi()
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
    if (simularErro401) {
      throw const FalhaDeAutenticacao(
        mensagem: 'Matrícula ou senha inválida.',
        tipo: TipoDeFalhaDeAutenticacao.credenciaisInvalidas,
        codigoHttp: 401,
      );
    }

    if (caminho == '/v1/auth/login') {
      final mapa =
          respostaLogin ??
          {
            'session_id': 'sess-123',
            'profile_id': 'prof-123',
            'organization_id': 'org-123',
            'role': 'owner',
            'access_token': 'fake-access-token',
            'refresh_token': 'fake-refresh-token',
            'expires_in': 3600,
          };
      return RespostaHttp(status: 200, corpo: jsonEncode(mapa));
    }

    if (caminho == '/v1/auth/refresh') {
      chamadasRefresh++;
      final mapa =
          respostaRefresh ??
          {
            'access_token': 'new-access-token',
            'refresh_token': 'new-refresh-token',
            'expires_in': 3600,
          };
      return RespostaHttp(status: 200, corpo: jsonEncode(mapa));
    }

    if (caminho == '/v1/auth/logout') {
      chamadasLogout++;
      return const RespostaHttp(status: 204, corpo: '');
    }

    return const RespostaHttp(status: 200, corpo: '{}');
  }

  @override
  Future<RespostaHttp> get(String caminho, {String? tokenBearer}) async {
    if (simularErro401) {
      throw const FalhaDeAutenticacao(
        mensagem: 'Sessão revogada.',
        tipo: TipoDeFalhaDeAutenticacao.sessaoRevogada,
        codigoHttp: 401,
      );
    }

    if (caminho == '/v1/me') {
      final mapa =
          respostaMe ??
          {
            'profile': {'id': 'prof-123', 'display_name': 'Artur Performance'},
            'memberships': [
              {
                'organization_id': 'org-123',
                'role': 'owner',
                'status': 'active',
              },
            ],
          };
      return RespostaHttp(status: 200, corpo: jsonEncode(mapa));
    }

    return const RespostaHttp(status: 200, corpo: '{}');
  }
}

void main() {
  group('Aplicação: GerenciadorDeSessao', () {
    late FakeClienteDaApi fakeClienteApi;
    late ArmazenamentoSeguroDeSessao repositorioSessao;
    late GerenciadorDeSessao gerenciador;

    setUp(() {
      fakeClienteApi = FakeClienteDaApi();
      repositorioSessao = ArmazenamentoSeguroDeSessao();
      gerenciador = GerenciadorDeSessao(
        clienteApi: fakeClienteApi,
        repositorioSessao: repositorioSessao,
      );
    });

    test(
      'inicialização com cofre vazio transiciona para SessaoNaoAutenticada',
      () async {
        await gerenciador.inicializar();
        expect(gerenciador.value, isA<SessaoNaoAutenticada>());
      },
    );

    test('rejeita login com matrícula fora do padrão de 10 dígitos', () async {
      await gerenciador.login(matricula: '123', senha: 'uma_senha_valida_123');
      expect(gerenciador.value, isA<SessaoFalha>());
      final estado = gerenciador.value as SessaoFalha;
      expect(
        estado.falha.tipo,
        equals(TipoDeFalhaDeAutenticacao.formatoInvalido),
      );
    });

    test('executa login com sucesso e transiciona para SessaoAtiva', () async {
      await gerenciador.login(
        matricula: '2026000001',
        senha: 'senha_segura_de_teste_123',
      );

      expect(gerenciador.value, isA<SessaoAtiva>());
      final ativa = gerenciador.value as SessaoAtiva;
      expect(ativa.sessao.papel, equals(PapelDoUsuario.owner));
      expect(ativa.identidade.nomeDeExibicao, equals('Artur Performance'));
      expect(ativa.identidade.papelPrincipal.ehOwner, isTrue);

      final sessaoSalva = await repositorioSessao.lerSessao();
      expect(sessaoSalva, isNotNull);
      expect(sessaoSalva!.tokenDeAcesso, equals('fake-access-token'));
    });

    test(
      'falha de credenciais no login limpa cofre e emite SessaoFalha',
      () async {
        fakeClienteApi.simularErro401 = true;

        await gerenciador.login(
          matricula: '2026000001',
          senha: 'senha_incorreta_123',
        );

        expect(gerenciador.value, isA<SessaoFalha>());
        final sessaoSalva = await repositorioSessao.lerSessao();
        expect(sessaoSalva, isNull);
      },
    );

    test('bloqueia login de usuário com membership suspensa', () async {
      fakeClienteApi.respostaMe = {
        'profile': {'id': 'prof-99', 'display_name': 'Atleta Inativo'},
        'memberships': [
          {
            'organization_id': 'org-99',
            'role': 'athlete',
            'status': 'suspended',
          },
        ],
      };

      await gerenciador.login(
        matricula: '2026000099',
        senha: 'senha_do_atleta_123',
      );

      expect(gerenciador.value, isA<SessaoAcessoNegado>());
    });

    test('logout limpa armazenamento local mesmo offline', () async {
      await gerenciador.login(
        matricula: '2026000001',
        senha: 'senha_segura_de_teste_123',
      );
      expect(gerenciador.value, isA<SessaoAtiva>());

      await gerenciador.logout();

      expect(gerenciador.value, isA<SessaoNaoAutenticada>());
      expect(await repositorioSessao.lerSessao(), isNull);
      expect(fakeClienteApi.chamadasLogout, equals(1));
    });

    test(
      'renovação coordenada de sessão deduplica requisições concorrentes',
      () async {
        final sessaoOriginal = SessaoAutenticada(
          tokenDeAcesso: 'old-acc',
          tokenDeAtualizacao: 'old-ref',
          idDaSessao: 'sess-1',
          idDoPerfil: 'prof-1',
          idDaOrganizacao: 'org-1',
          papel: PapelDoUsuario.athlete,
          expiraEmSegundos: 3600,
          criadoEm: DateTime.now().toUtc(),
        );
        await repositorioSessao.salvarSessao(sessaoOriginal);

        // Dispara duas renovações simultâneas
        final resultados = await Future.wait([
          gerenciador.renovarSessaoCoordenada(),
          gerenciador.renovarSessaoCoordenada(),
        ]);

        expect(resultados[0], isTrue);
        expect(resultados[1], isTrue);
        expect(
          fakeClienteApi.chamadasRefresh,
          equals(1),
        ); // Exatamente UMA chamada à API

        final sessaoAtualizada = await repositorioSessao.lerSessao();
        expect(sessaoAtualizada!.tokenDeAcesso, equals('new-access-token'));
      },
    );
  });
}
