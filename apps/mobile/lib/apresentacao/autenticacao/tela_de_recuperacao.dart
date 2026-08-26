import 'package:flutter/material.dart';
import 'package:flutter/services.dart';

import '../../aplicacao/autenticacao/recuperacao/servico_de_recuperacao.dart';
import '../../dominio/falhas/falha_de_autenticacao.dart';
import '../../infraestrutura/api/cliente_da_api.dart';
import '../tema/tema_sysap.dart';

/// Tela de recuperação de senha por matrícula e OTP com proteção antienumeração.
class TelaDeRecuperacao extends StatefulWidget {
  final ClienteDaApi clienteApi;

  const TelaDeRecuperacao({super.key, required this.clienteApi});

  @override
  State<TelaDeRecuperacao> createState() => _TelaDeRecuperacaoState();
}

class _TelaDeRecuperacaoState extends State<TelaDeRecuperacao> {
  late final ServicoDeRecuperacao _servico;

  int _etapa = 1;
  bool _processando = false;
  String? _mensagemErro;
  String? _mensagemSucesso;

  final _matriculaController = TextEditingController();
  final _codigoController = TextEditingController();
  final _senhaController = TextEditingController();
  String? _recoveryProof;

  @override
  void initState() {
    super.initState();
    _servico = ServicoDeRecuperacao(clienteApi: widget.clienteApi);
  }

  @override
  void dispose() {
    _matriculaController.dispose();
    _codigoController.dispose();
    _senhaController.dispose();
    _recoveryProof = null;
    super.dispose();
  }

  Future<void> _solicitarCodigo() async {
    setState(() {
      _processando = true;
      _mensagemErro = null;
    });

    try {
      await _servico.solicitarRecuperacao(_matriculaController.text);
      setState(() {
        _etapa = 2;
        _processando = false;
      });
    } on FalhaDeAutenticacao catch (e) {
      setState(() {
        _mensagemErro = e.mensagem;
        _processando = false;
      });
    } catch (_) {
      setState(() {
        _mensagemErro = 'Não foi possível solicitar o código de recuperação.';
        _processando = false;
      });
    }
  }

  Future<void> _verificarCodigo() async {
    setState(() {
      _processando = true;
      _mensagemErro = null;
    });

    try {
      final proof = await _servico.verificarCodigo(
        matricula: _matriculaController.text,
        codigo: _codigoController.text,
      );
      setState(() {
        _recoveryProof = proof;
        _etapa = 3;
        _processando = false;
      });
    } on FalhaDeAutenticacao catch (e) {
      setState(() {
        _mensagemErro = e.mensagem;
        _processando = false;
      });
    } catch (_) {
      setState(() {
        _mensagemErro = 'Código de verificação inválido.';
        _processando = false;
      });
    }
  }

