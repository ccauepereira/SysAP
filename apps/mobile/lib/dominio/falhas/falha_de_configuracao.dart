import 'falha_do_sysap.dart';

/// Categorias possíveis de falha na validação da configuração pública.
enum TipoDeFalhaDeConfiguracao {
  urlAusente,
  urlInvalida,
  esquemaInseguroEmProducao,
  credenciaisEmbutidasProibidas,
  ambienteInvalido,
}

/// Falha de domínio para configurações inválidas, ausentes ou inseguras.
class FalhaDeConfiguracao extends FalhaDoSysAP {
  final TipoDeFalhaDeConfiguracao tipo;

  const FalhaDeConfiguracao({required String mensagem, required this.tipo})
    : super(mensagem);

  @override
  String toString() => 'FalhaDeConfiguracao($tipo): $mensagem';
}
