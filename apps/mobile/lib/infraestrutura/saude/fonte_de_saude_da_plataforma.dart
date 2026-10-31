import '../../dominio/entidades/categoria_de_dado_sensivel.dart';
import '../../dominio/entidades/disponibilidade_de_recurso.dart';
import '../../dominio/entidades/resumo_de_atividade.dart';
import '../../dominio/servicos/fonte_de_dados_de_saude.dart';
import '../plataforma/identificador_de_plataforma.dart';
import 'fonte_health_connect.dart';
import 'fonte_healthkit.dart';

/// Adaptador unificado que resolve a fonte de saúde nativa conforme a plataforma detectada.
class FonteDeSaudeDaPlataforma implements FonteDeDadosDeSaude {
  final FonteDeDadosDeSaude _delegado;

  FonteDeSaudeDaPlataforma([FonteDeDadosDeSaude? delegado])
    : _delegado =
          delegado ??
          (const IdentificadorDePlataforma().identificar().ehIOS
              ? FonteHealthKit()
              : FonteHealthConnect());

  @override
  String get nomeDaPlataforma => _delegado.nomeDaPlataforma;

  @override
  Future<DisponibilidadeDeRecurso> verificarDisponibilidade() =>
      _delegado.verificarDisponibilidade();

  @override
  Future<bool> solicitarPermissoes(Set<CategoriaDeDadoSensivel> categorias) =>
      _delegado.solicitarPermissoes(categorias);

  @override
  Future<ResumoDeAtividade?> lerResumoDeTreino({
    required DateTime inicio,
    required DateTime fim,
    required Set<CategoriaDeDadoSensivel> categoriasAutorizadas,
  }) => _delegado.lerResumoDeTreino(
    inicio: inicio,
    fim: fim,
    categoriasAutorizadas: categoriasAutorizadas,
  );
}
