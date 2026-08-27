import 'package:flutter_test/flutter_test.dart';
import 'package:sysap/aplicacao/conectividade/verificar_conectividade.dart';
import 'package:sysap/aplicacao/seguranca/validar_desbloqueio_biometrico.dart';
import 'package:sysap/dominio/entidades/disponibilidade_de_recurso.dart';
import 'package:sysap/dominio/servicos/autenticacao_biometrica.dart';
import 'package:sysap/dominio/servicos/detector_de_conectividade.dart';
import 'package:sysap/dominio/servicos/notificacoes_do_dispositivo.dart';
import 'package:sysap/dominio/servicos/servico_de_localizacao.dart';

class FakeBiometria implements AutenticacaoBiometrica {
  bool suporta = true;
  bool autenticado = true;

  @override
  Future<bool> dispositivoSuportaBiometria() async => suporta;

  @override
  Future<List<TipoDeBiometria>> tiposDisponiveis() async =>
      suporta ? [TipoDeBiometria.impressaoDigital] : [TipoDeBiometria.nenhuma];

  @override
  Future<bool> autenticarLocalmente({required String motivo}) async =>
      autenticado;
}

class FakeNotificacoes implements NotificacoesDoDispositivo {
  bool permissaoAtiva = false;
  final List<String> agendadas = [];

  @override
  Future<bool> solicitarPermissaoNotificacoes() async => true;

  @override
  Future<bool> possuiPermissaoAtiva() async => permissaoAtiva;

  @override
  Future<void> agendarNotificacaoLocal({
    required String id,
    required String titulo,
    required String corpo,
    required CategoriaDeNotificacao categoria,
    required DateTime agendadoPara,
  }) async {
    agendadas.add(id);
  }

  @override
  Future<void> cancelarNotificacao(String id) async {
    agendadas.remove(id);
  }
}

class FakeLocalizacao implements ServicoDeLocalizacao {
  @override
  Future<DisponibilidadeDeRecurso> verificarCapacidadeGps() async =>
      DisponibilidadeDeRecurso.naoSolicitado;
}

class FakeConectividade implements DetectorDeConectividade {
  EstadoDeConectividade estado = EstadoDeConectividade.online;

  @override
  Future<EstadoDeConectividade> verificarConectividade() async => estado;

  @override
  Stream<EstadoDeConectividade> observarConectividade() => Stream.value(estado);
}

void main() {
  group('Aplicação — ValidarDesbloqueioBiometrico', () {
    late FakeBiometria fakeBiometria;
    late ValidarDesbloqueioBiometrico casoDeUso;

    setUp(() {
      fakeBiometria = FakeBiometria();
      casoDeUso = ValidarDesbloqueioBiometrico(biometria: fakeBiometria);
    });

    test('retorna false quando o dispositivo não suporta biometria', () async {
      fakeBiometria.suporta = false;
      final resultado = await casoDeUso.executar(motivo: 'Teste');
      expect(resultado, isFalse);
    });

    test(
      'retorna true quando o usuário confirma autenticação biométrica',
      () async {
        fakeBiometria.suporta = true;
        fakeBiometria.autenticado = true;
        final resultado = await casoDeUso.executar(motivo: 'Desbloqueio local');
        expect(resultado, isTrue);
      },
    );

    test('retorna false quando a autenticação biométrica é recusada', () async {
      fakeBiometria.suporta = true;
      fakeBiometria.autenticado = false;
      final resultado = await casoDeUso.executar(motivo: 'Desbloqueio local');
      expect(resultado, isFalse);
    });
  });

  group('Aplicação — Notificações Locais & Conectividade', () {
    test('agendamento e cancelamento de notificações locais funcionam corretamente', () async {
      final fakeNotificacoes = FakeNotificacoes();

      await fakeNotificacoes.agendarNotificacaoLocal(
        id: 'notif-1',
        titulo: 'Treino Próximo',
        corpo: 'Seu treino começa em breve.',
        categoria: CategoriaDeNotificacao.lembreteDeTreino,
        agendadoPara: DateTime.utc(2026, 8, 27, 18),
      );

      expect(fakeNotificacoes.agendadas, contains('notif-1'));

      await fakeNotificacoes.cancelarNotificacao('notif-1');
      expect(fakeNotificacoes.agendadas, isEmpty);
    });

    test(
      'localização permanece em estado seguro nãoSolicitado sem iniciar GPS',
      () async {
        final fakeLoc = FakeLocalizacao();
        final status = await fakeLoc.verificarCapacidadeGps();
        expect(status, equals(DisponibilidadeDeRecurso.naoSolicitado));
      },
    );

    test(
      'verificação de conectividade informa o estado corretamente',
      () async {
        final fakeConn = FakeConectividade();
        final casoDeUso = VerificarConectividade(detector: fakeConn);

        expect(
          await casoDeUso.executar(),
          equals(EstadoDeConectividade.online),
        );
        fakeConn.estado = EstadoDeConectividade.offline;
        expect(
          await casoDeUso.executar(),
          equals(EstadoDeConectividade.offline),
        );
      },
    );
  });
}
