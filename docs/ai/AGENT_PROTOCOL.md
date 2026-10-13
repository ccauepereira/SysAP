# Protocolo de agentes — SysAP

## Antes de editar

1. Confirmar branch e `git status --short --branch`.
2. Ler o prompt e as fontes de verdade atingidas.
3. Inspecionar arquivos próximos, scripts e testes existentes.
4. Declarar o resultado observável, riscos e plano curto quando a tarefa for
   complexa.
5. Preservar alterações de terceiros; se o worktree não estiver limpo, isolar
   a própria alteração ou pedir orientação.

## Durante a execução

- Implementar o menor corte que atende o pedido.
- Não expandir escopo para feature posterior nem reformatar arquivos sem
  relação.
- Não abrir `.env.local`; tratar segredos e PII como dados proibidos em saída,
  logs, fixtures e documentação.
- Não executar commit, push, deploy, login/link remoto ou serviço externo sem
  autorização explícita.
- Usar adaptadores para qualquer fronteira externa e fakes somente em testes.
- Para banco, preservar migrations forward-only, RLS e privilégios mínimos.

## Revisão de segurança

Antes de concluir, conferir: autenticação versus autorização local, isolamento
por organização, validação de entrada, rate limiting quando exposto, cache de
respostas sensíveis, logs sem segredos/PII e ausência de permissões amplas.

## Evidência

Executar os gates existentes relevantes, do mais específico ao mais amplo.
Classificar cada requisito como confirmado, parcial, não verificado ou falho.
Falha de ambiente deve incluir comando, causa resumida e efeito na evidência;
nunca ser apresentada como aprovação.

## Laya PR gate

`tools/laya_pr_gate/review_diff.py` é uma revisão experimental e advisory. Ele
gera perguntas e sinais a partir do diff local, não acessa rede, não interpreta
segredos e sempre sai com status zero depois de gerar um relatório válido. A
decisão de merge continua sendo humana e os checks obrigatórios continuam
independentes.
