import 'falha_do_sysap.dart';

/// Categorias seguras de falhas durante autenticação e validação de sessão.
enum TipoDeFalhaDeAutenticacao {
  credenciaisInvalidas,
  sessaoExpirada,
  sessaoRevogada,
  acessoNegado,
  membershipInativa,
  mfaRequerido,
  formatoInvalido,
  servicoIndisponivel,
  redeIndisponivel,
  falhaDeArmazenamento,
  desconhecida,
}

/// Falha de domínio para operações de autenticação e sessão.
class FalhaDeAutenticacao extends FalhaDoSysAP {
  final TipoDeFalhaDeAutenticacao tipo;
  final int? codigoHttp;

  const FalhaDeAutenticacao({
    required String mensagem,
    required this.tipo,
    this.codigoHttp,
  }) : super(mensagem);

  @override
  String toString() =>
      'FalhaDeAutenticacao($tipo, http: $codigoHttp): $mensagem';
}
