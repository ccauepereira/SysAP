import 'dart:async';

import 'package:flutter/foundation.dart';

import '../../dominio/entidades/identidade_do_usuario.dart';
import '../../dominio/entidades/sessao_autenticada.dart';
import '../../dominio/falhas/falha_de_autenticacao.dart';
import '../../dominio/repositorios/repositorio_de_sessao.dart';
import '../../infraestrutura/api/cliente_da_api.dart';

/// Estados possíveis da sessão no ciclo de vida do aplicativo.
sealed class EstadoDaSessao {
  const EstadoDaSessao();
}

final class SessaoInicializando extends EstadoDaSessao {
  const SessaoInicializando();
}

final class SessaoNaoAutenticada extends EstadoDaSessao {
  const SessaoNaoAutenticada();
}

final class SessaoAutenticando extends EstadoDaSessao {
  const SessaoAutenticando();
}

final class SessaoAtiva extends EstadoDaSessao {
  final SessaoAutenticada sessao;
  final IdentidadeDoUsuario identidade;

  const SessaoAtiva({required this.sessao, required this.identidade});
}

final class SessaoExpirada extends EstadoDaSessao {
  const SessaoExpirada();
}

final class SessaoAcessoNegado extends EstadoDaSessao {
  final String motivo;
  const SessaoAcessoNegado(this.motivo);
}

final class SessaoFalha extends EstadoDaSessao {
  final FalhaDeAutenticacao falha;
  const SessaoFalha(this.falha);
}

/// Orquestrador reativo responsável pelo ciclo de vida da sessão e autenticação.
class GerenciadorDeSessao extends ValueNotifier<EstadoDaSessao> {
  final ClienteDaApi clienteApi;
  final RepositorioDeSessao repositorioSessao;

  Completer<bool>? _renovacaoEmAndamento;

  GerenciadorDeSessao({
    required this.clienteApi,
    required this.repositorioSessao,
    EstadoDaSessao estadoInicial = const SessaoInicializando(),
  }) : super(estadoInicial);

  /// Inicializa e valida a sessão existente no cofre seguro.
  Future<void> inicializar() async {
    value = const SessaoInicializando();

    final sessao = await repositorioSessao.lerSessao();
    if (sessao == null) {
      value = const SessaoNaoAutenticada();
      return;
    }

    // Se o token estiver expirado ou muito próximo do vencimento, tenta renovar
    if (sessao.precisaRenovar) {
      final sucessoRenovacao = await renovarSessaoCoordenada(sessao);
      if (!sucessoRenovacao) {
        await repositorioSessao.limparSessao();
        value = const SessaoNaoAutenticada();
        return;
      }
    }

    final sessaoAtualizada = await repositorioSessao.lerSessao();
    if (sessaoAtualizada == null) {
      value = const SessaoNaoAutenticada();
      return;
    }

    try {
      final identidade = await consultarIdentidade(
        sessaoAtualizada.tokenDeAcesso,
      );
      if (!identidade.possuiAcessoAtivo) {
        value = const SessaoAcessoNegado(
          'Acesso suspenso ou sem vínculo ativo na organização.',
        );
        return;
      }
      value = SessaoAtiva(sessao: sessaoAtualizada, identidade: identidade);
    } on FalhaDeAutenticacao catch (e) {
      if (_ehFalhaDeCredenciaisOuSessao(e.tipo)) {
        await repositorioSessao.limparSessao();
        value = const SessaoNaoAutenticada();
      } else {
        value = SessaoFalha(e);
      }
    } catch (_) {
      value = const SessaoFalha(
        FalhaDeAutenticacao(
          mensagem: 'Falha ao confirmar identidade do usuário.',
          tipo: TipoDeFalhaDeAutenticacao.desconhecida,
        ),
      );
    }
  }

  bool _ehFalhaDeCredenciaisOuSessao(TipoDeFalhaDeAutenticacao tipo) =>
      tipo == TipoDeFalhaDeAutenticacao.credenciaisInvalidas ||
      tipo == TipoDeFalhaDeAutenticacao.sessaoRevogada ||
      tipo == TipoDeFalhaDeAutenticacao.sessaoExpirada;

