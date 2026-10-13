# Defesa em profundidade — SysAP

## Princípio

Nenhuma camada isolada autoriza acesso. Cada fronteira confirma somente o que
é sua responsabilidade e falha fechada quando não consegue confirmar estado.

| Camada | Controle | Falha esperada |
| --- | --- | --- |
| Cliente Web/Mobile | Sem acesso direto a Auth/tabelas; tokens opacos | Não concede acesso local por papel/cache |
| BFF e API | Validação de formato, origem, limites, envelope seguro | Rejeita sem vazar detalhe interno |
| Identidade | JWT/JWKS, sessão local, perfil, organização e membership | JWT válido não supera revogação/suspensão |
| Banco | RLS forçada, contexto transacional e privilégios mínimos | Papel cliente não lê nem escreve negócio |
| Segredos | Variáveis locais/CI, scanners e exemplos vazios | Segredo nunca entra em Git ou log |
| Dependências/CI | versões fixadas, auditoria e checks reproduzíveis | Falha antes de merge quando gate obrigatório falha |
| Operação | auditoria segura, request_id, no-store e adaptadores | Dependência indisponível não gera bypass |

## Controles obrigatórios por mudança

- Autenticação: validar assinatura, expiração e sessão local; não confiar só no
  payload do token.
- Autorização: confirmar organização, papel e membership na API/banco; negar
  cross-tenant.
- Dados sensíveis: mascarar/omitir em respostas e nunca incluir em logs,
  auditoria, testes versionados ou documentação.
- Persistência: migrations forward-only, RLS habilitada/forçada, grants mínimos
  e nenhuma policy `USING (true)` ou `WITH CHECK (true)` para negócio.
- Integração: adapters server-side; sem fallback silencioso para mock/local em
  produção.
- Mobile: senha/matrícula não persistem; tokens somente em Keystore/Keychain;
  permissões de saúde são granulares e revogáveis.

## Revisão e resposta

O checklist de PR e o Laya advisory ajudam a encontrar perguntas; não substituem
testes, revisão humana nem os gates obrigatórios. Achado plausível de segredo,
vazamento de autorização ou permissões amplas bloqueia a revisão até análise
humana e correção verificável.
