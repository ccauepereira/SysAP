import 'package:flutter_test/flutter_test.dart';
import 'package:sysap/aplicacao/autenticacao/ativacao/servico_de_ativacao.dart';
import 'package:sysap/aplicacao/autenticacao/recuperacao/servico_de_recuperacao.dart';
import 'package:sysap/aplicacao/configuracao/ambiente.dart';
import 'package:sysap/aplicacao/configuracao/configuracao_publica.dart';
import 'package:sysap/dominio/falhas/falha_de_autenticacao.dart';
import 'package:sysap/infraestrutura/api/cliente_da_api.dart';

class MockApiAuthServices extends ClienteDaApi {
  String? ultimoCaminho;
  Map<String, dynamic>? ultimoCorpo;

  MockApiAuthServices()
    : super(
        configuracao: ConfiguracaoPublica(
          ambiente: AmbienteDeExecucao.desenvolvimento,
          urlDaApi: Uri.parse('http://127.0.0.1:8080'),
        ),
      );

  @override
  Future<RespostaHttp> post(
    String caminho, {
    Map<String, dynamic>? corpo,
    String? tokenBearer,
  }) async {
    ultimoCaminho = caminho;
    ultimoCorpo = corpo;

    if (caminho == '/v1/activation/verify') {
      return const RespostaHttp(
        status: 200,
        corpo:
            '{"activation_proof":"proof_hex_99999999999999999999999999999999"}',
      );
    }

    if (caminho == '/v1/auth/password-recovery/verify') {
      return const RespostaHttp(
        status: 200,
        corpo: '{"proof":"recovery_proof_123456789012345678901234567890"}',
      );
    }

    return const RespostaHttp(status: 200, corpo: '{"status":"accepted"}');
  }
}

void main() {
  group('Aplicação: Serviços de Ativação e Recuperação', () {
    late MockApiAuthServices mockApi;
    late ServicoDeAtivacao servicoAtivacao;
    late ServicoDeRecuperacao servicoRecuperacao;

    setUp(() {
      mockApi = MockApiAuthServices();
      servicoAtivacao = ServicoDeAtivacao(clienteApi: mockApi);
      servicoRecuperacao = ServicoDeRecuperacao(clienteApi: mockApi);
    });

    test('ServicoDeAtivacao executa os 3 passos com validação', () async {
      // 1. Start
      await servicoAtivacao.solicitarDesafio('2026000001');
      expect(mockApi.ultimoCaminho, equals('/v1/activation/start'));
      expect(mockApi.ultimoCorpo?['enrollment_number'], equals('2026000001'));

      // 2. Verify
      final proof = await servicoAtivacao.verificarCodigo(
        matricula: '2026000001',
        codigo: '123456',
      );
      expect(proof, equals('proof_hex_99999999999999999999999999999999'));
      expect(mockApi.ultimoCaminho, equals('/v1/activation/verify'));

      // 3. Complete
      await servicoAtivacao.concluirAtivacao(
        activationProof: proof,
        novaSenha: 'senha_muito_segura_123',
      );
      expect(mockApi.ultimoCaminho, equals('/v1/activation/complete'));
    });

    test('ServicoDeAtivacao rejeita senha curta na conclusão', () async {
      expect(
        () => servicoAtivacao.concluirAtivacao(
          activationProof: 'proof123',
          novaSenha: 'curta',
        ),
        throwsA(isA<FalhaDeAutenticacao>()),
      );
    });

    test('ServicoDeRecuperacao executa os 3 passos antienumeração', () async {
      // 1. Start
      await servicoRecuperacao.solicitarRecuperacao('2026000001');
      expect(mockApi.ultimoCaminho, equals('/v1/auth/password-recovery/start'));

      // 2. Verify
      final proof = await servicoRecuperacao.verificarCodigo(
        matricula: '2026000001',
        codigo: '654321',
      );
      expect(proof, equals('recovery_proof_123456789012345678901234567890'));
      expect(
        mockApi.ultimoCaminho,
        equals('/v1/auth/password-recovery/verify'),
      );

      // 3. Complete
      await servicoRecuperacao.concluirRecuperacao(
        proof: proof,
        novaSenha: 'nova_senha_recuperada_123',
      );
      expect(
        mockApi.ultimoCaminho,
        equals('/v1/auth/password-recovery/complete'),
      );
    });
  });
}
