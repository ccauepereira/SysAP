import 'package:flutter_test/flutter_test.dart';
import 'package:sysap/dominio/entidades/papel_do_usuario.dart';
import 'package:sysap/dominio/entidades/sessao_autenticada.dart';
import 'package:sysap/infraestrutura/armazenamento_seguro/armazenamento_seguro_de_sessao.dart';

void main() {
  group('Infraestrutura: ArmazenamentoSeguroDeSessao', () {
    test('grava, recupera e limpa a sessão corretamente', () async {
      final armazenamento = ArmazenamentoSeguroDeSessao();

      expect(await armazenamento.lerSessao(), isNull);

      final sessaoOriginal = SessaoAutenticada(
        tokenDeAcesso: 'jwt-access-token',
        tokenDeAtualizacao: 'jwt-refresh-token',
        idDaSessao: 'sess-10',
        idDoPerfil: 'prof-10',
        idDaOrganizacao: 'org-10',
        papel: PapelDoUsuario.athlete,
        expiraEmSegundos: 3600,
        criadoEm: DateTime.now().toUtc(),
      );

      await armazenamento.salvarSessao(sessaoOriginal);

      final sessaoLida = await armazenamento.lerSessao();
      expect(sessaoLida, isNotNull);
      expect(sessaoLida!.tokenDeAcesso, equals('jwt-access-token'));
      expect(sessaoLida.tokenDeAtualizacao, equals('jwt-refresh-token'));
      expect(sessaoLida.idDaSessao, equals('sess-10'));
      expect(sessaoLida.papel, equals(PapelDoUsuario.athlete));

      await armazenamento.limparSessao();
      expect(await armazenamento.lerSessao(), isNull);
    });

    test(
      'limpa e retorna nulo se os dados no cofre estiverem corrompidos',
      () async {
        final cofreCorrompido = {
          'sysap_secure_session_v1': '{corrompido: true, sem_fechamento',
        };
        final armazenamento = ArmazenamentoSeguroDeSessao(
          cofreInicial: cofreCorrompido,
        );

        final sessao = await armazenamento.lerSessao();
        expect(sessao, isNull);
        expect(cofreCorrompido.containsKey('sysap_secure_session_v1'), isFalse);
      },
    );
  });
}
