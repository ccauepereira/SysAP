import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:sysap/aplicacao/configuracao/ambiente.dart';
import 'package:sysap/aplicacao/configuracao/configuracao_publica.dart';
import 'package:sysap/aplicacao/sessao/gerenciador_de_sessao.dart';
import 'package:sysap/apresentacao/navegacao/navegacao_por_perfil.dart';
import 'package:sysap/apresentacao/tema/tema_sysap.dart';
import 'package:sysap/dominio/entidades/identidade_do_usuario.dart';
import 'package:sysap/dominio/entidades/papel_do_usuario.dart';
import 'package:sysap/dominio/entidades/sessao_autenticada.dart';
import 'package:sysap/dominio/entidades/status_do_vinculo.dart';
import 'package:sysap/dominio/entidades/vinculo_organizacional.dart';
import 'package:sysap/infraestrutura/api/cliente_da_api.dart';
import 'package:sysap/infraestrutura/armazenamento_seguro/armazenamento_seguro_de_sessao.dart';

void main() {
  group('Apresentação: NavegacaoPorPerfil', () {
    late GerenciadorDeSessao gerenciador;

    setUp(() {
      gerenciador = GerenciadorDeSessao(
        clienteApi: ClienteDaApi(
          configuracao: ConfiguracaoPublica(
            ambiente: AmbienteDeExecucao.desenvolvimento,
            urlDaApi: Uri.parse('http://127.0.0.1:8080'),
          ),
        ),
        repositorioSessao: ArmazenamentoSeguroDeSessao(),
      );
    });

    final sessaoPadrao = SessaoAutenticada(
      tokenDeAcesso: 'acc-1',
      tokenDeAtualizacao: 'ref-1',
      idDaSessao: 's-1',
      idDoPerfil: 'p-1',
      idDaOrganizacao: 'o-1',
      papel: PapelDoUsuario.owner,
      expiraEmSegundos: 3600,
      criadoEm: DateTime.now().toUtc(),
    );

    testWidgets('Owner autenticado renderiza casca da área owner', (
      tester,
    ) async {
      const identidadeOwner = IdentidadeDoUsuario(
        idDoPerfil: 'p-1',
        nomeDeExibicao: 'Artur Performance',
        vinculos: [
          VinculoOrganizacional(
            idDaOrganizacao: 'o-1',
            papel: PapelDoUsuario.owner,
            status: StatusDoVinculo.ativo,
          ),
        ],
      );

      await tester.pumpWidget(
        MaterialApp(
          theme: TemaSysAP.temaEscuro,
          home: NavegacaoPorPerfil(
            gerenciadorDeSessao: gerenciador,
            sessao: sessaoPadrao,
            identidade: identidadeOwner,
          ),
        ),
      );

      expect(find.byType(CascaDaAreaOwner), findsOneWidget);
      expect(find.text('Área de Gestão — Artur Performance'), findsOneWidget);
      expect(find.text('Artur Performance'), findsOneWidget);
      expect(find.text('Owner / Administrador'), findsOneWidget);
    });

    testWidgets('Athlete autenticado renderiza casca da área athlete', (
      tester,
    ) async {
      const identidadeAthlete = IdentidadeDoUsuario(
        idDoPerfil: 'p-2',
        nomeDeExibicao: 'Lucas Silva',
        vinculos: [
          VinculoOrganizacional(
            idDaOrganizacao: 'o-1',
            papel: PapelDoUsuario.athlete,
            status: StatusDoVinculo.ativo,
          ),
        ],
      );

      await tester.pumpWidget(
        MaterialApp(
          theme: TemaSysAP.temaEscuro,
          home: NavegacaoPorPerfil(
            gerenciadorDeSessao: gerenciador,
            sessao: sessaoPadrao,
            identidade: identidadeAthlete,
          ),
        ),
      );

      expect(find.byType(CascaDaAreaAthlete), findsOneWidget);
      expect(find.text('Área do Atleta — SysAP'), findsOneWidget);
      expect(find.text('Lucas Silva'), findsOneWidget);
      expect(find.text('Atleta'), findsOneWidget);
    });

    testWidgets('Usuário com vínculo inativo renderiza TelaDeAcessoNegado', (
      tester,
    ) async {
      const identidadeInativa = IdentidadeDoUsuario(
        idDoPerfil: 'p-3',
        nomeDeExibicao: 'Usuário Suspenso',
        vinculos: [
          VinculoOrganizacional(
            idDaOrganizacao: 'o-1',
            papel: PapelDoUsuario.athlete,
            status: StatusDoVinculo.suspenso,
          ),
        ],
      );

      await tester.pumpWidget(
        MaterialApp(
          theme: TemaSysAP.temaEscuro,
          home: NavegacaoPorPerfil(
            gerenciadorDeSessao: gerenciador,
            sessao: sessaoPadrao,
            identidade: identidadeInativa,
          ),
        ),
      );

      expect(find.byType(TelaDeAcessoNegado), findsOneWidget);
      expect(find.text('Acesso Não Autorizado'), findsOneWidget);
    });
  });
}
