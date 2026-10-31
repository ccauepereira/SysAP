import 'package:flutter/services.dart';

import '../../dominio/servicos/autenticacao_biometrica.dart';

/// Implementação da autenticação biométrica local via canais de plataforma nativos.
///
/// Atua apenas no desbloqueio da interface local, nunca persistindo templates biométricos.
class AutenticacaoBiometricaDaPlataforma implements AutenticacaoBiometrica {
  static const String nomeDoCanal = 'com.sysap.mobile/biometria';
  final MethodChannel _canal;

  AutenticacaoBiometricaDaPlataforma([MethodChannel? canal])
    : _canal = canal ?? const MethodChannel(nomeDoCanal);

  @override
  Future<bool> dispositivoSuportaBiometria() async {
    try {
      final resultado = await _canal.invokeMethod<bool>('suportaBiometria');
      return resultado ?? false;
    } on PlatformException {
      return false;
    } on MissingPluginException {
      return false;
    }
  }

  @override
  Future<List<TipoDeBiometria>> tiposDisponiveis() async {
    try {
      final resultado = await _canal.invokeListMethod<String>(
        'tiposDisponiveis',
      );
      if (resultado == null) return [TipoDeBiometria.nenhuma];

      return resultado.map((tipo) {
        switch (tipo) {
          case 'impressao_digital':
            return TipoDeBiometria.impressaoDigital;
          case 'reconhecimento_facial':
            return TipoDeBiometria.reconhecimentoFacial;
          default:
            return TipoDeBiometria.nenhuma;
        }
      }).toList();
    } on PlatformException {
      return [TipoDeBiometria.nenhuma];
    } on MissingPluginException {
      return [TipoDeBiometria.nenhuma];
    }
  }

  @override
  Future<bool> autenticarLocalmente({required String motivo}) async {
    try {
      final resultado = await _canal.invokeMethod<bool>('autenticar', {
        'motivo': motivo,
      });
      return resultado ?? false;
    } on PlatformException {
      return false;
    } on MissingPluginException {
      return false;
    }
  }
}
