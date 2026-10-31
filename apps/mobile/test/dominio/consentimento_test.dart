import 'package:flutter_test/flutter_test.dart';
import 'package:sysap/aplicacao/consentimento/gerenciador_de_consentimento.dart';
import 'package:sysap/dominio/entidades/categoria_de_dado_sensivel.dart';
import 'package:sysap/dominio/entidades/consentimento_de_dados.dart';
import 'package:sysap/dominio/entidades/falha_de_integracao.dart';
import 'package:sysap/dominio/entidades/papel_do_usuario.dart';
import 'package:sysap/infraestrutura/consentimento/repositorio_de_consentimentos_local.dart';

void main() {
  group('Domínio — ConsentimentoDeDados', () {
    test('estaVigente é verdadeiro apenas para concedido e sem revogação', () {
      final consentimento = ConsentimentoDeDados.criarNovo(
        id: 'cons-1',
        atletaId: 'atleta-100',
        organizacaoId: 'org-1',
        categoria: CategoriaDeDadoSensivel.dadosEsportivosBasicos,
        finalidade: 'Registro de passos e duração',
        versaoDoTermo: '1.0',
        dataDeConcessao: DateTime.utc(2026, 8, 27),
      );

      expect(consentimento.estaVigente, isTrue);
      expect(consentimento.status, equals(StatusDoConsentimento.concedido));
      expect(consentimento.revogadoEm, isNull);

      final revogado = consentimento.revogar(
        dataDaRevogacao: DateTime.utc(2026, 8, 28),
      );

      expect(revogado.estaVigente, isFalse);
      expect(revogado.status, equals(StatusDoConsentimento.revogado));
      expect(revogado.revogadoEm, equals(DateTime.utc(2026, 8, 28)));
    });
  });

  group('Aplicação — GerenciadorDeConsentimento & Autorização de Acesso', () {
    late RepositorioDeConsentimentosLocal repositorio;
    late GerenciadorDeConsentimento gerenciador;

    const atletaId = 'atleta-001';
    const organizacaoId = 'org-sysap-01';

    setUp(() {
      repositorio = RepositorioDeConsentimentosLocal();
      gerenciador = GerenciadorDeConsentimento(repositorio: repositorio);
    });

    test(
      'consentimento ausente impede leitura com FalhaDeConsentimentoAusente',
      () async {
        expect(
          () => gerenciador.exigirConsentimentoVigente(
            atletaId: atletaId,
            organizacaoId: organizacaoId,
            categoria: CategoriaDeDadoSensivel.dadosEsportivosBasicos,
          ),
          throwsA(isA<FalhaDeConsentimentoAusente>()),
        );
      },
    );

    test(
      'consentimento revogado bloqueia nova leitura imediatamente',
      () async {
        await repositorio.salvarConsentimento(
          ConsentimentoDeDados.criarNovo(
            id: 'cons-1',
            atletaId: atletaId,
            organizacaoId: organizacaoId,
            categoria: CategoriaDeDadoSensivel.dadosEsportivosBasicos,
            finalidade: 'Passos e treino',
            versaoDoTermo: '1.0',
            dataDeConcessao: DateTime.utc(2026, 8, 27),
          ),
        );

        expect(
          await gerenciador.estaConsentido(
            atletaId: atletaId,
            organizacaoId: organizacaoId,
            categoria: CategoriaDeDadoSensivel.dadosEsportivosBasicos,
          ),
          isTrue,
        );

        await repositorio.revogarConsentimento(
          atletaId: atletaId,
          organizacaoId: organizacaoId,
          categoria: CategoriaDeDadoSensivel.dadosEsportivosBasicos,
          dataDaRevogacao: DateTime.utc(2026, 8, 27, 12),
        );

        expect(
          await gerenciador.estaConsentido(
            atletaId: atletaId,
            organizacaoId: organizacaoId,
            categoria: CategoriaDeDadoSensivel.dadosEsportivosBasicos,
          ),
          isFalse,
        );

        expect(
          () => gerenciador.exigirConsentimentoVigente(
            atletaId: atletaId,
            organizacaoId: organizacaoId,
            categoria: CategoriaDeDadoSensivel.dadosEsportivosBasicos,
          ),
          throwsA(isA<FalhaDeConsentimentoAusente>()),
        );
      },
    );

    test('consentimento de saúde não concede GPS e consentimento de GPS não concede frequência cardíaca', () async {
      await repositorio.salvarConsentimento(
        ConsentimentoDeDados.criarNovo(
          id: 'cons-saude',
          atletaId: atletaId,
          organizacaoId: organizacaoId,
          categoria: CategoriaDeDadoSensivel.dadosEsportivosBasicos,
          finalidade: 'Passos',
          versaoDoTermo: '1.0',
          dataDeConcessao: DateTime.utc(2026, 8, 27),
        ),
      );

      expect(
        await gerenciador.estaConsentido(
          atletaId: atletaId,
          organizacaoId: organizacaoId,
          categoria: CategoriaDeDadoSensivel.dadosEsportivosBasicos,
        ),
        isTrue,
      );
      expect(
        await gerenciador.estaConsentido(
          atletaId: atletaId,
          organizacaoId: organizacaoId,
          categoria: CategoriaDeDadoSensivel.localizacaoGpsPosTreino,
        ),
        isFalse,
      );
      expect(
        await gerenciador.estaConsentido(
          atletaId: atletaId,
          organizacaoId: organizacaoId,
          categoria: CategoriaDeDadoSensivel.frequenciaCardiaca,
        ),
        isFalse,
      );
    });

    test('Owner sem consentimento de compartilhamento do atleta é bloqueado com FalhaDeAutorizacaoOwner', () async {
      // Atleta concedeu passos para si, mas NÃO concedeu compartilhamento com a equipe
      await repositorio.salvarConsentimento(
        ConsentimentoDeDados.criarNovo(
          id: 'cons-passos',
          atletaId: atletaId,
          organizacaoId: organizacaoId,
          categoria: CategoriaDeDadoSensivel.dadosEsportivosBasicos,
          finalidade: 'Passos pessoais',
          versaoDoTermo: '1.0',
          dataDeConcessao: DateTime.utc(2026, 8, 27),
        ),
      );

      expect(
        () => gerenciador.validarAutorizacaoDeAcesso(
          solicitanteId: 'owner-artur',
          solicitantePapel: PapelDoUsuario.owner,
          solicitanteOrganizacaoId: organizacaoId,
          solicitanteVinculoAtivo: true,
          alvoAtletaId: atletaId,
          alvoOrganizacaoId: organizacaoId,
          categoria: CategoriaDeDadoSensivel.dadosEsportivosBasicos,
        ),
        throwsA(isA<FalhaDeAutorizacaoOwner>()),
      );
    });

    test('Owner recebe acesso se houver mesma organização, vínculo ativo e consentimentos de compartilhamento e categoria', () async {
      await repositorio.salvarConsentimento(
        ConsentimentoDeDados.criarNovo(
          id: 'cons-compartilhamento',
          atletaId: atletaId,
          organizacaoId: organizacaoId,
          categoria: CategoriaDeDadoSensivel.compartilhamentoEquipeTecnica,
          finalidade: 'Compartilhamento com comissão',
          versaoDoTermo: '1.0',
          dataDeConcessao: DateTime.utc(2026, 8, 27),
        ),
      );

      await repositorio.salvarConsentimento(
        ConsentimentoDeDados.criarNovo(
          id: 'cons-passos',
          atletaId: atletaId,
          organizacaoId: organizacaoId,
          categoria: CategoriaDeDadoSensivel.dadosEsportivosBasicos,
          finalidade: 'Passos',
          versaoDoTermo: '1.0',
          dataDeConcessao: DateTime.utc(2026, 8, 27),
        ),
      );

      // Deve passar sem lançar exceção
      await gerenciador.validarAutorizacaoDeAcesso(
        solicitanteId: 'owner-artur',
        solicitantePapel: PapelDoUsuario.owner,
        solicitanteOrganizacaoId: organizacaoId,
        solicitanteVinculoAtivo: true,
        alvoAtletaId: atletaId,
        alvoOrganizacaoId: organizacaoId,
        categoria: CategoriaDeDadoSensivel.dadosEsportivosBasicos,
      );
    });

    test(
      'Owner de outra organização é bloqueado com FalhaDeAutorizacaoOwner',
      () async {
        await repositorio.salvarConsentimento(
          ConsentimentoDeDados.criarNovo(
            id: 'cons-comp',
            atletaId: atletaId,
            organizacaoId: organizacaoId,
            categoria: CategoriaDeDadoSensivel.compartilhamentoEquipeTecnica,
            finalidade: 'Compartilhamento',
            versaoDoTermo: '1.0',
            dataDeConcessao: DateTime.utc(2026, 8, 27),
          ),
        );

        expect(
          () => gerenciador.validarAutorizacaoDeAcesso(
            solicitanteId: 'owner-invasor',
            solicitantePapel: PapelDoUsuario.owner,
            solicitanteOrganizacaoId: 'outra-organizacao-99',
            solicitanteVinculoAtivo: true,
            alvoAtletaId: atletaId,
            alvoOrganizacaoId: organizacaoId,
            categoria: CategoriaDeDadoSensivel.dadosEsportivosBasicos,
          ),
          throwsA(isA<FalhaDeAutorizacaoOwner>()),
        );
      },
    );

    test('solicitante com vínculo inativo é bloqueado', () async {
      expect(
        () => gerenciador.validarAutorizacaoDeAcesso(
          solicitanteId: 'owner-artur',
          solicitantePapel: PapelDoUsuario.owner,
          solicitanteOrganizacaoId: organizacaoId,
          solicitanteVinculoAtivo: false,
          alvoAtletaId: atletaId,
          alvoOrganizacaoId: organizacaoId,
          categoria: CategoriaDeDadoSensivel.dadosEsportivosBasicos,
        ),
        throwsA(isA<FalhaDeAutorizacaoOwner>()),
      );
    });
  });
}
