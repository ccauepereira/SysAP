import 'package:flutter/material.dart';
import 'package:flutter/services.dart';

import '../../aplicacao/sessao/gerenciador_de_sessao.dart';
import '../tema/tema_sysap.dart';
import 'tela_de_ativacao.dart';
import 'tela_de_recuperacao.dart';

/// Tela de login oficial do SysAP Mobile.
///
/// Implementa a identidade visual Obsidiana + Dourado Artur Performance,
/// acessibilidade com leitores de tela e bloqueio contra envio duplo.
class TelaDeLogin extends StatefulWidget {
  final GerenciadorDeSessao gerenciadorDeSessao;

  const TelaDeLogin({super.key, required this.gerenciadorDeSessao});

  @override
  State<TelaDeLogin> createState() => _TelaDeLoginState();
}

class _TelaDeLoginState extends State<TelaDeLogin> {
  final _formKey = GlobalKey<FormState>();
  final _matriculaController = TextEditingController();
  final _senhaController = TextEditingController();
  bool _ocultarSenha = true;
  String? _mensagemErroLocal;

  @override
  void dispose() {
    _matriculaController.dispose();
    _senhaController.dispose();
    super.dispose();
  }

  Future<void> _submeter() async {
    setState(() => _mensagemErroLocal = null);

    if (!_formKey.currentState!.validate()) {
      return;
    }

    final matricula = _matriculaController.text.trim();
    final senha = _senhaController.text;

    await widget.gerenciadorDeSessao.login(matricula: matricula, senha: senha);
  }

