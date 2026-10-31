import '../../dominio/falhas/falha_de_configuracao.dart';

/// Ambientes de execução suportados pelo SysAP Mobile.
enum AmbienteDeExecucao {
  desenvolvimento('Desenvolvimento'),
  homologacao('Homologação'),
  producao('Produção');

  final String rotulo;
  const AmbienteDeExecucao(this.rotulo);

  bool get ehDesenvolvimento => this == AmbienteDeExecucao.desenvolvimento;
  bool get ehHomologacao => this == AmbienteDeExecucao.homologacao;
  bool get ehProducao => this == AmbienteDeExecucao.producao;

  /// Converte texto para [AmbienteDeExecucao] de forma segura e determinística.
  static AmbienteDeExecucao aPartirDeTexto(String? valor) {
    if (valor == null || valor.trim().isEmpty) {
      return AmbienteDeExecucao.desenvolvimento;
    }
    final normalizado = valor.trim().toLowerCase();
    switch (normalizado) {
      case 'desenvolvimento':
      case 'development':
      case 'dev':
      case 'local':
        return AmbienteDeExecucao.desenvolvimento;
      case 'homologacao':
      case 'homologação':
      case 'staging':
      case 'hml':
        return AmbienteDeExecucao.homologacao;
      case 'producao':
      case 'produção':
      case 'production':
      case 'prod':
        return AmbienteDeExecucao.producao;
      default:
        throw const FalhaDeConfiguracao(
          mensagem: 'Ambiente de execução não reconhecido.',
          tipo: TipoDeFalhaDeConfiguracao.ambienteInvalido,
        );
    }
  }
}
