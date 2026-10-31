# AGENTS.md — SysAP

## Missão

Construir o SysAP em pequenas entregas verificáveis. O sistema deve ser explicável por Cauê, seguro para dados de atletas e honesto sobre o que smartwatch e colete GPS realmente medem.

## Fontes de verdade e hierarquia documental

Leia antes de alterar código, respeitando a seguinte precedência:

1. Decisões atuais de produto formalmente aprovadas.
2. Roadmap canônico vigente do SysAP.
3. `docs/architecture.md` — arquitetura, escopo e decisões aplicáveis.
4. `contracts/openapi/` — contrato HTTP quando existir.
5. `infra/supabase/migrations/` — modelo persistido.
6. `assets/brand/README.md` — identidade visual e regras da marca.
7. ADRs e especificações técnicas históricas (interpretadas no contexto de sua criação).
8. O prompt da tarefa atual — objetivo e limite estrito da entrega.

Se houver conflito entre documentos históricos e as decisões aprovadas de produto, prevalece a decisão atual. Pare e explique o conflito antes de prosseguir; nunca altere uma decisão arquitetural silenciosamente.

## Roadmap Canônico do SysAP

Toda entrega no repositório deve respeitar rigorosamente as fases canônicas do produto:

- **Fase 1 — Fundação, Identidade e Acesso:**
  Infraestrutura básica (Go, PostgreSQL, Supabase local), autenticação, MFA administrativo, RBAC/RLS, matrícula institucional, fluxos de ativação (presencial/e-mail) e recuperação de acesso. Inclui o fluxo mínimo operacional para cadastrar e ativar um atleta.
  *Estado:* Fluxo principal homologado manualmente pelo usuário; a recuperação presencial de senha ainda aguarda homologação manual e os testes de banco exigem ambiente isolado.
- **Fase 2 — Operação da Academia (Próxima Fase):**
  Gestão operacional do dia a dia: atletas, turmas, treinadores, responsáveis legais, horários, sessões de treinamento e controle de presença. Priorizar fluxos administrativos completos e utilizáveis pela interface web.
- **Fase 3 — Gestão Financeira:**
  Mensalidades, pagamentos, cobranças, inadimplência, competências financeiras e relatórios.
- **Fase 4 — Estabilização e Lançamento:**
  Homologação final, correções, testes integrados e de estresse, segurança, documentação e implantação operacional para lançamento.
- **Evolução Posterior (Pós-Núcleo Operacional e Financeiro):**
  Funcionalidades avançadas de desempenho esportivo, GPS, smartwatches, Health Connect, HealthKit e algoritmos especializados de carga. **Estas funcionalidades não são prioridades das Fases 1 a 3 e não devem ser introduzidas sem pedido explícito.**

## Forma de trabalhar

- Comece inspecionando o repositório e descrevendo, em português, o que entendeu.
- Para mudanças complexas, apresente um plano curto antes de editar.
- Implemente estritamente a fase ou o corte vertical solicitado. Não expanda escopo para fases futuras sem autorização explícita.
- Faça alterações pequenas, coesas e fáceis de revisar.
- Não esconda decisões atrás de abstrações genéricas ou código excessivamente sofisticado.
- Explique os arquivos alterados, as decisões tomadas e os comandos de verificação executados.
- Não faça commit, push, deploy, migração remota ou alteração em produção sem pedido explícito.

## Arquitetura obrigatória

- `apps/api`: API Go em monólito modular.
- `apps/web`: painel Next.js/TypeScript do treinador.
- `apps/mobile`: aplicativo Flutter/Dart compartilhado para Android e iOS;
  Kotlin e Swift ficam restritos a adapters que exigem APIs nativas.
- `contracts/openapi`: contrato REST.
- `infra/supabase`: migrations, seeds de desenvolvimento e configuração local.
- Clientes nunca acessam tabelas de negócio diretamente; usam a API.
- Supabase Auth autentica e gerencia senha, OTP, MFA e sessões; a API valida o
  JWT e autoriza pelo estado atual de organização, papel e acesso no banco.
- Regras de domínio ficam na API, não em componentes React nem em handlers HTTP.
- Integrações externas ficam atrás de adaptadores com contratos testáveis.
- A ativação do atleta pode usar código manual emitido pelo administrador ou um
  único canal de contato disponível; MFA do administrador é separado.
- Twilio Verify é o provider inicial preferencial de OTP SMS, somente pelo
  provider `twilio_verify` do Supabase Auth; API, Web e mobile nunca integram
  Twilio diretamente.

## Regras de domínio e dispositivos (Evolução Posterior e Salvaguardas)

- O MVP operacional não inclui rastreamento GPS ao vivo nem integração obrigatória com relógios; o foco atual são as operações da academia e a gestão financeira.
- Quando a evolução esportiva for iniciada:
  - Health Connect e HealthKit dependerão de consentimento e permissões granulares do atleta, ficando atrás de adapters Kotlin e Swift.
  - Não crie aplicativo de relógio no MVP; o mobile lerá dados autorizados recebidos pelo sistema operacional.
  - Não invente formato de CSV ou API de fabricante de colete; exija um arquivo real antes de implementar parsers.
  - Um import deve ser idempotente, auditável e manter as unidades originais e normalizadas.
  - GPS não comprova gol, passe, assistência, toque na bola ou xG.
  - Pontuações e alertas devem ser determinísticos, versionados e explicar os dados usados.
  - Nunca apresente prontidão ou carga como diagnóstico médico ou previsão de lesão.
- Regras de identidade e segurança (permanentes):
  - OTP nunca cria usuário automaticamente: desabilite signup global, email e
    anônimo quando não usados, mantenha `auth.sms.enable_signup = false`, use o equivalente a
    `shouldCreateUser: false` e solicite código somente para identidade
    previamente provisionada.
  - Testes e CI usam somente `auth.sms.test_otp`, telefone reservado fictício,
    provider externo não configurado, credenciais ausentes e nenhuma rede
    externa; testes de carga nunca enviam SMS real.

## Código e qualidade

- Código, identificadores e mensagens técnicas em inglês; interface e documentação de produto em português do Brasil.
- Use UTC no banco e ISO 8601 na API; converta para `America/Fortaleza` na
  apresentação e em regra civil explicitamente documentada, como o ano da
  matrícula, nunca por fuso implícito do host.
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
- Mobile, quando criado: análise/testes Flutter e testes dos adapters nativos.
- Contrato: validação do OpenAPI.
- Infra: migrations aplicáveis do zero em banco local limpo.

Antes de finalizar, revise o diff e informe qualquer teste que não pôde ser executado.

## Definição de pronto

Uma tarefa só está pronta quando o comportamento solicitado funciona, há teste proporcional ao risco, os comandos relevantes passam, a documentação afetada foi atualizada e nenhuma parte fora do escopo foi implementada.
