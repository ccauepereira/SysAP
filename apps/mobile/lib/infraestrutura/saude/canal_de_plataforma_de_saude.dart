import 'package:flutter/services.dart';

/// Adaptador tipado para comunicação com os canais de plataforma nativos de saúde
/// (Kotlin / Health Connect no Android e Swift / HealthKit no iOS).
class CanalDePlataformaDeSaude {
  static const String nomeDoCanal = 'com.sysap.mobile/saude';
  final MethodChannel _canal;

  CanalDePlataformaDeSaude([MethodChannel? canal])
    : _canal = canal ?? const MethodChannel(nomeDoCanal);

  /// Invoca verificação de disponibilidade nativa.
  Future<String> verificarDisponibilidade() async {
    try {
      final resultado = await _canal.invokeMethod<String>(
        'verificarDisponibilidade',
      );
      return resultado ?? 'indisponivel';
    } on PlatformException {
      return 'indisponivel';
    } on MissingPluginException {
      return 'indisponivel';
    }
  }

  /// Invoca pedido de permissões nativas para a lista de códigos de categorias.
  Future<bool> solicitarPermissoes(List<String> codigosCategorias) async {
    try {
      final resultado = await _canal.invokeMethod<bool>('solicitarPermissoes', {
        'categorias': codigosCategorias,
      });
      return resultado ?? false;
    } on PlatformException {
      return false;
    } on MissingPluginException {
      return false;
    }
  }

  /// Invoca leitura de dados agregados de treino no intervalo.
  Future<Map<String, dynamic>?> lerResumoDeTreino({
    required int inicioMilissegundos,
    required int fimMilissegundos,
    required List<String> codigosCategorias,
  }) async {
    try {
      final resultado = await _canal.invokeMapMethod<String, dynamic>(
        'lerResumoDeTreino',
        {
          'inicioMs': inicioMilissegundos,
          'fimMs': fimMilissegundos,
          'categorias': codigosCategorias,
        },
      );
      return resultado;
    } on PlatformException {
      return null;
    } on MissingPluginException {
      return null;
    }
  }
}
