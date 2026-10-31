import 'package:flutter/material.dart';

import '../../aplicacao/consentimento/gerenciador_de_consentimento.dart';
import '../../dominio/entidades/categoria_de_dado_sensivel.dart';
import '../../dominio/entidades/consentimento_de_dados.dart';
import '../../dominio/entidades/disponibilidade_de_recurso.dart';
import '../../dominio/repositorios/repositorio_de_consentimentos.dart';
import '../../dominio/servicos/fonte_de_dados_de_saude.dart';
import '../tema/tema_sysap.dart';

/// Tela de gerenciamento de consentimentos e permissões de sensores esportivos.
class TelaDeGerenciamentoDeConsentimento extends StatefulWidget {
  final String atletaId;
  final String organizacaoId;
  final RepositorioDeConsentimentos repositorioConsentimentos;
  final GerenciadorDeConsentimento gerenciadorConsentimento;
  final FonteDeDadosDeSaude fonteSaude;

  const TelaDeGerenciamentoDeConsentimento({
    super.key,
    required this.atletaId,
    required this.organizacaoId,
    required this.repositorioConsentimentos,
    required this.gerenciadorConsentimento,
    required this.fonteSaude,
  });

  @override
  State<TelaDeGerenciamentoDeConsentimento> createState() =>
      _TelaDeGerenciamentoDeConsentimentoState();
}

