import 'package:flutter/services.dart';

import '../../dominio/servicos/notificacoes_do_dispositivo.dart';

/// Implementação de notificações locais via canal de plataforma nativo.
///
/// Não envia nem recebe dados através de serviços externos (Firebase/FCM/APNs).
class NotificacoesLocaisDaPlataforma implements NotificacoesDoDispositivo {
  static const String nomeDoCanal = 'com.sysap.mobile/notificacoes';
  final MethodChannel _canal;

  NotificacoesLocaisDaPlataforma([MethodChannel? canal])
    : _canal = canal ?? const MethodChannel(nomeDoCanal);

  @override
  Future<bool> solicitarPermissaoNotificacoes() async {
    try {
      final resultado = await _canal.invokeMethod<bool>('solicitarPermissao');
      return resultado ?? false;
    } on PlatformException {
      return false;
    } on MissingPluginException {
      return false;
    }
  }

  @override
  Future<bool> possuiPermissaoAtiva() async {
    try {
      final resultado = await _canal.invokeMethod<bool>('possuiPermissao');
      return resultado ?? false;
    } on PlatformException {
      return false;
    } on MissingPluginException {
      return false;
    }
  }

  @override
  Future<void> agendarNotificacaoLocal({
    required String id,
    required String titulo,
    required String corpo,
    required CategoriaDeNotificacao categoria,
    required DateTime agendadoPara,
  }) async {
    try {
      await _canal.invokeMethod('agendarNotificacao', {
        'id': id,
        'titulo': titulo,
        'corpo': corpo,
        'categoria': categoria.name,
        'agendadoParaMs': agendadoPara.millisecondsSinceEpoch,
      });
    } on PlatformException {
      // Falha segura: não propaga exceção para não quebrar fluxos do app
    } on MissingPluginException {
      // Falha segura
    }
  }

  @override
  Future<void> cancelarNotificacao(String id) async {
    try {
      await _canal.invokeMethod('cancelarNotificacao', {'id': id});
    } on PlatformException {
      // Falha segura
    } on MissingPluginException {
      // Falha segura
    }
  }
}
