/// Categorias de notificações locais suportadas.
enum CategoriaDeNotificacao {
  lembreteDeTreino,
  lembreteDePresenca,
  avisoDeSessaoExpirada,
  lembreteDeConsentimentoPendente,
}

/// Porta de serviço para gerenciamento de notificações locais consentidas.
///
/// Não utiliza serviços em nuvem (FCM/APNs/Firebase) nesta fase.
abstract class NotificacoesDoDispositivo {
  /// Solicita ao sistema operacional a permissão para emitir notificações locais.
  Future<bool> solicitarPermissaoNotificacoes();

  /// Confirma se o app possui permissão ativa para notificações.
  Future<bool> possuiPermissaoAtiva();

  /// Agenda uma notificação local segura.
  Future<void> agendarNotificacaoLocal({
    required String id,
    required String titulo,
    required String corpo,
    required CategoriaDeNotificacao categoria,
    required DateTime agendadoPara,
  });

  /// Cancela uma notificação agendada.
  Future<void> cancelarNotificacao(String id);
}
