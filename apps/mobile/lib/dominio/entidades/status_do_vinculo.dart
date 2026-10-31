/// Estado de acesso da membership confirmado pela API.
enum StatusDoVinculo {
  ativo('active', 'Ativo'),
  suspenso('suspended', 'Suspenso'),
  desconhecido('unknown', 'Desconhecido');

  final String valorApi;
  final String rotulo;

  const StatusDoVinculo(this.valorApi, this.rotulo);

  bool get ehAtivo => this == StatusDoVinculo.ativo;
  bool get ehSuspenso => this == StatusDoVinculo.suspenso;

  static StatusDoVinculo aPartirDeTexto(String? valor) {
    if (valor == null || valor.trim().isEmpty) {
      return StatusDoVinculo.desconhecido;
    }
    switch (valor.trim().toLowerCase()) {
      case 'active':
        return StatusDoVinculo.ativo;
      case 'suspended':
        return StatusDoVinculo.suspenso;
      default:
        return StatusDoVinculo.desconhecido;
    }
  }
}
