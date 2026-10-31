import 'papel_do_usuario.dart';

/// Modelo de sessão autenticada do SysAP Mobile.
///
/// Mantém os tokens de acesso e refresh em memória enquanto ativos.
/// Sua representação textual nunca expõe tokens ou segredos.
class SessaoAutenticada {
  final String tokenDeAcesso;
  final String tokenDeAtualizacao;
  final String idDaSessao;
  final String idDoPerfil;
  final String idDaOrganizacao;
  final PapelDoUsuario papel;
  final int expiraEmSegundos;
  final DateTime criadoEm;

  const SessaoAutenticada({
    required this.tokenDeAcesso,
    required this.tokenDeAtualizacao,
    required this.idDaSessao,
    required this.idDoPerfil,
    required this.idDaOrganizacao,
    required this.papel,
    required this.expiraEmSegundos,
    required this.criadoEm,
  });

  /// Verifica se o token de acesso expirou.
  bool get expirou {
    final expiracao = criadoEm.add(Duration(seconds: expiraEmSegundos));
    return DateTime.now().toUtc().isAfter(expiracao);
  }

  /// Indica se a sessão está próxima do vencimento (60 segundos de margem de segurança).
  bool get precisaRenovar {
    final margem = expiraEmSegundos > 60
        ? expiraEmSegundos - 60
        : expiraEmSegundos ~/ 2;
    final limiteRenovacao = criadoEm.add(Duration(seconds: margem));
    return DateTime.now().toUtc().isAfter(limiteRenovacao);
  }

  /// Retorna uma cópia atualizada da sessão com novos tokens após rotação.
  SessaoAutenticada comNovosTokens({
    required String novoTokenDeAcesso,
    required String novoTokenDeAtualizacao,
    required int novoExpiraEmSegundos,
  }) {
    return SessaoAutenticada(
      tokenDeAcesso: novoTokenDeAcesso,
      tokenDeAtualizacao: novoTokenDeAtualizacao,
      idDaSessao: idDaSessao,
      idDoPerfil: idDoPerfil,
      idDaOrganizacao: idDaOrganizacao,
      papel: papel,
      expiraEmSegundos: novoExpiraEmSegundos,
      criadoEm: DateTime.now().toUtc(),
    );
  }

  factory SessaoAutenticada.doLoginJson(Map<String, dynamic> json) {
    return SessaoAutenticada(
      tokenDeAcesso: (json['access_token'] as String?) ?? '',
      tokenDeAtualizacao: (json['refresh_token'] as String?) ?? '',
      idDaSessao: (json['session_id'] as String?) ?? '',
      idDoPerfil: (json['profile_id'] as String?) ?? '',
      idDaOrganizacao: (json['organization_id'] as String?) ?? '',
      papel: PapelDoUsuario.aPartirDeTexto(json['role'] as String?),
      expiraEmSegundos: (json['expires_in'] as int?) ?? 3600,
      criadoEm: DateTime.now().toUtc(),
    );
  }

  factory SessaoAutenticada.fromJson(Map<String, dynamic> json) {
    return SessaoAutenticada(
      tokenDeAcesso: (json['access_token'] as String?) ?? '',
      tokenDeAtualizacao: (json['refresh_token'] as String?) ?? '',
      idDaSessao: (json['session_id'] as String?) ?? '',
      idDoPerfil: (json['profile_id'] as String?) ?? '',
      idDaOrganizacao: (json['organization_id'] as String?) ?? '',
      papel: PapelDoUsuario.aPartirDeTexto(json['role'] as String?),
      expiraEmSegundos: (json['expires_in'] as int?) ?? 3600,
      criadoEm: json['criado_em'] != null
          ? DateTime.parse(json['criado_em'] as String).toUtc()
          : DateTime.now().toUtc(),
    );
  }

  Map<String, dynamic> toJson() {
    return {
      'access_token': tokenDeAcesso,
      'refresh_token': tokenDeAtualizacao,
      'session_id': idDaSessao,
      'profile_id': idDoPerfil,
      'organization_id': idDaOrganizacao,
      'role': papel.valorApi,
      'expires_in': expiraEmSegundos,
      'criado_em': criadoEm.toIso8601String(),
    };
  }

  @override
  String toString() =>
      'SessaoAutenticada(papel: ${papel.valorApi}, org: $idDaOrganizacao, expiraEm: ${expiraEmSegundos}s, ativa: ${!expirou})';
}