  @override
  Widget build(BuildContext context) {
    final estadoSessao = widget.gerenciadorDeSessao.value;
    final autenticando = estadoSessao is SessaoAutenticando;

    String? mensagemErro;
    if (_mensagemErroLocal != null) {
      mensagemErro = _mensagemErroLocal;
    } else if (estadoSessao is SessaoFalha) {
      mensagemErro = estadoSessao.falha.mensagem;
    } else if (estadoSessao is SessaoAcessoNegado) {
      mensagemErro = estadoSessao.motivo;
    }

    return Scaffold(
      backgroundColor: TemaSysAP.corFundoObsidiana,
      body: SafeArea(
        child: Center(
          child: SingleChildScrollView(
            padding: const EdgeInsets.symmetric(horizontal: 24, vertical: 32),
            child: ConstrainedBox(
              constraints: const BoxConstraints(maxWidth: 400),
              child: Form(
                key: _formKey,
                child: Column(
                  mainAxisAlignment: MainAxisAlignment.center,
                  crossAxisAlignment: CrossAxisAlignment.stretch,
                  children: [
                    // Logo Canônica Artur Performance
                    Semantics(
                      label: 'Logo oficial Artur Performance',
                      child: Image.asset(
                        'assets/brand/artur-performance-logo.png',
                        height: 72,
                        fit: BoxFit.contain,
                        errorBuilder: (context, error, stackTrace) {
                          return const Text(
                            'SysAP — Artur Performance',
                            textAlign: TextAlign.center,
                            style: TextStyle(
                              fontSize: 22,
                              fontWeight: FontWeight.bold,
                              color: TemaSysAP.corDouradoPrincipal,
                            ),
                          );
                        },
                      ),
                    ),
                    const SizedBox(height: 12),
                    const Text(
                      'Laboratório de Performance Esportiva',
                      textAlign: TextAlign.center,
                      style: TextStyle(
                        fontSize: 14,
                        color: TemaSysAP.corTextoSecundario,
                      ),
                    ),
                    const SizedBox(height: 36),

                    // Mensagem de Erro Segura
                    if (mensagemErro != null) ...[
                      Semantics(
                        liveRegion: true,
                        child: Container(
                          padding: const EdgeInsets.all(12),
                          decoration: BoxDecoration(
                            color: TemaSysAP.corErro.withValues(alpha: 0.15),
                            borderRadius: BorderRadius.circular(8),
                            border: Border.all(
                              color: TemaSysAP.corErro.withValues(alpha: 0.4),
                            ),
                          ),
                          child: Row(
                            children: [
                              const Icon(
                                Icons.error_outline,
                                color: TemaSysAP.corErro,
                                size: 20,
                              ),
                              const SizedBox(width: 10),
                              Expanded(
                                child: Text(
                                  mensagemErro,
                                  style: const TextStyle(
                                    color: Colors.white,
                                    fontSize: 13,
                                  ),
                                ),
                              ),
                            ],
                          ),
                        ),
                      ),
                      const SizedBox(height: 20),
                    ],

                    // Campo de Matrícula
                    Semantics(
                      label: 'Campo de Matrícula (10 dígitos numéricos)',
                      child: TextFormField(
                        controller: _matriculaController,
                        keyboardType: TextInputType.number,
                        inputFormatters: [
                          FilteringTextInputFormatter.digitsOnly,
                          LengthLimitingTextInputFormatter(10),
                        ],
                        enabled: !autenticando,
                        style: const TextStyle(color: Colors.white),
                        decoration: InputDecoration(
                          labelText: 'Matrícula',
                          labelStyle: const TextStyle(
                            color: TemaSysAP.corTextoSecundario,
                          ),
                          hintText: 'Ex: 2026000001',
                          hintStyle: TextStyle(
                            color: TemaSysAP.corTextoSecundario.withValues(
                              alpha: 0.5,
                            ),
                          ),
                          prefixIcon: const Icon(
                            Icons.badge_outlined,
                            color: TemaSysAP.corDouradoPrincipal,
                          ),
                          filled: true,
                          fillColor: TemaSysAP.corSuperficieEscura,
                          border: OutlineInputBorder(
                            borderRadius: BorderRadius.circular(8),
                            borderSide: const BorderSide(
                              color: TemaSysAP.corBordaTatica,
                            ),
                          ),
                          enabledBorder: OutlineInputBorder(
                            borderRadius: BorderRadius.circular(8),
                            borderSide: const BorderSide(
                              color: TemaSysAP.corBordaTatica,
                            ),
                          ),
                          focusedBorder: OutlineInputBorder(
                            borderRadius: BorderRadius.circular(8),
                            borderSide: const BorderSide(
                              color: TemaSysAP.corDouradoPrincipal,
                              width: 1.5,
                            ),
                          ),
                        ),
                        validator: (valor) {
                          if (valor == null || valor.trim().isEmpty) {
                            return 'Informe sua matrícula.';
                          }
                          if (valor.trim().length != 10) {
                            return 'A matrícula deve ter exatamente 10 dígitos.';
                          }
                          return null;
                        },
                      ),
                    ),
                    const SizedBox(height: 16),

                    // Campo de Senha
                    Semantics(
                      label: 'Campo de Senha',
                      child: TextFormField(
                        controller: _senhaController,
                        obscureText: _ocultarSenha,
                        enabled: !autenticando,
                        style: const TextStyle(color: Colors.white),
                        decoration: InputDecoration(
                          labelText: 'Senha',
                          labelStyle: const TextStyle(
                            color: TemaSysAP.corTextoSecundario,
                          ),
                          prefixIcon: const Icon(
                            Icons.lock_outline,
                            color: TemaSysAP.corDouradoPrincipal,
                          ),
                          suffixIcon: IconButton(
                            icon: Icon(
                              _ocultarSenha
                                  ? Icons.visibility_outlined
                                  : Icons.visibility_off_outlined,
                              color: TemaSysAP.corTextoSecundario,
                            ),
                            onPressed: () {
                              setState(() => _ocultarSenha = !_ocultarSenha);
                            },
                          ),
                          filled: true,
                          fillColor: TemaSysAP.corSuperficieEscura,
                          border: OutlineInputBorder(
                            borderRadius: BorderRadius.circular(8),
                            borderSide: const BorderSide(
                              color: TemaSysAP.corBordaTatica,
                            ),
                          ),
                          enabledBorder: OutlineInputBorder(
                            borderRadius: BorderRadius.circular(8),
                            borderSide: const BorderSide(
                              color: TemaSysAP.corBordaTatica,
                            ),
                          ),
                          focusedBorder: OutlineInputBorder(
                            borderRadius: BorderRadius.circular(8),
                            borderSide: const BorderSide(
                              color: TemaSysAP.corDouradoPrincipal,
                              width: 1.5,
                            ),
                          ),
                        ),
                        validator: (valor) {
                          if (valor == null || valor.isEmpty) {
                            return 'Informe sua senha.';
                          }
                          return null;
                        },
                      ),
                    ),
                    const SizedBox(height: 24),

                    // Botão Entrar
                    SizedBox(
                      height: 48,
                      child: ElevatedButton(
                        onPressed: autenticando ? null : _submeter,
                        style: ElevatedButton.styleFrom(
                          backgroundColor: TemaSysAP.corDouradoPrincipal,
                          foregroundColor: Colors.black,
                          disabledBackgroundColor: TemaSysAP.corDouradoPrincipal
                              .withValues(alpha: 0.5),
                          shape: RoundedRectangleBorder(
                            borderRadius: BorderRadius.circular(8),
                          ),
                          textStyle: const TextStyle(
                            fontSize: 16,
                            fontWeight: FontWeight.bold,
                          ),
                        ),
                        child: autenticando
                            ? const SizedBox(
                                height: 20,
                                width: 20,
                                child: CircularProgressIndicator(
                                  strokeWidth: 2.5,
                                  color: Colors.black,
                                ),
                              )
                            : const Text('Entrar'),
                      ),
                    ),
                    const SizedBox(height: 24),

                    // Ações Secundárias (Ativação e Recuperação)
                    Wrap(
                      alignment: WrapAlignment.spaceBetween,
                      crossAxisAlignment: WrapCrossAlignment.center,
                      children: [
                        TextButton(
                          onPressed: autenticando
                              ? null
                              : () {
                                  Navigator.of(context).push(
                                    MaterialPageRoute(
                                      builder: (_) => TelaDeAtivacao(
                                        clienteApi: widget
                                            .gerenciadorDeSessao
                                            .clienteApi,
                                      ),
                                    ),
                                  );
                                },
                          child: const Text(
                            'Ativar conta',
                            style: TextStyle(
                              color: TemaSysAP.corDouradoInteracao,
                              fontSize: 13,
                            ),
                          ),
                        ),
                        TextButton(
                          onPressed: autenticando
                              ? null
                              : () {
                                  Navigator.of(context).push(
                                    MaterialPageRoute(
                                      builder: (_) => TelaDeRecuperacao(
                                        clienteApi: widget
                                            .gerenciadorDeSessao
                                            .clienteApi,
                                      ),
                                    ),
                                  );
                                },
                          child: const Text(
                            'Esqueci a senha',
                            style: TextStyle(
                              color: TemaSysAP.corTextoSecundario,
                              fontSize: 13,
                            ),
                          ),
                        ),
                      ],
                    ),
                  ],
                ),
              ),
            ),
          ),
        ),
      ),
    );
  }
}
