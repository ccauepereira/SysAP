import 'package:flutter/material.dart';

import '../../dominio/servicos/autenticacao_biometrica.dart';
import '../tema/tema_sysap.dart';

/// Componente para exibição e controle do desbloqueio biométrico local.
class ComponenteDeBloqueioBiometrico extends StatefulWidget {
  final AutenticacaoBiometrica biometria;

  const ComponenteDeBloqueioBiometrico({super.key, required this.biometria});

  @override
  State<ComponenteDeBloqueioBiometrico> createState() =>
      _ComponenteDeBloqueioBiometricoState();
}

class _ComponenteDeBloqueioBiometricoState
    extends State<ComponenteDeBloqueioBiometrico> {
  bool _suportaBiometria = false;
  bool _autenticado = false;
  String? _mensagemStatus;

  @override
  void initState() {
    super.initState();
    _verificarHardware();
  }

  Future<void> _verificarHardware() async {
    final suporta = await widget.biometria.dispositivoSuportaBiometria();
    if (mounted) {
      setState(() => _suportaBiometria = suporta);
    }
  }

  Future<void> _testarAutenticacao() async {
    setState(() => _mensagemStatus = null);
    final sucesso = await widget.biometria.autenticarLocalmente(
      motivo: 'Confirme sua biometria para desbloquear o aplicativo SysAP localmente.',
    );

    if (mounted) {
      setState(() {
        _autenticado = sucesso;
        _mensagemStatus = sucesso
            ? 'Biometria validada com sucesso!'
            : 'Autenticação biométrica não concluída ou recusada.';
      });
    }
  }

  @override
  Widget build(BuildContext context) {
    return Card(
      color: TemaSysAP.corSuperficieEscura,
      child: Padding(
        padding: const EdgeInsets.all(16),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Row(
              children: [
                Icon(
                  Icons.fingerprint,
                  color: _suportaBiometria
                      ? TemaSysAP.corDouradoPrincipal
                      : TemaSysAP.corTextoSecundario,
                  size: 26,
                ),
                const SizedBox(width: 10),
                const Text(
                  'Desbloqueio Biométrico Local',
                  style: TextStyle(
                    color: Colors.white,
                    fontWeight: FontWeight.bold,
                    fontSize: 14,
                  ),
                ),
              ],
            ),
            const SizedBox(height: 8),
            const Text(
              'A biometria atua exclusivamente no desbloqueio rápido do aplicativo já logado, sem substituir sua senha na API ou criar tokens.',
              style: TextStyle(
                color: TemaSysAP.corTextoSecundario,
                fontSize: 12,
                height: 1.3,
              ),
            ),
            const SizedBox(height: 12),
            if (!_suportaBiometria)
              const Text(
                'Biometria não configurada ou não suportada neste aparelho.',
                style: TextStyle(color: TemaSysAP.corAlerta, fontSize: 12),
              )
            else ...[
              ElevatedButton.icon(
                onPressed: _testarAutenticacao,
                icon: const Icon(Icons.lock_open, size: 18),
                label: const Text('Testar Desbloqueio Biométrico'),
                style: ElevatedButton.styleFrom(
                  backgroundColor: TemaSysAP.corFundoObsidiana,
                  foregroundColor: TemaSysAP.corDouradoInteracao,
                  side: const BorderSide(color: TemaSysAP.corBordaTatica),
                ),
              ),
              if (_mensagemStatus != null) ...[
                const SizedBox(height: 8),
                Text(
                  _mensagemStatus!,
                  style: TextStyle(
                    color: _autenticado
                        ? TemaSysAP.corSucesso
                        : TemaSysAP.corAlerta,
                    fontSize: 12,
                    fontWeight: FontWeight.w500,
                  ),
                ),
              ],
            ],
          ],
        ),
      ),
    );
  }
}
