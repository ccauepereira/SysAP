import 'papel_do_usuario.dart';
import 'status_do_vinculo.dart';

/// Vínculo organizacional e papel do usuário retornado por /v1/me.
class VinculoOrganizacional {
  final String idDaOrganizacao;
  final PapelDoUsuario papel;
  final StatusDoVinculo status;

  const VinculoOrganizacional({
    required this.idDaOrganizacao,
    required this.papel,
    required this.status,
  });

  bool get ehAtivo => status.ehAtivo && papel.ehValido;

  factory VinculoOrganizacional.doJson(Map<String, dynamic> json) {
    return VinculoOrganizacional(
      idDaOrganizacao: (json['organization_id'] as String?) ?? '',
      papel: PapelDoUsuario.aPartirDeTexto(json['role'] as String?),
      status: StatusDoVinculo.aPartirDeTexto(json['status'] as String?),
    );
  }

  @override
  String toString() =>
      'VinculoOrganizacional(org: $idDaOrganizacao, papel: ${papel.valorApi}, status: ${status.valorApi})';
}
