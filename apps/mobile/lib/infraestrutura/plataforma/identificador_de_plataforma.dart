import 'package:flutter/foundation.dart';

/// Plataformas suportadas oficialmente pelo SysAP Mobile.
enum PlataformaDeExecucao {
  android('Android'),
  ios('iOS'),
  outro('Outro');

  final String rotulo;
  const PlataformaDeExecucao(this.rotulo);

  bool get ehAndroid => this == PlataformaDeExecucao.android;
  bool get ehIOS => this == PlataformaDeExecucao.ios;
}

/// Adaptador técnico de infraestrutura para identificação da plataforma de execução.
class IdentificadorDePlataforma {
  const IdentificadorDePlataforma();

  /// Identifica a plataforma de destino do Flutter de forma segura e desacoplada.
  PlataformaDeExecucao identificar() {
    switch (defaultTargetPlatform) {
      case TargetPlatform.android:
        return PlataformaDeExecucao.android;
      case TargetPlatform.iOS:
        return PlataformaDeExecucao.ios;
      default:
        return PlataformaDeExecucao.outro;
    }
  }
}
