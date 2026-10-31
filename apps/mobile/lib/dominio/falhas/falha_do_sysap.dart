/// Contrato base de falhas do domínio SysAP.
///
/// Não possui dependências do Flutter ou de bibliotecas externas.
abstract class FalhaDoSysAP implements Exception {
  final String mensagem;

  const FalhaDoSysAP(this.mensagem);

  @override
  String toString() => mensagem;
}
