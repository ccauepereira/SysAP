# Context Pack de agentes — SysAP

## Propósito

Este arquivo é a orientação mínima para uma tarefa assistida por agente. Ele
não substitui a leitura do prompt, do diff atual e das fontes de verdade
afetadas.

## Produto e limites atuais

- SysAP acompanha a rotina esportiva do Artur Performance.
- A API Go é a autoridade de negócio e autorização; clientes não acessam
  tabelas de negócio ou Supabase Auth diretamente.
- Web usa Next.js como BFF. Mobile é Flutter/Dart e fala somente com a API Go.
- Não antecipar dashboard novo, GPS, saúde, smartwatch, colete ou financeiro
  sem uma tarefa explícita, contrato e dados de origem quando necessários.
- Dados de saúde e localização são sensíveis; não podem ir para logs, erros ou
  auditoria técnica.

## Fontes de verdade

1. Prompt da tarefa atual.
2. `docs/architecture.md` e ADRs aplicáveis.
3. `docs/fase-3-arquitetura-mobile.md` para trabalho mobile.
4. `contracts/openapi/openapi.yaml` para interfaces HTTP.
5. `infra/supabase/migrations/` para estado persistido e RLS.
6. Código e testes próximos ao comportamento alterado.

Quando houver conflito material entre essas fontes, pare e peça decisão; não
altere contrato, segurança ou arquitetura silenciosamente.

## Invariantes de segurança

- Não ler, imprimir, copiar, versionar ou modificar `.env.local`.
- Não registrar senha, token, refresh token, OTP, proof, segredo TOTP, e-mail,
  telefone, matrícula, IP bruto, DSN ou erro bruto de dependência.
- Tokens são opacos no cliente. Mobile usa armazenamento seguro de plataforma;
  Web usa BFF/cookies HttpOnly; nenhum usa `localStorage` para tokens.
- JWT válido não substitui validação local de perfil, organização, membership e
  sessão. RLS não usa políticas permissivas universais.
- Serviços externos ficam atrás de adaptadores testáveis. CI/testes usam apenas
  dados fictícios e não enviam mensagens reais.

## Comandos existentes

- Qualidade geral: `pnpm check`
- API/Web: `pnpm check:api`, `pnpm check:web`
- Contrato: `pnpm openapi:lint`
- Segurança: `pnpm security:dependencies`, `pnpm security:secrets`
- Integração local: `pnpm test:integration`
- Mobile, quando aplicável: `flutter analyze`, `flutter test` em `apps/mobile`

Use somente o subconjunto proporcional à alteração e informe resultados novos,
inclusive limitações de ambiente.

## Entrega esperada

Entregue alterações pequenas e coesas, testes proporcionais e um resumo com
arquivos alterados, decisões, comandos/resultados, limitações e estado Git.
Não faça commit, push, deploy, mudança remota ou chamada externa sem pedido
explícito.
