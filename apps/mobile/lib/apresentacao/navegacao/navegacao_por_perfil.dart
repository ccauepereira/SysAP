import 'package:flutter/material.dart';

import '../../aplicacao/sessao/gerenciador_de_sessao.dart';
import '../../dominio/entidades/identidade_do_usuario.dart';
import '../../dominio/entidades/papel_do_usuario.dart';
import '../../dominio/entidades/sessao_autenticada.dart';
import '../tema/tema_sysap.dart';

/// Roteador de apresentação que renderiza a casca segura correspondente ao papel confirmado pela API.
class NavegacaoPorPerfil extends StatelessWidget {
  final GerenciadorDeSessao gerenciadorDeSessao;
  final SessaoAutenticada sessao;
  final IdentidadeDoUsuario identidade;

  const NavegacaoPorPerfil({
    super.key,
    required this.gerenciadorDeSessao,
    required this.sessao,
    required this.identidade,
  });

  @override
  Widget build(BuildContext context) {
    if (!identidade.possuiAcessoAtivo) {
      return TelaDeAcessoNegado(
        motivo:
            'Seu acesso está suspenso ou sem vínculo ativo nesta organização.',
        gerenciadorDeSessao: gerenciadorDeSessao,
      );
    }

    switch (identidade.papelPrincipal) {
      case PapelDoUsuario.owner:
        return CascaDaAreaOwner(
          identidade: identidade,
          gerenciadorDeSessao: gerenciadorDeSessao,
        );
      case PapelDoUsuario.athlete:
        return CascaDaAreaAthlete(
          identidade: identidade,
          gerenciadorDeSessao: gerenciadorDeSessao,
        );
      case PapelDoUsuario.trainer:
        return CascaDaAreaTrainer(
          identidade: identidade,
          gerenciadorDeSessao: gerenciadorDeSessao,
        );
      case PapelDoUsuario.desconhecido:
        return TelaDeAcessoNegado(
          motivo: 'Papel do usuário não suportado ou desconhecido.',
          gerenciadorDeSessao: gerenciadorDeSessao,
        );
    }
  }
}

/// Casca segura da Área do Owner (Gestão).
class CascaDaAreaOwner extends StatelessWidget {
  final IdentidadeDoUsuario identidade;
  final GerenciadorDeSessao gerenciadorDeSessao;

  const CascaDaAreaOwner({
    super.key,
    required this.identidade,
    required this.gerenciadorDeSessao,
  });

  @override
  Widget build(BuildContext context) {
    return _LayoutCascaSegura(
      tituloArea: 'Área de Gestão — Artur Performance',
      rotuloPapel: 'Owner / Administrador',
      nomeUsuario: identidade.nomeDeExibicao,
      icone: Icons.admin_panel_settings_outlined,
      gerenciadorDeSessao: gerenciadorDeSessao,
      descricao: 'Centro de Comando Comercial e Operacional em preparação.\nOs módulos reais de turmas, ocupação e presença serão conectados a partir das próximas fases.',
    );
  }
}

/// Casca segura da Área do Atleta.
class CascaDaAreaAthlete extends StatelessWidget {
  final IdentidadeDoUsuario identidade;
  final GerenciadorDeSessao gerenciadorDeSessao;

  const CascaDaAreaAthlete({
    super.key,
    required this.identidade,
    required this.gerenciadorDeSessao,
  });

  @override
  Widget build(BuildContext context) {
    return _LayoutCascaSegura(
      tituloArea: 'Área do Atleta — SysAP',
      rotuloPapel: 'Atleta',
      nomeUsuario: identidade.nomeDeExibicao,
      icone: Icons.directions_run_outlined,
      gerenciadorDeSessao: gerenciadorDeSessao,
      descricao: 'Painel do atleta em preparação.\nAs rotas de treinos, confirmação de presença e integrações de sensores esportivos serão conectadas com consentimento explícito.',
    );
  }
}

/// Casca segura da Área do Treinador.
class CascaDaAreaTrainer extends StatelessWidget {
  final IdentidadeDoUsuario identidade;
  final GerenciadorDeSessao gerenciadorDeSessao;

  const CascaDaAreaTrainer({
    super.key,
    required this.identidade,
    required this.gerenciadorDeSessao,
  });

  @override
  Widget build(BuildContext context) {
    return _LayoutCascaSegura(
      tituloArea: 'Área Técnica — SysAP',
      rotuloPapel: 'Treinador',
      nomeUsuario: identidade.nomeDeExibicao,
      icone: Icons.sports_outlined,
      gerenciadorDeSessao: gerenciadorDeSessao,
      descricao: 'Painel técnico em preparação.\nA chamada e acompanhamento tático de turmas serão liberados nas fases subsequentes.',
    );
  }
}

/// Tela de Acesso Negado segura.
class TelaDeAcessoNegado extends StatelessWidget {
  final String motivo;
  final GerenciadorDeSessao gerenciadorDeSessao;

