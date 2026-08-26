import '../../dominio/falhas/falha_de_configuracao.dart';
import 'ambiente.dart';

/// Configuração pública e imutável de execução do SysAP Mobile.
///
/// Não contém nem aceita dados sensíveis, chaves ou tokens.
class ConfiguracaoPublica {
  final Uri urlDaApi;
  final AmbienteDeExecucao ambiente;

  const ConfiguracaoPublica({required this.urlDaApi, required this.ambiente});

  /// Valida e constrói [ConfiguracaoPublica] a partir de strings brutas.
  static ConfiguracaoPublica validar({
    required String? urlBruta,
    required String? ambienteBruto,
  }) {
    if (urlBruta == null || urlBruta.trim().isEmpty) {
      throw const FalhaDeConfiguracao(
        mensagem: 'A URL pública da API não foi informada.',
        tipo: TipoDeFalhaDeConfiguracao.urlAusente,
      );
    }

    final ambiente = AmbienteDeExecucao.aPartirDeTexto(ambienteBruto);

    final Uri uri;
    try {
      uri = Uri.parse(urlBruta.trim());
    } catch (_) {
      throw const FalhaDeConfiguracao(
        mensagem: 'A URL pública da API é inválida ou malformada.',
        tipo: TipoDeFalhaDeConfiguracao.urlInvalida,
      );
    }

    if (!uri.hasScheme || (uri.scheme != 'https' && uri.scheme != 'http')) {
      throw const FalhaDeConfiguracao(
        mensagem: 'A URL da API deve utilizar esquema HTTP ou HTTPS.',
        tipo: TipoDeFalhaDeConfiguracao.urlInvalida,
      );
    }

    if (uri.host.isEmpty) {
      throw const FalhaDeConfiguracao(
        mensagem: 'A URL da API deve conter um host válido.',
        tipo: TipoDeFalhaDeConfiguracao.urlInvalida,
      );
    }

    if (uri.userInfo.isNotEmpty) {
      throw const FalhaDeConfiguracao(
        mensagem: 'A URL da API não pode conter credenciais embutidas.',
        tipo: TipoDeFalhaDeConfiguracao.credenciaisEmbutidasProibidas,
      );
    }

    if (ambiente.ehProducao && uri.scheme != 'https') {
      throw const FalhaDeConfiguracao(
        mensagem: 'O ambiente de produção exige obrigatoriamente conexão segura HTTPS.',
        tipo: TipoDeFalhaDeConfiguracao.esquemaInseguroEmProducao,
      );
    }

    return ConfiguracaoPublica(urlDaApi: uri, ambiente: ambiente);
  }

  /// Lê a configuração a partir de variáveis de build (--dart-define).
  static ConfiguracaoPublica? tentarDoAmbienteDeBuild() {
    const url = String.fromEnvironment('SYSAP_API_URL');
    const env = String.fromEnvironment('SYSAP_ENV');

    if (url.trim().isEmpty) {
      return null;
    }

    return validar(urlBruta: url, ambienteBruto: env);
  }

  @override
  String toString() =>
      'ConfiguracaoPublica(ambiente: ${ambiente.rotulo}, url: ${urlDaApi.scheme}://${urlDaApi.host}${urlDaApi.hasPort ? ':${urlDaApi.port}' : ''})';

  @override
  bool operator ==(Object other) =>
      identical(this, other) ||
      other is ConfiguracaoPublica &&
          runtimeType == other.runtimeType &&
          urlDaApi == other.urlDaApi &&
          ambiente == other.ambiente;

  @override
  int get hashCode => urlDaApi.hashCode ^ ambiente.hashCode;
}