  /// Executa o fluxo de login chamando a API oficial /v1/auth/login.
  Future<void> login({required String matricula, required String senha}) async {
    final matriculaLimpa = matricula.trim();
    if (matriculaLimpa.length != 10 ||
        !RegExp(r'^[0-9]{10}$').hasMatch(matriculaLimpa)) {
      value = const SessaoFalha(
        FalhaDeAutenticacao(
          mensagem: 'A matrícula deve conter exatamente 10 dígitos numéricos.',
          tipo: TipoDeFalhaDeAutenticacao.formatoInvalido,
        ),
      );
      return;
    }

    if (senha.isEmpty || senha.length < 8) {
      value = const SessaoFalha(
        FalhaDeAutenticacao(
          mensagem: 'A senha informada é inválida.',
          tipo: TipoDeFalhaDeAutenticacao.formatoInvalido,
        ),
      );
      return;
    }

    value = const SessaoAutenticando();

    try {
      final resposta = await clienteApi.post(
        '/v1/auth/login',
        corpo: {'enrollment_number': matriculaLimpa, 'password': senha},
      );

      final sessao = SessaoAutenticada.doLoginJson(resposta.corpoJson);
      await repositorioSessao.salvarSessao(sessao);

      final identidade = await consultarIdentidade(sessao.tokenDeAcesso);
      if (!identidade.possuiAcessoAtivo) {
        value = const SessaoAcessoNegado(
          'Acesso suspenso ou sem vínculo ativo na organização.',
        );
        return;
      }

      value = SessaoAtiva(sessao: sessao, identidade: identidade);
    } on FalhaDeAutenticacao catch (falha) {
      await repositorioSessao.limparSessao();
      value = SessaoFalha(falha);
    } catch (_) {
      await repositorioSessao.limparSessao();
      value = const SessaoFalha(
        FalhaDeAutenticacao(
          mensagem: 'Não foi possível completar o login.',
          tipo: TipoDeFalhaDeAutenticacao.desconhecida,
        ),
      );
    }
  }

  /// Consulta a identidade atual e perfil confirmado pela API (/v1/me).
  Future<IdentidadeDoUsuario> consultarIdentidade(String tokenDeAcesso) async {
    final resposta = await clienteApi.get('/v1/me', tokenBearer: tokenDeAcesso);
    return IdentidadeDoUsuario.doJson(resposta.corpoJson);
  }

  /// Renova a sessão usando o refresh token com deduplicação de chamadas concorrentes.
  Future<bool> renovarSessaoCoordenada([SessaoAutenticada? sessaoAtual]) async {
    if (_renovacaoEmAndamento != null) {
      return _renovacaoEmAndamento!.future;
    }

    final completer = Completer<bool>();
    _renovacaoEmAndamento = completer;

    try {
      final sessao = sessaoAtual ?? await repositorioSessao.lerSessao();
      if (sessao == null || sessao.tokenDeAtualizacao.isEmpty) {
        completer.complete(false);
        return false;
      }

      final resposta = await clienteApi.post(
        '/v1/auth/refresh',
        corpo: {'refresh_token': sessao.tokenDeAtualizacao},
      );

      final json = resposta.corpoJson;
      final novoAccessToken = (json['access_token'] as String?) ?? '';
      final novoRefreshToken = (json['refresh_token'] as String?) ?? '';
      final novoExpiresIn = (json['expires_in'] as int?) ?? 3600;

      if (novoAccessToken.isEmpty || novoRefreshToken.isEmpty) {
        completer.complete(false);
        return false;
      }

      final novaSessao = sessao.comNovosTokens(
        novoTokenDeAcesso: novoAccessToken,
        novoTokenDeAtualizacao: novoRefreshToken,
        novoExpiraEmSegundos: novoExpiresIn,
      );

      await repositorioSessao.salvarSessao(novaSessao);
      completer.complete(true);
      return true;
    } catch (_) {
      completer.complete(false);
      return false;
    } finally {
      _renovacaoEmAndamento = null;
    }
  }

  /// Encerra a sessão atual localmente e notifica a API remota.
  Future<void> logout() async {
    final sessao = await repositorioSessao.lerSessao();
    if (sessao != null && sessao.tokenDeAcesso.isNotEmpty) {
      try {
        await clienteApi.post(
          '/v1/auth/logout',
          tokenBearer: sessao.tokenDeAcesso,
        );
      } catch (_) {
        // Falhas remotas (ex: offline) não impedem o logout local imediato
      }
    }

    await repositorioSessao.limparSessao();
    value = const SessaoNaoAutenticada();
  }
}
