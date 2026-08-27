import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:sysap/aplicacao/consentimento/gerenciador_de_consentimento.dart';
import 'package:sysap/apresentacao/consentimento/tela_de_gerenciamento_de_consentimento.dart';
import 'package:sysap/apresentacao/tema/tema_sysap.dart';
import 'package:sysap/dominio/entidades/categoria_de_dado_sensivel.dart';
import 'package:sysap/dominio/entidades/disponibilidade_de_recurso.dart';
import 'package:sysap/dominio/entidades/resumo_de_atividade.dart';
import 'package:sysap/dominio/servicos/fonte_de_dados_de_saude.dart';
import 'package:sysap/infraestrutura/consentimento/repositorio_de_consentimentos_local.dart';

class FakeFonteWidgetTest implements FonteDeDadosDeSaude {
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
  }) async => null;
}

void main() {
  testWidgets(
    'TelaDeGerenciamentoDeConsentimento inicializa categorias sem pré-seleção e com acessibilidade',
    (tester) async {
      final repo = RepositorioDeConsentimentosLocal();
      final gerenciador = GerenciadorDeConsentimento(repositorio: repo);
      final fakeFonte = FakeFonteWidgetTest();

      await tester.pumpWidget(
        MaterialApp(
          theme: TemaSysAP.temaEscuro,
          home: TelaDeGerenciamentoDeConsentimento(
            atletaId: 'atleta-test',
            organizacaoId: 'org-test',
            repositorioConsentimentos: repo,
            gerenciadorConsentimento: gerenciador,
            fonteSaude: fakeFonte,
          ),
        ),
      );

      await tester.pumpAndSettle();

      // Verifica presença de cabeçalhos e textos informativos
      expect(find.text('Privacidade e Sensores'), findsOneWidget);
      expect(find.text('Transparência e Controle'), findsOneWidget);
      expect(find.text('Fonte: Health Connect (Android)'), findsOneWidget);
      expect(find.text('Disponível'), findsOneWidget);

      // Verifica que todas as 4 categorias aparecem na tela
      expect(find.text('Dados Esportivos Básicos'), findsOneWidget);
      expect(find.text('Frequência Cardíaca em Treino'), findsOneWidget);
      expect(find.text('Localização e GPS Pós-Treino'), findsOneWidget);
      expect(find.text('Compartilhamento com Equipe Técnica'), findsOneWidget);

      // Verifica que nenhum Switch está pré-selecionado como ativo
      final switches = tester.widgetList<Switch>(find.byType(Switch));
      expect(switches.length, equals(4));
      for (final s in switches) {
        expect(s.value, isFalse);
      }

      // Ativa o consentimento de Dados Esportivos Básicos
      await tester.tap(find.byType(Switch).first);
      await tester.pumpAndSettle();

      // Confirma que agora o primeiro switch está ativo
      final switchesAposAtivar = tester.widgetList<Switch>(find.byType(Switch));
      expect(switchesAposAtivar.first.value, isTrue);

      // Tenta desativar: deve abrir diálogo de confirmação de revogação
      await tester.tap(find.byType(Switch).first);
      await tester.pumpAndSettle();

      expect(find.text('Revogar Consentimento'), findsOneWidget);
      expect(find.text('Revogar'), findsOneWidget);
      expect(find.text('Cancelar'), findsOneWidget);

      // Cancela a revogação
      await tester.tap(find.text('Cancelar'));
      await tester.pumpAndSettle();

      // O switch continua ativo
      final switchesAposCancelar = tester.widgetList<Switch>(
        find.byType(Switch),
      );
      expect(switchesAposCancelar.first.value, isTrue);
    },
  );
}
