import 'dart:async';

import 'package:flutter/services.dart';

import '../../dominio/servicos/detector_de_conectividade.dart';

/// Implementação do detector de conectividade via canal de plataforma.
class DetectorDeConectividadeDaPlataforma implements DetectorDeConectividade {
  static const String nomeDoCanal = 'com.sysap.mobile/conectividade';
  final MethodChannel _canal;
  final StreamController<EstadoDeConectividade> _controlador =
      StreamController<EstadoDeConectividade>.broadcast();

  DetectorDeConectividadeDaPlataforma([MethodChannel? canal])
    : _canal = canal ?? const MethodChannel(nomeDoCanal);

  @override
  Future<EstadoDeConectividade> verificarConectividade() async {
    try {
      final resultado = await _canal.invokeMethod<String>(
        'verificarConectividade',
      );
      switch (resultado) {
        case 'online':
          return EstadoDeConectividade.online;
        case 'offline':
          return EstadoDeConectividade.offline;
        default:
          return EstadoDeConectividade.desconhecido;
      }
    } on PlatformException {
      return EstadoDeConectividade.online;
    } on MissingPluginException {
      return EstadoDeConectividade.online;
    }
  }

  @override
  Stream<EstadoDeConectividade> observarConectividade() => _controlador.stream;

  void emitirEstado(EstadoDeConectividade novoEstado) {
    if (!_controlador.isClosed) {
      _controlador.add(novoEstado);
    }
  }

  void dispose() {
    _controlador.close();
  }
}
