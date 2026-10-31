import 'package:flutter/material.dart';

import 'aplicacao/inicializacao/estado_de_inicializacao.dart';
import 'aplicacao/inicializacao/orquestrador_de_inicializacao.dart';
import 'aplicacao/sessao/gerenciador_de_sessao.dart';
import 'apresentacao/autenticacao/tela_de_login.dart';
import 'apresentacao/inicializacao/tela_de_inicializacao.dart';
import 'apresentacao/navegacao/navegacao_por_perfil.dart';
import 'apresentacao/tema/tema_sysap.dart';
import 'infraestrutura/api/cliente_da_api.dart';
import 'infraestrutura/armazenamento_seguro/armazenamento_seguro_de_sessao.dart';

void main() {
  WidgetsFlutterBinding.ensureInitialized();

  const orquestrador = OrquestradorDeInicializacao();
  final estadoInicial = orquestrador.inicializar();

  GerenciadorDeSessao? gerenciadorSessao;
  if (estadoInicial is EstadoPreparado) {
    final clienteApi = ClienteDaApi(configuracao: estadoInicial.configuracao);
    final repositorioSessao = ArmazenamentoSeguroDeSessao();
    gerenciadorSessao = GerenciadorDeSessao(
      clienteApi: clienteApi,
      repositorioSessao: repositorioSessao,
    );
    gerenciadorSessao.inicializar();
  }

  runApp(
    AplicativoSysAP(
      estadoInicial: estadoInicial,
      gerenciadorDeSessao: gerenciadorSessao,
    ),
  );
}

/// Ponto de entrada visual do SysAP Mobile com suporte a sessão e perfil.
class AplicativoSysAP extends StatelessWidget {
  final EstadoDeInicializacao estadoInicial;
  final GerenciadorDeSessao? gerenciadorDeSessao;

  const AplicativoSysAP({
    super.key,
    required this.estadoInicial,
    this.gerenciadorDeSessao,
  });

  @override
  Widget build(BuildContext context) {
    return MaterialApp(
      title: 'SysAP',
      debugShowCheckedModeBanner: false,
      theme: TemaSysAP.temaClaro,
      darkTheme: TemaSysAP.temaEscuro,
      themeMode: ThemeMode.dark,
      home: _construirTelaRaiz(),
    );
  }

  Widget _construirTelaRaiz() {
    if (estadoInicial is! EstadoPreparado || gerenciadorDeSessao == null) {
      return TelaDeInicializacao(estado: estadoInicial);
    }

    final gerenciador = gerenciadorDeSessao!;
    return ListenableBuilder(
      listenable: gerenciador,
      builder: (context, _) {
        final estadoSessao = gerenciador.value;

        switch (estadoSessao) {
          case SessaoInicializando():
            return const Scaffold(
              backgroundColor: TemaSysAP.corFundoObsidiana,
              body: Center(
                child: CircularProgressIndicator(
                  color: TemaSysAP.corDouradoPrincipal,
                ),
              ),
            );
          case SessaoAtiva(sessao: final s, identidade: final id):
            return NavegacaoPorPerfil(
              gerenciadorDeSessao: gerenciador,
              sessao: s,
              identidade: id,
            );
          case SessaoAcessoNegado(motivo: final m):
            return TelaDeAcessoNegado(
              motivo: m,
              gerenciadorDeSessao: gerenciador,
            );
          case SessaoNaoAutenticada():
          case SessaoAutenticando():
          case SessaoExpirada():
          case SessaoFalha():
            return TelaDeLogin(gerenciadorDeSessao: gerenciador);
        }
      },
    );
  }
}
