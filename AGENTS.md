# AGENTS.md — SysAP

## Missão

Construir o SysAP em pequenas entregas verificáveis. O sistema deve ser explicável por Cauê, seguro para dados de atletas e honesto sobre o que smartwatch e colete GPS realmente medem.

## Fontes de verdade

Leia antes de alterar código:

1. `docs/architecture.md` — arquitetura, escopo e prioridades.
2. `contracts/openapi/` — contrato HTTP quando existir.
3. `infra/supabase/migrations/` — modelo persistido.
4. `assets/brand/README.md` — identidade visual e regras da marca.
5. O prompt da tarefa atual — objetivo e limite da entrega.

Se houver conflito, pare e explique o conflito. Não mude uma decisão arquitetural silenciosamente.

## Forma de trabalhar

- Comece inspecionando o repositório e descrevendo, em português, o que entendeu.
- Para mudanças complexas, apresente um plano curto antes de editar.
- Implemente somente a fase ou o corte vertical solicitado.
- Faça alterações pequenas, coesas e fáceis de revisar.
- Não esconda decisões atrás de abstrações genéricas ou código excessivamente sofisticado.
- Explique os arquivos alterados, as decisões tomadas e os comandos de verificação executados.
- Não faça commit, push, deploy, migração remota ou alteração em produção sem pedido explícito.

## Arquitetura obrigatória

- `apps/api`: API Go em monólito modular.
- `apps/web`: painel Next.js/TypeScript do treinador.
- `apps/android`: aplicativo Kotlin/Jetpack Compose do atleta.
- `contracts/openapi`: contrato REST.
- `infra/supabase`: migrations, seeds de desenvolvimento e configuração local.
- Clientes nunca acessam tabelas de negócio diretamente; usam a API.
- Supabase Auth autentica; a API valida o JWT e autoriza por organização e papel.
- Regras de domínio ficam na API, não em componentes React nem em handlers HTTP.
- Integrações externas ficam atrás de adaptadores com contratos testáveis.

## Regras de domínio e dispositivos

- O MVP processa os dados após o treino; não implemente rastreamento GPS ao vivo.
- Health Connect depende de consentimento e permissões granulares do atleta.
- Não crie aplicativo de relógio no MVP; o app Android lê dados autorizados do Health Connect.
- Não invente formato de CSV ou API de fabricante de colete. Exija um arquivo real antes de implementar um parser específico.
- Um import deve ser idempotente, auditável e manter as unidades originais e normalizadas.
- GPS não comprova gol, passe, assistência, toque na bola ou xG.
- Pontuações e alertas devem ser determinísticos, versionados e explicar os dados usados.
- Nunca apresente prontidão ou carga como diagnóstico médico ou previsão de lesão.

## Código e qualidade

- Código, identificadores e mensagens técnicas em inglês; interface e documentação de produto em português do Brasil.
- Use UTC no banco e ISO 8601 na API; converta para `America/Fortaleza` somente na apresentação.
- Dinheiro, distância, velocidade e duração devem ter unidades explícitas.
- Evite pacote `utils`; nomeie código pela responsabilidade.
- Não adicione dependência sem justificar sua necessidade e verificar manutenção/licença.
- Não redesenhe, recolora, distorça ou aplique efeitos à logo oficial. Use o arquivo canônico em `assets/brand/`.
- Use os tokens da marca e respeite contraste; texto sobre fundo dourado deve ser escuro.
- Nunca coloque chaves, tokens, dados pessoais ou arquivos reais de atletas no Git.
- Seeds devem ser claramente fictícios.
- Toda migration deve ter estratégia de rollback ou explicação de irreversibilidade.

## Verificação mínima

Execute apenas os comandos existentes no repositório. Quando os módulos forem criados, mantenha estes gates:

- API: formatação, `go vet ./...` e `go test ./...`.
- Web: lint, typecheck, testes e build.
- Android: lint e testes unitários Gradle.
- Contrato: validação do OpenAPI.
- Infra: migrations aplicáveis do zero em banco local limpo.

Antes de finalizar, revise o diff e informe qualquer teste que não pôde ser executado.

## Definição de pronto

Uma tarefa só está pronta quando o comportamento solicitado funciona, há teste proporcional ao risco, os comandos relevantes passam, a documentação afetada foi atualizada e nenhuma parte fora do escopo foi implementada.