  Future<void> _concluir() async {
    if (_recoveryProof == null) {
      setState(
        () => _mensagemErro =
            'Fluxo de recuperação expirado. Reinicie o processo.',
      );
      return;
    }

    setState(() {
      _processando = true;
      _mensagemErro = null;
    });

    try {
      await _servico.concluirRecuperacao(
        proof: _recoveryProof!,
        novaSenha: _senhaController.text,
      );
      setState(() {
        _mensagemSucesso = 'Senha redefinida com sucesso! Você já pode entrar com sua nova senha.';
        _processando = false;
      });
    } on FalhaDeAutenticacao catch (e) {
      setState(() {
        _mensagemErro = e.mensagem;
        _processando = false;
      });
    } catch (_) {
      setState(() {
        _mensagemErro = 'Não foi possível redefinir a senha.';
        _processando = false;
      });
    }
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      backgroundColor: TemaSysAP.corFundoObsidiana,
      appBar: AppBar(
        title: const Text('Recuperação de Acesso'),
        leading: IconButton(
          icon: const Icon(Icons.arrow_back, color: Colors.white),
          onPressed: () => Navigator.of(context).pop(),
        ),
      ),
      body: SafeArea(
        child: SingleChildScrollView(
          padding: const EdgeInsets.symmetric(horizontal: 24, vertical: 24),
          child: ConstrainedBox(
            constraints: const BoxConstraints(maxWidth: 400),
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.stretch,
              children: [
                Text(
                  'Etapa $_etapa de 3',
                  style: const TextStyle(
                    color: TemaSysAP.corDouradoPrincipal,
                    fontWeight: FontWeight.bold,
                    fontSize: 14,
                  ),
                ),
                const SizedBox(height: 8),

                if (_mensagemErro != null) ...[
                  Container(
                    padding: const EdgeInsets.all(12),
                    decoration: BoxDecoration(
                      color: TemaSysAP.corErro.withValues(alpha: 0.15),
                      borderRadius: BorderRadius.circular(8),
                      border: Border.all(
                        color: TemaSysAP.corErro.withValues(alpha: 0.4),
                      ),
                    ),
                    child: Text(
                      _mensagemErro!,
                      style: const TextStyle(color: Colors.white, fontSize: 13),
                    ),
                  ),
                  const SizedBox(height: 16),
                ],

                if (_mensagemSucesso != null) ...[
                  Container(
                    padding: const EdgeInsets.all(12),
                    decoration: BoxDecoration(
                      color: TemaSysAP.corSucesso.withValues(alpha: 0.15),
                      borderRadius: BorderRadius.circular(8),
                      border: Border.all(
                        color: TemaSysAP.corSucesso.withValues(alpha: 0.4),
                      ),
                    ),
                    child: Text(
                      _mensagemSucesso!,
                      style: const TextStyle(color: Colors.white, fontSize: 13),
                    ),
                  ),
                  const SizedBox(height: 24),
                  ElevatedButton(
                    onPressed: () => Navigator.of(context).pop(),
                    style: ElevatedButton.styleFrom(
                      backgroundColor: TemaSysAP.corDouradoPrincipal,
                      foregroundColor: Colors.black,
                    ),
                    child: const Text('Ir para o Login'),
                  ),
                ] else if (_etapa == 1) ...[
                  const Text(
                    'Informe sua matrícula de 10 dígitos cadastrada.',
                    style: TextStyle(
                      color: TemaSysAP.corTextoSecundario,
                      fontSize: 14,
                    ),
                  ),
                  const SizedBox(height: 20),
                  TextFormField(
                    controller: _matriculaController,
                    keyboardType: TextInputType.number,
                    inputFormatters: [
                      FilteringTextInputFormatter.digitsOnly,
                      LengthLimitingTextInputFormatter(10),
                    ],
                    style: const TextStyle(color: Colors.white),
                    decoration: InputDecoration(
                      labelText: 'Matrícula',
                      hintText: 'Ex: 2026000001',
                      filled: true,
                      fillColor: TemaSysAP.corSuperficieEscura,
                      border: OutlineInputBorder(
                        borderRadius: BorderRadius.circular(8),
                      ),
                    ),
                  ),
                  const SizedBox(height: 24),
                  ElevatedButton(
                    onPressed: _processando ? null : _solicitarCodigo,
                    style: ElevatedButton.styleFrom(
                      backgroundColor: TemaSysAP.corDouradoPrincipal,
                      foregroundColor: Colors.black,
                    ),
                    child: _processando
                        ? const SizedBox(
                            height: 18,
                            width: 18,
                            child: CircularProgressIndicator(
                              strokeWidth: 2,
                              color: Colors.black,
                            ),
                          )
                        : const Text('Enviar Código'),
                  ),
                ] else if (_etapa == 2) ...[
                  const Text(
                    'Digite o código OTP de 6 dígitos recebido.',
                    style: TextStyle(
                      color: TemaSysAP.corTextoSecundario,
                      fontSize: 14,
                    ),
                  ),
                  const SizedBox(height: 20),
                  TextFormField(
                    controller: _codigoController,
                    keyboardType: TextInputType.number,
                    inputFormatters: [
                      FilteringTextInputFormatter.digitsOnly,
                      LengthLimitingTextInputFormatter(6),
                    ],
                    style: const TextStyle(color: Colors.white),
                    decoration: InputDecoration(
                      labelText: 'Código OTP (6 dígitos)',
                      filled: true,
                      fillColor: TemaSysAP.corSuperficieEscura,
                      border: OutlineInputBorder(
                        borderRadius: BorderRadius.circular(8),
                      ),
                    ),
                  ),
                  const SizedBox(height: 24),
                  ElevatedButton(
                    onPressed: _processando ? null : _verificarCodigo,
                    style: ElevatedButton.styleFrom(
                      backgroundColor: TemaSysAP.corDouradoPrincipal,
                      foregroundColor: Colors.black,
                    ),
                    child: _processando
                        ? const SizedBox(
                            height: 18,
                            width: 18,
                            child: CircularProgressIndicator(
                              strokeWidth: 2,
                              color: Colors.black,
                            ),
                          )
                        : const Text('Validar Código'),
                  ),
                ] else if (_etapa == 3) ...[
                  const Text(
                    'Digite sua nova senha (mínimo de 15 caracteres).',
                    style: TextStyle(
                      color: TemaSysAP.corTextoSecundario,
                      fontSize: 14,
                    ),
                  ),
                  const SizedBox(height: 20),
                  TextFormField(
                    controller: _senhaController,
                    obscureText: true,
                    style: const TextStyle(color: Colors.white),
                    decoration: InputDecoration(
                      labelText: 'Nova Senha',
                      filled: true,
                      fillColor: TemaSysAP.corSuperficieEscura,
                      border: OutlineInputBorder(
                        borderRadius: BorderRadius.circular(8),
                      ),
                    ),
                  ),
                  const SizedBox(height: 24),
                  ElevatedButton(
                    onPressed: _processando ? null : _concluir,
                    style: ElevatedButton.styleFrom(
                      backgroundColor: TemaSysAP.corDouradoPrincipal,
                      foregroundColor: Colors.black,
                    ),
                    child: _processando
                        ? const SizedBox(
                            height: 18,
                            width: 18,
                            child: CircularProgressIndicator(
                              strokeWidth: 2,
                              color: Colors.black,
                            ),
                          )
                        : const Text('Redefinir Senha'),
                  ),
                ],
              ],
            ),
          ),
        ),
      ),
    );
  }
}
