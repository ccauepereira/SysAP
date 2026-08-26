import 'papel_do_usuario.dart';
import 'vinculo_organizacional.dart';

/// Identidade atual do usuário confirmada pelo endpoint /v1/me.
class IdentidadeDoUsuario {
  final String idDoPerfil;
  final String nomeDeExibicao;
  final List<VinculoOrganizacional> vinculos;

  const IdentidadeDoUsuario({
    required this.idDoPerfil,
    required this.nomeDeExibicao,
    required this.vinculos,
  });

  /// Retorna o vínculo principal ativo, se existir.
  VinculoOrganizacional? get vinculoPrincipal {
    if (vinculos.isEmpty) return null;
    try {
      return vinculos.firstWhere((v) => v.ehAtivo);
    } catch (_) {
      return vinculos.first;
    }
  }

  /// Retorna o papel principal confirmado e ativo.
  PapelDoUsuario get papelPrincipal =>
      vinculoPrincipal?.papel ?? PapelDoUsuario.desconhecido;

  bool get possuiAcessoAtivo => vinculos.any((v) => v.ehAtivo);

  factory IdentidadeDoUsuario.doJson(Map<String, dynamic> json) {
    final profile = (json['profile'] as Map<String, dynamic>?) ?? {};
    final membershipsList = (json['memberships'] as List<dynamic>?) ?? [];

    return IdentidadeDoUsuario(
      idDoPerfil: (profile['id'] as String?) ?? '',
      nomeDeExibicao: (profile['display_name'] as String?) ?? '',
      vinculos: membershipsList
          .whereType<Map<String, dynamic>>()
          .map(VinculoOrganizacional.doJson)
          .toList(),
    );
  }

  @override
  String toString() =>
      'IdentidadeDoUsuario(perfil: $idDoPerfil, nome: $nomeDeExibicao, vinculos: ${vinculos.length})';
}
