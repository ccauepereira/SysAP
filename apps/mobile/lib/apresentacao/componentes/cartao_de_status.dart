import 'package:flutter/material.dart';

/// Componente de apresentação acessível para exibir mensagens de status e bootstrap.
class CartaoDeStatus extends StatelessWidget {
  final IconData icone;
  final Color? corDoIcone;
  final String titulo;
  final String descricao;
  final Widget? acaoOuDetalhes;

  const CartaoDeStatus({
    super.key,
    required this.icone,
    this.corDoIcone,
    required this.titulo,
    required this.descricao,
    this.acaoOuDetalhes,
  });

  @override
  Widget build(BuildContext context) {
    final tema = Theme.of(context);
    final corIconeEfetiva = corDoIcone ?? tema.colorScheme.primary;

    return Semantics(
      label: '$titulo: $descricao',
      container: true,
      child: Card(
        margin: const EdgeInsets.symmetric(horizontal: 24, vertical: 12),
        child: Padding(
          padding: const EdgeInsets.all(24),
          child: Column(
            mainAxisSize: MainAxisSize.min,
            children: [
              Icon(icone, size: 48, color: corIconeEfetiva),
              const SizedBox(height: 16),
              Text(
                titulo,
                style: tema.textTheme.titleMedium?.copyWith(
                  fontWeight: FontWeight.bold,
                ),
                textAlign: TextAlign.center,
              ),
              const SizedBox(height: 8),
              Text(
                descricao,
                style: tema.textTheme.bodyMedium?.copyWith(
                  color: tema.colorScheme.onSurfaceVariant,
                ),
                textAlign: TextAlign.center,
              ),
              if (acaoOuDetalhes != null) ...[
                const SizedBox(height: 16),
                acaoOuDetalhes!,
              ],
            ],
          ),
        ),
      ),
    );
  }
}
