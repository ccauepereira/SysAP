import '../../dominio/entidades/categoria_de_dado_sensivel.dart';
import '../../dominio/entidades/disponibilidade_de_recurso.dart';
import '../../dominio/entidades/resumo_de_atividade.dart';
import '../../dominio/servicos/fonte_de_dados_de_saude.dart';
import 'canal_de_plataforma_de_saude.dart';

/// Adaptador para a fonte de dados Apple HealthKit (iOS).
///
/// Regra estrita de privacidade do iOS:
/// A ausência de dados lidos nunca é interpretada como negação deliberada de permissão
/// (o iOS preserva a privacidade do usuário não confirmando se negou permissão de leitura);
/// é tratada com segurança como "indisponível ou dados insuficientes".
class FonteHealthKit implements FonteDeDadosDeSaude {
  final CanalDePlataformaDeSaude _canal;

  FonteHealthKit([CanalDePlataformaDeSaude? canal])
    : _canal = canal ?? CanalDePlataformaDeSaude();

  @override
  String get nomeDaPlataforma => 'healthkit';

  @override
  Future<DisponibilidadeDeRecurso> verificarDisponibilidade() async {
    final statusCodigo = await _canal.verificarDisponibilidade();
    switch (statusCodigo) {
      case 'disponivel':
        return DisponibilidadeDeRecurso.disponivel;
      case 'nao_solicitado':
        return DisponibilidadeDeRecurso.naoSolicitado;
      case 'indisponivel':
      default:
        // No iOS, qualquer falha ou falta de suporte é tratada de forma fechada como indisponível
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

    if (mapa == null || mapa.isEmpty) {
      // Ausência de dados não é erro; é tratada como retorno nulo (sem dados disponíveis)
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
