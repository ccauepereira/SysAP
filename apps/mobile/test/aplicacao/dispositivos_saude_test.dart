import 'package:flutter/services.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:sysap/aplicacao/consentimento/gerenciador_de_consentimento.dart';
import 'package:sysap/aplicacao/dispositivos/sincronizador_de_dados_de_dispositivo.dart';
import 'package:sysap/dominio/entidades/categoria_de_dado_sensivel.dart';
import 'package:sysap/dominio/entidades/consentimento_de_dados.dart';
import 'package:sysap/dominio/entidades/disponibilidade_de_recurso.dart';
import 'package:sysap/dominio/entidades/falha_de_integracao.dart';
import 'package:sysap/dominio/entidades/resumo_de_atividade.dart';
import 'package:sysap/dominio/repositorios/repositorio_de_dispositivos.dart';
import 'package:sysap/dominio/repositorios/repositorio_de_sincronizacao.dart';
import 'package:sysap/dominio/servicos/detector_de_conectividade.dart';
import 'package:sysap/dominio/servicos/fonte_de_dados_de_saude.dart';
import 'package:sysap/infraestrutura/consentimento/repositorio_de_consentimentos_local.dart';
import 'package:sysap/infraestrutura/saude/canal_de_plataforma_de_saude.dart';
import 'package:sysap/infraestrutura/saude/fonte_health_connect.dart';
import 'package:sysap/infraestrutura/saude/fonte_healthkit.dart';

class FakeRepositorioDispositivos implements RepositorioDeDispositivos {
  final Map<String, DateTime> sincronizacoes = {};

  @override
  Future<List<InformacaoDeDispositivo>> listarFontesConectadas() async => [];

  @override
  Future<void> atualizarUltimaSincronizacao({
    required String identificador,
    required DateTime sincronizadoEm,
  }) async {
    sincronizacoes[identificador] = sincronizadoEm;
  }
}

class FakeRepositorioSincronizacao implements RepositorioDeSincronizacao {
  final List<RegistroDeSincronizacao> registros = [];
  final Map<String, EstadoDeSincronizacao> status = {};

