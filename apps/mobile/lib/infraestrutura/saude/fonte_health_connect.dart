import '../../dominio/entidades/categoria_de_dado_sensivel.dart';
import '../../dominio/entidades/disponibilidade_de_recurso.dart';
import '../../dominio/entidades/resumo_de_atividade.dart';
import '../../dominio/servicos/fonte_de_dados_de_saude.dart';
import 'canal_de_plataforma_de_saude.dart';

/// Adaptador para a fonte de dados Health Connect (Android).
class FonteHealthConnect implements FonteDeDadosDeSaude {
  final CanalDePlataformaDeSaude _canal;

  FonteHealthConnect([CanalDePlataformaDeSaude? canal])
    : _canal = canal ?? CanalDePlataformaDeSaude();

  @override
  String get nomeDaPlataforma => 'health_connect';

  @override
  Future<DisponibilidadeDeRecurso> verificarDisponibilidade() async {
    final statusCodigo = await _canal.verificarDisponibilidade();
    switch (statusCodigo) {
      case 'disponivel':
        return DisponibilidadeDeRecurso.disponivel;
      case 'requer_atualizacao':
        return DisponibilidadeDeRecurso.requerAtualizacao;
      case 'permissao_negada':
        return DisponibilidadeDeRecurso.permissaoNegada;
      case 'nao_solicitado':
        return DisponibilidadeDeRecurso.naoSolicitado;
      default:
        return DisponibilidadeDeRecurso.indisponivel;
    }
  }

  @override
  Future<bool> solicitarPermissoes(
    Set<CategoriaDeDadoSensivel> categorias,
  ) async {
    final codigos = categorias.map((c) => c.codigo).toList();
    return _canal.solicitarPermissoes(codigos);
  }

  @override
  Future<ResumoDeAtividade?> lerResumoDeTreino({
    required DateTime inicio,
    required DateTime fim,
    required Set<CategoriaDeDadoSensivel> categoriasAutorizadas,
  }) async {
    final codigos = categoriasAutorizadas.map((c) => c.codigo).toList();
    final mapa = await _canal.lerResumoDeTreino(
      inicioMilissegundos: inicio.millisecondsSinceEpoch,
      fimMilissegundos: fim.millisecondsSinceEpoch,
      codigosCategorias: codigos,
    );

    if (mapa == null) {
      return null;
    }

    final duracaoMs =
        mapa['duracaoMs'] as int? ?? fim.difference(inicio).inMilliseconds;
    return ResumoDeAtividade(
      inicio: inicio,
      fim: fim,
      duracao: Duration(milliseconds: duracaoMs),
      passos: mapa['passos'] as int?,
      distanciaMetros: (mapa['distanciaMetros'] as num?)?.toDouble(),
      frequenciaCardiacaMediaBpm: mapa['frequenciaCardiacaMediaBpm'] as int?,
      frequenciaCardiacaMaximaBpm: mapa['frequenciaCardiacaMaximaBpm'] as int?,
      origem: nomeDaPlataforma,
    );
  }
}
