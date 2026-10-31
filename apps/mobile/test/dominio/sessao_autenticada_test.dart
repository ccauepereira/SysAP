import 'package:flutter_test/flutter_test.dart';
import 'package:sysap/dominio/entidades/identidade_do_usuario.dart';
import 'package:sysap/dominio/entidades/papel_do_usuario.dart';
import 'package:sysap/dominio/entidades/sessao_autenticada.dart';
import 'package:sysap/dominio/entidades/status_do_vinculo.dart';

void main() {
  group('Domínio: SessaoAutenticada e Identidade', () {
    test('SessaoAutenticada calcula expiração e renovação corretamente', () {
      final sessaoAtiva = SessaoAutenticada(
        tokenDeAcesso: 'access-123',
        tokenDeAtualizacao: 'refresh-123',
        idDaSessao: 'session-123',
        idDoPerfil: 'profile-123',
        idDaOrganizacao: 'org-123',
        papel: PapelDoUsuario.owner,
        expiraEmSegundos: 3600,
        criadoEm: DateTime.now().toUtc(),
      );

      expect(sessaoAtiva.expirou, isFalse);
      expect(sessaoAtiva.precisaRenovar, isFalse);
      expect(sessaoAtiva.papel, equals(PapelDoUsuario.owner));
      expect(sessaoAtiva.papel.ehOwner, isTrue);

      // Garante que a representação textual nunca vaza tokens
      expect(sessaoAtiva.toString(), isNot(contains('access-123')));
      expect(sessaoAtiva.toString(), isNot(contains('refresh-123')));
      expect(sessaoAtiva.toString(), contains('owner'));
    });

    test('SessaoAutenticada identifica sessão expirada', () {
      final sessaoExpirada = SessaoAutenticada(
        tokenDeAcesso: 'access-old',
        tokenDeAtualizacao: 'refresh-old',
        idDaSessao: 'session-old',
        idDoPerfil: 'profile-old',
        idDaOrganizacao: 'org-old',
        papel: PapelDoUsuario.athlete,
        expiraEmSegundos: 10,
        criadoEm: DateTime.now().toUtc().subtract(const Duration(minutes: 5)),
      );

      expect(sessaoExpirada.expirou, isTrue);
      expect(sessaoExpirada.precisaRenovar, isTrue);
      expect(sessaoExpirada.papel.ehAthlete, isTrue);
    });

    test('SessaoAutenticada rotaciona tokens com segurança', () {
      final sessaoOriginal = SessaoAutenticada(
        tokenDeAcesso: 'token-antigo',
        tokenDeAtualizacao: 'refresh-antigo',
        idDaSessao: 'session-1',
        idDoPerfil: 'profile-1',
        idDaOrganizacao: 'org-1',
        papel: PapelDoUsuario.trainer,
        expiraEmSegundos: 3600,
        criadoEm: DateTime.now().toUtc(),
      );

      final sessaoAtualizada = sessaoOriginal.comNovosTokens(
        novoTokenDeAcesso: 'token-novo',
        novoTokenDeAtualizacao: 'refresh-novo',
        novoExpiraEmSegundos: 7200,
      );

      expect(sessaoAtualizada.tokenDeAcesso, equals('token-novo'));
      expect(sessaoAtualizada.tokenDeAtualizacao, equals('refresh-novo'));
      expect(sessaoAtualizada.expiraEmSegundos, equals(7200));
      expect(sessaoAtualizada.idDaSessao, equals('session-1'));
    });

    test('IdentidadeDoUsuario valida vinculos ativos e papel principal', () {
      final json = {
        'profile': {
          'id': '40000000-0000-4000-8000-000000000001',
          'display_name': 'Artur Silva',
        },
        'memberships': [
          {
            'organization_id': '30000000-0000-4000-8000-000000000001',
            'role': 'owner',
            'status': 'active',
          },
        ],
      };

      final identidade = IdentidadeDoUsuario.doJson(json);
      expect(
        identidade.idDoPerfil,
        equals('40000000-0000-4000-8000-000000000001'),
      );
      expect(identidade.nomeDeExibicao, equals('Artur Silva'));
      expect(identidade.possuiAcessoAtivo, isTrue);
      expect(identidade.papelPrincipal, equals(PapelDoUsuario.owner));
      expect(identidade.papelPrincipal.ehOwner, isTrue);
    });

    test(
      'IdentidadeDoUsuario rejeita vinculo suspenso ou papel desconhecido',
      () {
        final jsonSuspenso = {
          'profile': {'id': 'id-1', 'display_name': 'Atleta Suspenso'},
          'memberships': [
            {
              'organization_id': 'org-1',
              'role': 'athlete',
              'status': 'suspended',
            },
          ],
        };

        final identidadeSuspensa = IdentidadeDoUsuario.doJson(jsonSuspenso);
        expect(identidadeSuspensa.possuiAcessoAtivo, isFalse);
        expect(
          identidadeSuspensa.vinculoPrincipal?.status,
          equals(StatusDoVinculo.suspenso),
        );
      },
    );
  });
}
