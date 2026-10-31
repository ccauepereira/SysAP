import '../../dominio/falhas/falha_de_configuracao.dart';
import '../configuracao/configuracao_publica.dart';
import 'estado_de_inicializacao.dart';

/// Orquestrador responsável por coordenar a inicialização segura do aplicativo.
class OrquestradorDeInicializacao {
  const OrquestradorDeInicializacao();

  /// Executa a validação e determina o estado inicial do aplicativo.
  EstadoDeInicializacao inicializar({
    String? urlBruta,
    String? ambienteBruto,
    bool usarVariaveisDeBuild = true,
  }) {
    if (urlBruta != null) {
      try {
        final config = ConfiguracaoPublica.validar(
          urlBruta: urlBruta,
          ambienteBruto: ambienteBruto,
        );
        return EstadoPreparado(config);
      } on FalhaDeConfiguracao catch (falha) {
        return EstadoFalha(falha);
      } catch (_) {
        return const EstadoFalha(
          FalhaDeConfiguracao(
            mensagem: 'Falha desconhecida durante a validação da configuração.',
            tipo: TipoDeFalhaDeConfiguracao.urlInvalida,
          ),
        );
      }
    }

    if (usarVariaveisDeBuild) {
      try {
        final config = ConfiguracaoPublica.tentarDoAmbienteDeBuild();
        if (config == null) {
          return const EstadoConfiguracaoPendente();
        }
        return EstadoPreparado(config);
      } on FalhaDeConfiguracao catch (falha) {
        return EstadoFalha(falha);
      } catch (_) {
        return const EstadoFalha(
          FalhaDeConfiguracao(
            mensagem: 'Falha desconhecida ao ler variáveis de ambiente.',
            tipo: TipoDeFalhaDeConfiguracao.urlInvalida,
          ),
        );
      }
    }

    return const EstadoConfiguracaoPendente();
  }
}
