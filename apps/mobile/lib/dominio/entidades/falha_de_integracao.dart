import '../falhas/falha_do_sysap.dart';

/// Falhas de integração relacionadas a sensores, consentimento, saúde, biometria e conectividade.
sealed class FalhaDeIntegracao extends FalhaDoSysAP {
  const FalhaDeIntegracao(super.mensagem);
}

/// Falha quando uma operação sensível é tentada sem consentimento prévio do atleta.
class FalhaDeConsentimentoAusente extends FalhaDeIntegracao {
  const FalhaDeConsentimentoAusente([
    super.mensagem =
        'Consentimento explícito não concedido para esta categoria de dado.',
  ]);
}

/// Falha quando o consentimento foi revogado pelo atleta.
class FalhaDeConsentimentoRevogado extends FalhaDeIntegracao {
  const FalhaDeConsentimentoRevogado([
    super.mensagem = 'Consentimento foi revogado pelo atleta; novas leituras estão bloqueadas.',
  ]);
}

/// Falha quando a permissão no sistema operacional (Android/iOS) foi negada.
class FalhaDePermissaoDeSistema extends FalhaDeIntegracao {
  const FalhaDePermissaoDeSistema([
    super.mensagem = 'Permissão negada pelo sistema operacional.',
  ]);
}

/// Falha quando a fonte de dados (Health Connect / HealthKit) está indisponível ou desatualizada.
class FalhaDeFonteIndisponivel extends FalhaDeIntegracao {
  const FalhaDeFonteIndisponivel([
    super.mensagem =
        'Fonte de dados de saúde indisponível ou requer atualização.',
  ]);
}

/// Falha durante validação de biometria local.
class FalhaDeBiometria extends FalhaDeIntegracao {
  const FalhaDeBiometria([
    super.mensagem = 'Falha na autenticação biométrica local.',
  ]);
}

/// Falha de conectividade ou rede offline.
class FalhaDeConectividade extends FalhaDeIntegracao {
  const FalhaDeConectividade([
    super.mensagem = 'Dispositivo offline ou conectividade indisponível.',
  ]);
}

/// Falha de autorização: tentativa indevida de acesso por Owner sem consentimento ou vínculo ativo.
class FalhaDeAutorizacaoOwner extends FalhaDeIntegracao {
  const FalhaDeAutorizacaoOwner([
    super.mensagem = 'Acesso individual não autorizado. Exige papel ativo, mesma organização e consentimento vigente do atleta.',
  ]);
}
