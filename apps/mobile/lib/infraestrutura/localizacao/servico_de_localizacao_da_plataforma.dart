import 'package:flutter/services.dart';

import '../../dominio/entidades/disponibilidade_de_recurso.dart';
import '../../dominio/servicos/servico_de_localizacao.dart';

/// Implementação da porta de localização e GPS.
///
/// Mantém o recurso em estado seguro sem iniciar rastreamento ou solicitar permissão de sistema.
class ServicoDeLocalizacaoDaPlataforma implements ServicoDeLocalizacao {
  static const String nomeDoCanal = 'com.sysap.mobile/localizacao';
  final MethodChannel _canal;

  ServicoDeLocalizacaoDaPlataforma([MethodChannel? canal])
    : _canal = canal ?? const MethodChannel(nomeDoCanal);

  @override
  Future<DisponibilidadeDeRecurso> verificarCapacidadeGps() async {
    try {
      final resultado = await _canal.invokeMethod<String>(
        'verificarCapacidade',
      );
      switch (resultado) {
        case 'disponivel':
          return DisponibilidadeDeRecurso.disponivel;
        case 'indisponivel':
          return DisponibilidadeDeRecurso.indisponivel;
        case 'permissao_negada':
          return DisponibilidadeDeRecurso.permissaoNegada;
        default:
          return DisponibilidadeDeRecurso.naoSolicitado;
      }
    } on PlatformException {
      return DisponibilidadeDeRecurso.naoSolicitado;
    } on MissingPluginException {
      return DisponibilidadeDeRecurso.naoSolicitado;
    }
  }
}
