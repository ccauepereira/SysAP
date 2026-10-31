/// Papéis confirmados pela API no estado autoritativo do SysAP.
enum PapelDoUsuario {
  owner('owner', 'Proprietário / Gestor'),
  athlete('athlete', 'Atleta'),
  trainer('trainer', 'Treinador'),
  desconhecido('desconhecido', 'Papel Desconhecido');

  final String valorApi;
  final String rotulo;

  const PapelDoUsuario(this.valorApi, this.rotulo);

  bool get ehOwner => this == PapelDoUsuario.owner;
  bool get ehAthlete => this == PapelDoUsuario.athlete;
  bool get ehTrainer => this == PapelDoUsuario.trainer;
  bool get ehValido => this != PapelDoUsuario.desconhecido;

  /// Converte o papel retornado pela API para o enum seguro.
  static PapelDoUsuario aPartirDeTexto(String? valor) {
    if (valor == null || valor.trim().isEmpty) {
      return PapelDoUsuario.desconhecido;
    }
    final normalizado = valor.trim().toLowerCase();
    switch (normalizado) {
      case 'owner':
        return PapelDoUsuario.owner;
      case 'athlete':
        return PapelDoUsuario.athlete;
      case 'trainer':
        return PapelDoUsuario.trainer;
      default:
        return PapelDoUsuario.desconhecido;
    }
  }
}