  @override
  Future<void> registrarInicio(RegistroDeSincronizacao registro) async {
    registros.add(registro);
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

class FakeDetectorConectividade implements DetectorDeConectividade {
  EstadoDeConectividade estadoAtual = EstadoDeConectividade.online;

  @override
  Future<EstadoDeConectividade> verificarConectividade() async => estadoAtual;

  @override
  Stream<EstadoDeConectividade> observarConectividade() =>
      Stream.value(estadoAtual);
}

class FakeFonteSaude implements FonteDeDadosDeSaude {
  @override
  String nomeDaPlataforma = 'fake_health';

  DisponibilidadeDeRecurso disponibilidade =
      DisponibilidadeDeRecurso.disponivel;
  bool permissaoConcedida = true;
  ResumoDeAtividade? proximoResumo;

  @override
  Future<DisponibilidadeDeRecurso> verificarDisponibilidade() async =>
      disponibilidade;

  @override
  Future<bool> solicitarPermissoes(
    Set<CategoriaDeDadoSensivel> categorias,
  ) async => permissaoConcedida;

  @override
  Future<ResumoDeAtividade?> lerResumoDeTreino({
    required DateTime inicio,
    required DateTime fim,
    required Set<CategoriaDeDadoSensivel> categoriasAutorizadas,
  }) async => proximoResumo;
}

void main() {
  TestWidgetsFlutterBinding.ensureInitialized();

  group('Domínio — ResumoDeAtividade', () {
    test('validações de limites rejeitam dados inválidos', () {
      final inicio = DateTime.utc(2026, 8, 27, 10, 0);
      final fim = DateTime.utc(2026, 8, 27, 11, 0);

      // Fim antes de início
      expect(
        () => ResumoDeAtividade(
          inicio: fim,
          fim: inicio,
          duracao: const Duration(minutes: 60),
          origem: 'teste',
        ),
        throwsArgumentError,
      );

      // Duração negativa
      expect(
        () => ResumoDeAtividade(
          inicio: inicio,
          fim: fim,
          duracao: const Duration(minutes: -10),
          origem: 'teste',
        ),
        throwsArgumentError,
      );

      // Passos negativos
      expect(
        () => ResumoDeAtividade(
          inicio: inicio,
          fim: fim,
          duracao: const Duration(minutes: 60),
          passos: -5,
          origem: 'teste',
        ),
        throwsArgumentError,
      );
    });

    test('filtrarPorConsentimento omite dados não consentidos', () {
      final resumo = ResumoDeAtividade(
        inicio: DateTime.utc(2026, 8, 27, 10, 0),
        fim: DateTime.utc(2026, 8, 27, 11, 0),
        duracao: const Duration(minutes: 60),
        passos: 5000,
        distanciaMetros: 4200.5,
        frequenciaCardiacaMediaBpm: 145,
        frequenciaCardiacaMaximaBpm: 178,
        origem: 'health_connect',
      );

      // Apenas dados esportivos básicos (sem FC)
      final semFC = resumo.filtrarPorConsentimento(
        consentiuDadosEsportivos: true,
        consentiuFrequenciaCardiaca: false,
      );
      expect(semFC.passos, equals(5000));
      expect(semFC.distanciaMetros, equals(4200.5));
      expect(semFC.frequenciaCardiacaMediaBpm, isNull);
      expect(semFC.frequenciaCardiacaMaximaBpm, isNull);

      // Sem nenhum consentimento
      final semNada = resumo.filtrarPorConsentimento(
        consentiuDadosEsportivos: false,
        consentiuFrequenciaCardiaca: false,
      );
      expect(semNada.passos, isNull);
      expect(semNada.distanciaMetros, isNull);
      expect(semNada.frequenciaCardiacaMediaBpm, isNull);
    });
  });

  group('Infraestrutura — Fontes Nativas (Health Connect & HealthKit)', () {
    test(
      'FonteHealthKit trata ausência de dados como retorno nulo seguro',
      () async {
        final fonte = FonteHealthKit(
          CanalDePlataformaDeSaude(
            const MethodChannel('com.sysap.mobile/saude_fake_vazio'),
          ),
        );

        final resultado = await fonte.lerResumoDeTreino(
          inicio: DateTime.utc(2026, 8, 27, 10),
          fim: DateTime.utc(2026, 8, 27, 11),
          categoriasAutorizadas: {
            CategoriaDeDadoSensivel.dadosEsportivosBasicos,
          },
        );

        expect(resultado, isNull);
      },
    );

    test(
      'FonteHealthConnect responde com estado adequado de disponibilidade',
      () async {
        final fonte = FonteHealthConnect(
          CanalDePlataformaDeSaude(
            const MethodChannel('com.sysap.mobile/saude_fake_disponivel'),
          ),
        );

        final status = await fonte.verificarDisponibilidade();
        expect(status, equals(DisponibilidadeDeRecurso.indisponivel));
      },
    );
  });

  group('Aplicação — SincronizadorDeDadosDeDispositivo', () {
    late FakeFonteSaude fakeFonte;
    late RepositorioDeConsentimentosLocal repoConsentimento;
    late GerenciadorDeConsentimento gerenciadorConsentimento;
    late FakeRepositorioDispositivos repoDispositivos;
    late FakeRepositorioSincronizacao repoSincronizacao;
    late FakeDetectorConectividade fakeConectividade;
    late SincronizadorDeDadosDeDispositivo sincronizador;

    const atletaId = 'atleta-101';
    const orgId = 'org-sysap';

    setUp(() {
      fakeFonte = FakeFonteSaude();
      repoConsentimento = RepositorioDeConsentimentosLocal();
      gerenciadorConsentimento = GerenciadorDeConsentimento(
        repositorio: repoConsentimento,
      );
      repoDispositivos = FakeRepositorioDispositivos();
      repoSincronizacao = FakeRepositorioSincronizacao();
      fakeConectividade = FakeDetectorConectividade();

      sincronizador = SincronizadorDeDadosDeDispositivo(
        fonteSaude: fakeFonte,
        gerenciadorConsentimento: gerenciadorConsentimento,
        repositorioDispositivos: repoDispositivos,
        repositorioSincronizacao: repoSincronizacao,
        detectorConectividade: fakeConectividade,
      );
    });

    test(
      'sincronização falha de forma segura quando dispositivo está offline',
      () async {
        fakeConectividade.estadoAtual = EstadoDeConectividade.offline;

        expect(
          () => sincronizador.sincronizarTreino(
            atletaId: atletaId,
            organizacaoId: orgId,
            sincronizacaoId: 'sync-01',
            inicio: DateTime.utc(2026, 8, 27, 8),
            fim: DateTime.utc(2026, 8, 27, 9),
          ),
          throwsA(isA<FalhaDeConectividade>()),
        );
      },
    );

    test(
      'sincronização falha de forma segura sem consentimento ativo no SysAP',
      () async {
        expect(
          () => sincronizador.sincronizarTreino(
            atletaId: atletaId,
            organizacaoId: orgId,
            sincronizacaoId: 'sync-02',
            inicio: DateTime.utc(2026, 8, 27, 8),
            fim: DateTime.utc(2026, 8, 27, 9),
          ),
          throwsA(isA<FalhaDeConsentimentoAusente>()),
        );
      },
    );

    test(
      'sincronização falha de forma segura se permissão de sistema for negada',
      () async {
        await repoConsentimento.salvarConsentimento(
          ConsentimentoDeDados.criarNovo(
            id: 'c-1',
            atletaId: atletaId,
            organizacaoId: orgId,
            categoria: CategoriaDeDadoSensivel.dadosEsportivosBasicos,
            finalidade: 'Passos',
            versaoDoTermo: '1.0',
            dataDeConcessao: DateTime.utc(2026, 8, 27),
          ),
        );

        fakeFonte.disponibilidade = DisponibilidadeDeRecurso.permissaoNegada;

        expect(
          () => sincronizador.sincronizarTreino(
            atletaId: atletaId,
            organizacaoId: orgId,
            sincronizacaoId: 'sync-03',
            inicio: DateTime.utc(2026, 8, 27, 8),
            fim: DateTime.utc(2026, 8, 27, 9),
          ),
          throwsA(isA<FalhaDePermissaoDeSistema>()),
        );
      },
    );

    test('sincronização conclui com sucesso respeitando o filtro de categorias autorizadas', () async {
      // Concede passos mas NÃO concede frequência cardíaca
      await repoConsentimento.salvarConsentimento(
        ConsentimentoDeDados.criarNovo(
          id: 'c-1',
          atletaId: atletaId,
          organizacaoId: orgId,
          categoria: CategoriaDeDadoSensivel.dadosEsportivosBasicos,
          finalidade: 'Passos',
          versaoDoTermo: '1.0',
          dataDeConcessao: DateTime.utc(2026, 8, 27),
        ),
      );

      final inicio = DateTime.utc(2026, 8, 27, 8);
      final fim = DateTime.utc(2026, 8, 27, 9);

      fakeFonte.proximoResumo = ResumoDeAtividade(
        inicio: inicio,
        fim: fim,
        duracao: const Duration(minutes: 60),
        passos: 6200,
        distanciaMetros: 5100.0,
        frequenciaCardiacaMediaBpm: 152, // Não consentido pelo atleta
        frequenciaCardiacaMaximaBpm: 180,
        origem: 'fake_health',
      );

      final resultado = await sincronizador.sincronizarTreino(
        atletaId: atletaId,
        organizacaoId: orgId,
        sincronizacaoId: 'sync-04',
        inicio: inicio,
        fim: fim,
      );

      expect(resultado, isNotNull);
      expect(resultado!.passos, equals(6200));
      expect(resultado.distanciaMetros, equals(5100.0));
      expect(
        resultado.frequenciaCardiacaMediaBpm,
        isNull,
      ); // Filtrado com sucesso
      expect(resultado.frequenciaCardiacaMaximaBpm, isNull);

      expect(
        repoSincronizacao.status['sync-04'],
        equals(EstadoDeSincronizacao.concluida),
      );
      expect(repoDispositivos.sincronizacoes['fake_health'], isNotNull);
    });
  });
}
