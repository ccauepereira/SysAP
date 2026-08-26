import 'package:flutter/material.dart';

import '../../aplicacao/inicializacao/estado_de_inicializacao.dart';
import '../componentes/cartao_de_status.dart';

/// Tela inicial de bootstrap responsável por apresentar o estado de carregamento e configuração.
class TelaDeInicializacao extends StatelessWidget {
  final EstadoDeInicializacao estado;

  const TelaDeInicializacao({super.key, required this.estado});

  @override
  Widget build(BuildContext context) {
    final tema = Theme.of(context);

    return Scaffold(
      appBar: AppBar(title: const Text('SysAP')),
      body: SafeArea(
        child: Center(
          child: SingleChildScrollView(
            child: switch (estado) {
              EstadoInicializando() => Semantics(
                label: 'Inicializando o aplicativo',
                child: const Column(
                  mainAxisSize: MainAxisSize.min,
                  children: [
                    CircularProgressIndicator(),
                    SizedBox(height: 16),
                    Text('Inicializando o aplicativo...'),
                  ],
                ),
              ),
              EstadoPreparado(:final configuracao) => CartaoDeStatus(
                icone: Icons.check_circle_outline,
                corDoIcone: tema.colorScheme.primary,
                titulo: 'SysAP Preparado',
                descricao:
                    'Ambiente ativo: ${configuracao.ambiente.rotulo}.\nFundação técnica inicializada com sucesso.',
              ),
              EstadoConfiguracaoPendente(:final mensagem) => CartaoDeStatus(
                icone: Icons.info_outline,
                corDoIcone: tema.colorScheme.secondary,
                titulo: 'Configuração Pendente',
                descricao: mensagem,
              ),
              EstadoFalha(:final falha) => CartaoDeStatus(
                icone: Icons.error_outline,
                corDoIcone: tema.colorScheme.error,
                titulo: 'Falha na Inicialização',
                descricao: falha.mensagem,
              ),
            },
          ),
        ),
      ),
    );
  }
}