class _TelaDeGerenciamentoDeConsentimentoState
    extends State<TelaDeGerenciamentoDeConsentimento> {
  bool _carregando = true;
  DisponibilidadeDeRecurso _disponibilidadeFonte =
      DisponibilidadeDeRecurso.indisponivel;
  final Map<CategoriaDeDadoSensivel, bool> _estadosConsentimento = {};

  @override
  void initState() {
    super.initState();
    _carregarDados();
  }

  Future<void> _carregarDados() async {
    setState(() => _carregando = true);
    final disponibilidade = await widget.fonteSaude.verificarDisponibilidade();
    final consentimentos = await widget.repositorioConsentimentos
        .consultarTodos(
          atletaId: widget.atletaId,
          organizacaoId: widget.organizacaoId,
        );

    // Inicialização segura: nenhuma categoria é pré-selecionada por padrão
    for (final cat in CategoriaDeDadoSensivel.values) {
      _estadosConsentimento[cat] = false;
    }

    for (final c in consentimentos) {
      if (c.estaVigente) {
        _estadosConsentimento[c.categoria] = true;
      }
    }

    if (mounted) {
      setState(() {
        _disponibilidadeFonte = disponibilidade;
        _carregando = false;
      });
    }
  }

  Future<void> _alternarConsentimento(
    CategoriaDeDadoSensivel categoria,
    bool novoValor,
  ) async {
    if (!novoValor) {
      // Confirmação explícita para revogação
      final confirmar = await showDialog<bool>(
        context: context,
        builder: (ctx) => AlertDialog(
          backgroundColor: TemaSysAP.corSuperficieEscura,
          title: const Text(
            'Revogar Consentimento',
            style: TextStyle(color: Colors.white, fontWeight: FontWeight.bold),
          ),
          content: Text(
            'Tem certeza que deseja revogar o consentimento para ${categoria.tituloEmPortugues}? Novas sincronizações e compartilhamentos desta categoria serão interrompidos imediatamente.',
            style: const TextStyle(color: TemaSysAP.corTextoSecundario),
          ),
          actions: [
            TextButton(
              onPressed: () => Navigator.of(ctx).pop(false),
              child: const Text(
                'Cancelar',
                style: TextStyle(color: Colors.white70),
              ),
            ),
            ElevatedButton(
              onPressed: () => Navigator.of(ctx).pop(true),
              style: ElevatedButton.styleFrom(
                backgroundColor: TemaSysAP.corAlerta,
                foregroundColor: Colors.white,
              ),
              child: const Text('Revogar'),
            ),
          ],
        ),
      );

      if (confirmar != true) return;

      await widget.repositorioConsentimentos.revogarConsentimento(
        atletaId: widget.atletaId,
        organizacaoId: widget.organizacaoId,
        categoria: categoria,
        dataDaRevogacao: DateTime.now().toUtc(),
      );
    } else {
      await widget.repositorioConsentimentos.salvarConsentimento(
        ConsentimentoDeDados.criarNovo(
          id: 'cons-${categoria.codigo}-${DateTime.now().millisecondsSinceEpoch}',
          atletaId: widget.atletaId,
          organizacaoId: widget.organizacaoId,
          categoria: categoria,
          finalidade: categoria.descricaoFinalidade,
          versaoDoTermo: '1.0',
          dataDeConcessao: DateTime.now().toUtc(),
        ),
      );
    }

    await _carregarDados();
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      backgroundColor: TemaSysAP.corFundoObsidiana,
      appBar: AppBar(
        title: const Text(
          'Privacidade e Sensores',
          style: TextStyle(fontSize: 18, fontWeight: FontWeight.bold),
        ),
      ),
      body: _carregando
          ? const Center(
              child: CircularProgressIndicator(
                color: TemaSysAP.corDouradoPrincipal,
              ),
            )
          : SingleChildScrollView(
              padding: const EdgeInsets.all(20),
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.stretch,
                children: [
                  _construirCardExplicativo(),
                  const SizedBox(height: 20),
                  _construirCardStatusFonte(),
                  const SizedBox(height: 24),
                  const Text(
                    'Categorias e Consentimentos Granulares',
                    style: TextStyle(
                      color: Colors.white,
                      fontSize: 16,
                      fontWeight: FontWeight.bold,
                    ),
                  ),
                  const SizedBox(height: 12),
                  ...CategoriaDeDadoSensivel.values.map(
                    (cat) => _construirItemCategoria(cat),
                  ),
                ],
              ),
            ),
    );
  }

  Widget _construirCardExplicativo() {
    return Card(
      color: TemaSysAP.corSuperficieEscura,
      child: Padding(
        padding: const EdgeInsets.all(16),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Row(
              children: const [
                Icon(
                  Icons.shield_outlined,
                  color: TemaSysAP.corDouradoPrincipal,
                  size: 24,
                ),
                SizedBox(width: 10),
                Text(
                  'Transparência e Controle',
                  style: TextStyle(
                    color: Colors.white,
                    fontWeight: FontWeight.bold,
                    fontSize: 15,
                  ),
                ),
              ],
            ),
            const SizedBox(height: 8),
            const Text(
              'No SysAP, seus dados esportivos e de sensores pertencem a você. O consentimento é separado por categoria, totalmente revogável a qualquer momento e não impede seu login ou uso básico do aplicativo.',
              style: TextStyle(
                color: TemaSysAP.corTextoSecundario,
                fontSize: 13,
                height: 1.4,
              ),
            ),
          ],
        ),
      ),
    );
  }

  Widget _construirCardStatusFonte() {
    final statusColor = switch (_disponibilidadeFonte) {
      DisponibilidadeDeRecurso.disponivel => TemaSysAP.corSucesso,
      DisponibilidadeDeRecurso.requerAtualizacao =>
        TemaSysAP.corDouradoPrincipal,
      _ => TemaSysAP.corAlerta,
    };

    return Card(
      color: TemaSysAP.corSuperficieEscura,
      child: Padding(
        padding: const EdgeInsets.all(16),
        child: Row(
          children: [
            Icon(
              Icons.health_and_safety_outlined,
              color: statusColor,
              size: 28,
            ),
            const SizedBox(width: 14),
            Expanded(
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Text(
                    'Fonte: ${widget.fonteSaude.nomeDaPlataforma == "health_connect" ? "Health Connect (Android)" : "Apple HealthKit (iOS)"}',
                    style: const TextStyle(
                      color: Colors.white,
                      fontWeight: FontWeight.w600,
                      fontSize: 14,
                    ),
                  ),
                  const SizedBox(height: 2),
                  Text(
                    _disponibilidadeFonte.rotuloEmPortugues,
                    style: TextStyle(
                      color: statusColor,
                      fontSize: 12,
                      fontWeight: FontWeight.w500,
                    ),
                  ),
                ],
              ),
            ),
          ],
        ),
      ),
    );
  }

  Widget _construirItemCategoria(CategoriaDeDadoSensivel categoria) {
    final concedido = _estadosConsentimento[categoria] ?? false;

    return Semantics(
      label: 'Consentimento para ${categoria.tituloEmPortugues}',
      value: concedido ? 'Concedido' : 'Não concedido',
      child: Card(
        color: TemaSysAP.corSuperficieEscura,
        margin: const EdgeInsets.only(bottom: 12),
        child: Padding(
          padding: const EdgeInsets.all(16),
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              Row(
                mainAxisAlignment: MainAxisAlignment.spaceBetween,
                children: [
                  Expanded(
                    child: Text(
                      categoria.tituloEmPortugues,
                      style: const TextStyle(
                        color: Colors.white,
                        fontWeight: FontWeight.bold,
                        fontSize: 14,
                      ),
                    ),
                  ),
                  Switch(
                    value: concedido,
                    activeThumbColor: TemaSysAP.corDouradoPrincipal,
                    onChanged: (val) => _alternarConsentimento(categoria, val),
                  ),
                ],
              ),
              const SizedBox(height: 6),
              Text(
                categoria.descricaoFinalidade,
                style: const TextStyle(
                  color: TemaSysAP.corTextoSecundario,
                  fontSize: 12,
                  height: 1.3,
                ),
              ),
            ],
          ),
        ),
      ),
    );
  }
}