  const TelaDeAcessoNegado({
    super.key,
    required this.motivo,
    required this.gerenciadorDeSessao,
  });

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      backgroundColor: TemaSysAP.corFundoObsidiana,
      body: SafeArea(
        child: Center(
          child: Padding(
            padding: const EdgeInsets.all(24),
            child: Column(
              mainAxisAlignment: MainAxisAlignment.center,
              children: [
                const Icon(
                  Icons.lock_outline,
                  size: 64,
                  color: TemaSysAP.corAlerta,
                ),
                const SizedBox(height: 16),
                const Text(
                  'Acesso Não Autorizado',
                  style: TextStyle(
                    fontSize: 20,
                    fontWeight: FontWeight.bold,
                    color: Colors.white,
                  ),
                ),
                const SizedBox(height: 12),
                Text(
                  motivo,
                  textAlign: TextAlign.center,
                  style: const TextStyle(
                    color: TemaSysAP.corTextoSecundario,
                    fontSize: 14,
                  ),
                ),
                const SizedBox(height: 32),
                ElevatedButton.icon(
                  onPressed: () => gerenciadorDeSessao.logout(),
                  icon: const Icon(Icons.logout),
                  label: const Text('Voltar ao Login'),
                  style: ElevatedButton.styleFrom(
                    backgroundColor: TemaSysAP.corSuperficieEscura,
                    foregroundColor: Colors.white,
                  ),
                ),
              ],
            ),
          ),
        ),
      ),
    );
  }
}

class _LayoutCascaSegura extends StatelessWidget {
  final String tituloArea;
  final String rotuloPapel;
  final String nomeUsuario;
  final IconData icone;
  final String descricao;
  final GerenciadorDeSessao gerenciadorDeSessao;

  const _LayoutCascaSegura({
    required this.tituloArea,
    required this.rotuloPapel,
    required this.nomeUsuario,
    required this.icone,
    required this.descricao,
    required this.gerenciadorDeSessao,
  });

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      backgroundColor: TemaSysAP.corFundoObsidiana,
      appBar: AppBar(
        title: Text(tituloArea, style: const TextStyle(fontSize: 16)),
        actions: [
          IconButton(
            tooltip: 'Encerrar Sessão',
            icon: const Icon(Icons.logout, color: TemaSysAP.corTextoSecundario),
            onPressed: () => gerenciadorDeSessao.logout(),
          ),
        ],
      ),
      body: SafeArea(
        child: Padding(
          padding: const EdgeInsets.all(24),
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.stretch,
            children: [
              Card(
                color: TemaSysAP.corSuperficieEscura,
                child: Padding(
                  padding: const EdgeInsets.all(20),
                  child: Row(
                    children: [
                      CircleAvatar(
                        radius: 28,
                        backgroundColor: TemaSysAP.corDouradoPrincipal
                            .withValues(alpha: 0.15),
                        child: Icon(
                          icone,
                          color: TemaSysAP.corDouradoPrincipal,
                          size: 28,
                        ),
                      ),
                      const SizedBox(width: 16),
                      Expanded(
                        child: Column(
                          crossAxisAlignment: CrossAxisAlignment.start,
                          children: [
                            Text(
                              nomeUsuario.isNotEmpty
                                  ? nomeUsuario
                                  : 'Usuário Autenticado',
                              style: const TextStyle(
                                fontSize: 18,
                                fontWeight: FontWeight.bold,
                                color: Colors.white,
                              ),
                            ),
                            const SizedBox(height: 4),
                            Container(
                              padding: const EdgeInsets.symmetric(
                                horizontal: 8,
                                vertical: 2,
                              ),
                              decoration: BoxDecoration(
                                color: TemaSysAP.corDouradoPrincipal.withValues(
                                  alpha: 0.2,
                                ),
                                borderRadius: BorderRadius.circular(4),
                              ),
                              child: Text(
                                rotuloPapel,
                                style: const TextStyle(
                                  fontSize: 12,
                                  fontWeight: FontWeight.w600,
                                  color: TemaSysAP.corDouradoInteracao,
                                ),
                              ),
                            ),
                          ],
                        ),
                      ),
                    ],
                  ),
                ),
              ),
              const SizedBox(height: 24),
              Card(
                color: TemaSysAP.corSuperficieEscura,
                child: Padding(
                  padding: const EdgeInsets.all(20),
                  child: Column(
                    children: [
                      const Icon(
                        Icons.science_outlined,
                        size: 40,
                        color: TemaSysAP.corDouradoPrincipal,
                      ),
                      const SizedBox(height: 12),
                      const Text(
                        'Ambiente Seguro Confirmado',
                        style: TextStyle(
                          fontSize: 16,
                          fontWeight: FontWeight.bold,
                          color: Colors.white,
                        ),
                      ),
                      const SizedBox(height: 8),
                      Text(
                        descricao,
                        textAlign: TextAlign.center,
                        style: const TextStyle(
                          color: TemaSysAP.corTextoSecundario,
                          fontSize: 13,
                          height: 1.4,
                        ),
                      ),
                    ],
                  ),
                ),
              ),
              const Spacer(),
              ElevatedButton.icon(
                onPressed: () => gerenciadorDeSessao.logout(),
                icon: const Icon(Icons.logout),
                label: const Text('Encerrar Sessão'),
                style: ElevatedButton.styleFrom(
                  backgroundColor: TemaSysAP.corSuperficieEscura,
                  foregroundColor: Colors.white,
                  side: const BorderSide(color: TemaSysAP.corBordaTatica),
                  padding: const EdgeInsets.symmetric(vertical: 14),
                ),
              ),
            ],
          ),
        ),
      ),
    );
  }
}
