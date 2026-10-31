import 'categoria_de_dado_sensivel.dart';

/// Status do consentimento no SysAP.
enum StatusDoConsentimento { concedido, revogado, pendente }

/// Registro imutável de consentimento do atleta para uma categoria de dado sensível.
class ConsentimentoDeDados {
  final String id;
  final String atletaId;
  final String organizacaoId;
  final CategoriaDeDadoSensivel categoria;
  final String finalidade;
  final StatusDoConsentimento status;
  final String versaoDoTermo;
  final DateTime concedidoEm;
  final DateTime? revogadoEm;

  const ConsentimentoDeDados({
    required this.id,
    required this.atletaId,
    required this.organizacaoId,
    required this.categoria,
    required this.finalidade,
    required this.status,
    required this.versaoDoTermo,
    required this.concedidoEm,
    this.revogadoEm,
  });

  /// Indica se o consentimento está atualmente válido e ativo.
  bool get estaVigente =>
      status == StatusDoConsentimento.concedido && revogadoEm == null;

  ConsentimentoDeDados revogar({required DateTime dataDaRevogacao}) {
    return ConsentimentoDeDados(
      id: id,
      atletaId: atletaId,
      organizacaoId: organizacaoId,
      categoria: categoria,
      finalidade: finalidade,
      status: StatusDoConsentimento.revogado,
      versaoDoTermo: versaoDoTermo,
      concedidoEm: concedidoEm,
      revogadoEm: dataDaRevogacao.toUtc(),
    );
  }

  static ConsentimentoDeDados criarNovo({
    required String id,
    required String atletaId,
    required String organizacaoId,
    required CategoriaDeDadoSensivel categoria,
    required String finalidade,
    required String versaoDoTermo,
    required DateTime dataDeConcessao,
  }) {
    return ConsentimentoDeDados(
      id: id,
      atletaId: atletaId,
      organizacaoId: organizacaoId,
      categoria: categoria,
      finalidade: finalidade,
      status: StatusDoConsentimento.concedido,
      versaoDoTermo: versaoDoTermo,
      concedidoEm: dataDeConcessao.toUtc(),
      revogadoEm: null,
    );
  }

  @override
  bool operator ==(Object other) =>
      identical(this, other) ||
      other is ConsentimentoDeDados &&
          runtimeType == other.runtimeType &&
          id == other.id &&
          atletaId == other.atletaId &&
          organizacaoId == other.organizacaoId &&
          categoria == other.categoria &&
          status == other.status &&
          versaoDoTermo == other.versaoDoTermo &&
          concedidoEm == other.concedidoEm &&
          revogadoEm == other.revogadoEm;

  @override
  int get hashCode =>
      id.hashCode ^
      atletaId.hashCode ^
      organizacaoId.hashCode ^
      categoria.hashCode ^
      status.hashCode ^
      versaoDoTermo.hashCode ^
      concedidoEm.hashCode ^
      revogadoEm.hashCode;
}
