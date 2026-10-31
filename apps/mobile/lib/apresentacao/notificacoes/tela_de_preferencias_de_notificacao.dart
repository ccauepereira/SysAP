import 'package:flutter/material.dart';

import '../../dominio/servicos/notificacoes_do_dispositivo.dart';
import '../tema/tema_sysap.dart';

/// Tela de preferências de notificações locais do SysAP.
class TelaDePreferenciasDeNotificacao extends StatefulWidget {
  final NotificacoesDoDispositivo servicoNotificacoes;

  const TelaDePreferenciasDeNotificacao({
    super.key,
    required this.servicoNotificacoes,
  });

  @override
  State<TelaDePreferenciasDeNotificacao> createState() =>
      _TelaDePreferenciasDeNotificacaoState();
}

class _TelaDePreferenciasDeNotificacaoState
    extends State<TelaDePreferenciasDeNotificacao> {
  bool _permissaoAtiva = false;
  bool _lembreteTreino = false;
  bool _lembretePresenca = false;
  bool _avisoExpiracao = true;
  bool _lembreteConsentimento = false;

  @override
  void initState() {
    super.initState();
    _verificarPermissao();
  }

  Future<void> _verificarPermissao() async {
    final ativa = await widget.servicoNotificacoes.possuiPermissaoAtiva();
    if (mounted) {
      setState(() => _permissaoAtiva = ativa);
    }
  }

  Future<void> _alternarLembrete(
    CategoriaDeNotificacao categoria,
    bool valor,
  ) async {
    if (valor && !_permissaoAtiva) {
      final concedeu = await widget.servicoNotificacoes
          .solicitarPermissaoNotificacoes();
      if (!concedeu) {
        if (mounted) {
          ScaffoldMessenger.of(context).showSnackBar(
            const SnackBar(
              content: Text(
                'Permissão de notificação recusada no sistema. O aplicativo continuará funcionando normalmente.',
              ),
              backgroundColor: TemaSysAP.corAlerta,
            ),
          );
        }
        return;
      }
      setState(() => _permissaoAtiva = true);
    }

    setState(() {
      switch (categoria) {
        case CategoriaDeNotificacao.lembreteDeTreino:
          _lembreteTreino = valor;
        case CategoriaDeNotificacao.lembreteDePresenca:
          _lembretePresenca = valor;
        case CategoriaDeNotificacao.avisoDeSessaoExpirada:
          _avisoExpiracao = valor;
        case CategoriaDeNotificacao.lembreteDeConsentimentoPendente:
          _lembreteConsentimento = valor;
      }
    });
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      backgroundColor: TemaSysAP.corFundoObsidiana,
      appBar: AppBar(
        title: const Text(
          'Notificações Locais',
          style: TextStyle(fontSize: 18, fontWeight: FontWeight.bold),
        ),
      ),
      body: SingleChildScrollView(
        padding: const EdgeInsets.all(20),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.stretch,
          children: [
            Card(
              color: TemaSysAP.corSuperficieEscura,
              child: Padding(
                padding: const EdgeInsets.all(16),
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: const [
                    Text(
                      'Lembretes Privados e Locais',
                      style: TextStyle(
                        color: Colors.white,
                        fontWeight: FontWeight.bold,
                        fontSize: 15,
                      ),
                    ),
                    SizedBox(height: 6),
                    Text(
                      'As notificações do SysAP são emitidas diretamente pelo seu aparelho para lembretes de treinos e presenças. Nenhuma mensagem ou dado de saúde é trafegado por servidores de push externos.',
                      style: TextStyle(
                        color: TemaSysAP.corTextoSecundario,
                        fontSize: 13,
                        height: 1.4,
                      ),
                    ),
                  ],
                ),
              ),
            ),
            const SizedBox(height: 20),
            _construirItemNotificacao(
              titulo: 'Lembrete de Início de Treino',
              descricao:
                  'Aviso 30 minutos antes do início de uma sessão agendada.',
              categoria: CategoriaDeNotificacao.lembreteDeTreino,
              valor: _lembreteTreino,
            ),
            _construirItemNotificacao(
              titulo: 'Confirmação de Presença Pendente',
              descricao:
                  'Lembrete para responder a chamada do treino realizado.',
              categoria: CategoriaDeNotificacao.lembreteDePresenca,
              valor: _lembretePresenca,
            ),
            _construirItemNotificacao(
              titulo: 'Aviso de Sessão Expirada',
              descricao:
                  'Notificação local quando sua sessão de login expirar.',
              categoria: CategoriaDeNotificacao.avisoDeSessaoExpirada,
              valor: _avisoExpiracao,
            ),
            _construirItemNotificacao(
              titulo: 'Lembrete de Consentimento Pendente',
              descricao: 'Aviso quando houver nova categoria de sensor pendente de análise.',
              categoria: CategoriaDeNotificacao.lembreteDeConsentimentoPendente,
              valor: _lembreteConsentimento,
            ),
          ],
        ),
      ),
    );
  }

  Widget _construirItemNotificacao({
    required String titulo,
    required String descricao,
    required CategoriaDeNotificacao categoria,
    required bool valor,
  }) {
    return Card(
      color: TemaSysAP.corSuperficieEscura,
      margin: const EdgeInsets.only(bottom: 12),
      child: Padding(
        padding: const EdgeInsets.all(16),
        child: Row(
          children: [
            Expanded(
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Text(
                    titulo,
                    style: const TextStyle(
                      color: Colors.white,
                      fontWeight: FontWeight.bold,
                      fontSize: 14,
                    ),
                  ),
                  const SizedBox(height: 4),
                  Text(
                    descricao,
                    style: const TextStyle(
                      color: TemaSysAP.corTextoSecundario,
                      fontSize: 12,
                    ),
                  ),
                ],
              ),
            ),
            Switch(
              value: valor,
              activeThumbColor: TemaSysAP.corDouradoPrincipal,
              onChanged: (novo) => _alternarLembrete(categoria, novo),
            ),
          ],
        ),
      ),
    );
  }
}
