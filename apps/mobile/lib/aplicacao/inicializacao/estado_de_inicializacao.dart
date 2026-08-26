import '../../dominio/falhas/falha_de_configuracao.dart';
import '../configuracao/configuracao_publica.dart';

/// Estados possíveis do ciclo de inicialização (bootstrap) do SysAP Mobile.
sealed class EstadoDeInicializacao {
  const EstadoDeInicializacao();
}

/// Estado temporário durante a leitura e validação do bootstrap.
final class EstadoInicializando extends EstadoDeInicializacao {
  const EstadoInicializando();
}

/// Estado em que o aplicativo foi validado com sucesso e está pronto.
final class EstadoPreparado extends EstadoDeInicializacao {
  final ConfiguracaoPublica configuracao;

  const EstadoPreparado(this.configuracao);
}

/// Estado neutro de desenvolvimento quando a URL pública ainda não foi injetada via --dart-define.
final class EstadoConfiguracaoPendente extends EstadoDeInicializacao {
  final String mensagem;

  const EstadoConfiguracaoPendente([
    this.mensagem = 'Configuração pública pendente para desenvolvimento.',
  ]);
}

/// Estado seguro de falha na inicialização por parâmetros inválidos ou inseguros.
final class EstadoFalha extends EstadoDeInicializacao {
  final FalhaDeConfiguracao falha;

  const EstadoFalha(this.falha);
}
