# SysAP MVP 1.0 — auditoria da Fase 1

Data: 13/10/2026. Escopo: auditoria estática do backend, Web, Mobile, migrations, contratos e workflows na branch `mobileatleta`. Nenhuma funcionalidade foi implementada. `.env.local` não foi aberto, copiado nem alterado; não houve contato com Supabase remoto ou Brevo real.

## Resumo executivo

A base de identidade é real: a API Go implementa convite, ativação, login, ciclo de sessão, MFA e recuperação, com testes e persistência PostgreSQL. A Web tem BFF same-origin e formulários, mas a ativação mostrada ainda chama a sequência antiga de um único OTP; o envio de matrícula por SMS e a etapa SMS da ativação não são suportados pelo provider ligado hoje. O cadastro também depende de uma listagem de turmas que não existe na API. O Mobile tem telas e fluxo de autenticação, porém o repositório chamado “armazenamento seguro” é um `Map` em memória, não Keychain/Keystore.

Calendário, presença/observação/notas e mensalidades/pagamento não têm modelo, rotas ou telas funcionais no estado auditado. O dashboard Web contém dados demonstrativos identificados como tal. O gate de dependências está falhando por `braces`/GHSA-vfj7-8cjw-p6xm. Portanto a branch não está pronta para declarar o MVP pronto nem para uma PR de produto sem tratar esse bloqueio; status de proteção/required checks do GitHub não foi consultado e não pode ser inferido dos YAMLs.

## Matriz factual

| Fluxo | Status | Backend / dados | Interface | Testes / evidência | Falta |
|---|---|---|---|---|---|
| Cadastro de atleta pelo Artur e responsável | parcial | `POST /v1/organizations/{organization_id}/athlete-invitations`; owner, membership/organização, AAL2, idempotência, vínculo a `training_groups` e envio são verificados no handler. A API exige nome, celular e e-mail, mas não há campos/tabela de responsável nem validação de regra de menor de 16 anos. O envio ocorre antes do commit; em falha, o rollback remove atleta, convite e chave idempotente. | `/admin/atletas/novo` e formulário existem. A página consulta `/v1/organizations/{id}/teams`, endpoint ausente no router/OpenAPI; canal está efetivamente fixo em e-mail e SMS desabilitado. Trainer pode ver a página, mas o POST BFF/backend exige owner. Falha mostra erro neutro, porém não há convite persistido nem ação de reenvio. | `apps/api/internal/identity/invitation_integration_test.go` cobre owner, papéis, organização e idempotência com fixtures, mas não comprova reenvio após falha do provider. Seed versionado é intencionalmente vazio. | Decidir modelo e consentimento de responsáveis; listar/criar turmas; alinhar papéis; suportar os canais declarados; persistir um estado seguro de entrega e fluxo explícito de retry sem retry cego; separar envio externo da transação. |
| Ativação por e-mail (e confirmação dupla definida para o produto) | parcial | Contrato/handlers contêm start, verify, verify-sms, email/start, email/verify e complete; challenges/proofs são persistidos como HMAC. Provider configurado é `EmailOTPProvider`: `StartChannel` aceita `sms`, mas envia a destination por Brevo como e-mail. Não há provider SMS configurado. | Formulário atual (`activation-forms.tsx`) chama start/verify/complete antigos; não percorre as rotas SMS→e-mail. Rotas BFF para as etapas novas existem, mas não estão conectadas ao formulário. | `activation_integration_test.go` testa ativação local com providers/fakes; não comprova entrega SMS real. | Provider SMS server-side; ligar Web ao fluxo de duas etapas; testes end-to-end com fakes de cada canal e concorrência. |
| Login, sessão e logout | confirmado na API; Web parcial; Mobile parcial | `/v1/auth/login`, refresh, logout, logout-all e `/v1/me`; contexto exige sessão/membership locais. Testes de login cobrem athlete/trainer/owner e criação de sessão; há testes de ciclo de sessão. | Web usa BFF e valida sessão no servidor; rotas `/login` e `/atleta` existem. Dashboard do atleta ainda é explicitamente demonstrativo. Mobile tem cliente HTTP para a API e navegação por papel, mas `ArmazenamentoSeguroDeSessao` usa `Map<String,String>` em memória, não armazenamento seguro persistente do sistema. | `login_integration_test.go`, `mfa_recovery_integration_test.go`, `apps/mobile/integration_test/fluxo_de_autenticacao_integration_test.dart` (este último usa cliente controlado/fake). | Implementar armazenamento Keychain/Keystore real antes de tratar sessão Mobile como pronta; conectar telas a dados autorizados reais; testar expiração/revogação end-to-end em Web e Mobile. |
| Calendário / datas de treino | ausente | Não foram encontrados recursos, tabelas, handlers ou operações OpenAPI de treino/calendário. | `next-training.tsx` exibe cartão demonstrativo e ação de calendário desabilitada; sem rota funcional de calendário. | Sem testes de domínio/API para agenda. | Definir entidade, regras de recorrência/fuso/cancelamento e API antes da tela. |
| Presença/falta, nota e observação | ausente | Sem modelo, migration, endpoint ou serviço correspondente. | Sem fluxo funcional encontrado. | Sem testes de domínio/API. | Definir quem registra, quem pode ver/editar e histórico/auditoria; depois construir persistência/API e interface. |
| Mensalidade, comprovante WhatsApp, confirmação e tolerância | ausente | Sem entidades/rotas de cobrança, vencimento ou confirmação manual. | Sem fluxo funcional de mensalidade/WhatsApp. | Sem testes de domínio/API. | Artur precisa decidir valor, vencimento, tolerância de três dias, acordos e trilha de confirmação; só então modelar. |

## Gaps e riscos priorizados

| Risco / evidência | Severidade | Confiança |
|---|---|---|
| Ativação SMS não pode entregar OTP: `EmailOTPProvider.StartChannel` aceita `sms`, porém chama `BrevoEmailProvider.Send` com o destino recebido; o provider de matrícula explicitamente rejeita canal diferente de `email` (`otp_provider.go`, `enrollment_delivery.go`). | Alta — bloqueia o fluxo declarado | Alta |
| E-mail de matrícula é enviado dentro da transação PostgreSQL, antes dos inserts de idempotência/auditoria e do commit (`invitation_handler.go:303-341`). Um erro posterior pode deixar mensagem entregue para convite revertido; a transação também fica aberta durante I/O externo. | Média — consistência/recuperação | Alta |
| Cadastro não representa responsável nem aplica regra de idade/consentimento de menores, embora seja requisito do MVP. | Alta — produto e proteção de menores | Alta |
| UI depende de endpoint de turmas inexistente e admite trainer na página enquanto o POST só permite owner; turmas seed não existem (seed vazio). | Média — bloqueia cadastro de ponta a ponta | Alta |
| Armazenamento Mobile documentado como Keychain/Keystore é somente mapa em memória (`apps/mobile/lib/infraestrutura/armazenamento_seguro/armazenamento_seguro_de_sessao.dart`). | Alta para lançamento Mobile com sessão | Alta |
| `GET /v1/me` existe no router e handler, mas OpenAPI ainda o marca como `planned` (contrato desatualizado). | Baixa — drift de contrato | Alta |
| `pnpm security:dependencies` falha: `braces` transitivo, instalado `3.0.3`, GHSA-vfj7-8cjw-p6xm, high; advisory aponta faixa vulnerável `<=3.0.3` e corrigida `>=3.0.4`, caminho `eslint-config-next > @next/eslint-plugin-next > fast-glob > micromatch > braces`. | Alta — gate obrigatório falha | Alta |

## CI e rotina de PR

`.github/workflows/pr-checks.yml` roda qualidade (instalação frozen, OpenAPI, `pnpm check`, testes Laya e diff) e Mobile Flutter em jobs sem `continue-on-error`; uma falha deixa o job/workflow vermelho. `security.yml` tem jobs separados para segredos e dependências, sem advisory/`continue-on-error`; a auditoria de dependências falha no estado atual. `ci.yml` cobre API, Web, contrato e integração para PRs direcionadas a `main`; os gates de segurança e checks de PR também são acionados independentemente. Laya é corretamente separada e advisory (`continue-on-error: true`). YAML declara `permissions: contents: read` e checkout sem credenciais persistentes.

Isso comprova comportamento dos workflows, não que uma PR seja realmente impedida de merge: a branch protection/ruleset do GitHub é configuração remota e não foi consultada. O bloqueio de merge por status obrigatório permanece **não verificado**. No estado local, a falha de dependências é um bloqueio objetivo para o job correspondente.

## Checks executados

| Comando | Resultado | Observação |
|---|---|---|
| `git status --short --branch` (preflight) | PASS | `## mobileatleta...origin/mobileatleta`; árvore limpa antes deste relatório. |
| `pnpm install --frozen-lockfile --ignore-scripts` | PASS | Rodado em cópia temporária sem `.env.local`; lockfile congelado. |
| `pnpm openapi:lint` | PASS | Contrato validado. |
| `go mod verify`, `go vet ./...`, `env -u SYSAP_TEST_DATABASE_URL go test -race ./...` | PASS | Testes Go sem banco real; testes que exigem `SYSAP_TEST_DATABASE_URL` são pulados por ausência da variável. |
| Validação YAML de `.github/workflows/*.yml` | PASS | Três workflows parseados. |
| `python3 -m unittest discover -s tools/laya_pr_gate -p 'test_*.py'` | PASS | 4 testes. |
| `python3 tools/laya_pr_gate/review_diff.py` | PASS / advisory | No diff inicial, não havia arquivo para revisar; após criar este relatório, levantou apenas a pergunta de revisão humana `sensitive-data` para o documento novo, sem sinais automáticos. |
| `pnpm security:dependencies` | FAIL | Gate encerra com auditoria recusada; `pnpm audit --json` identificou o advisory de `braces` acima. |
| `pnpm security:secrets` | PASS | Estado e histórico verificados; saída redigida. `.env.local` está ignorado e não rastreado. |
| `pnpm db:status` | PASS | “Supabase local: running.” |
| `pnpm test:integration` | BLOCKED / NOT RUN | O script exige Supabase local inicialmente parado (`requireInactive: true`). Não foi executado para preservar o stack preexistente do usuário. |
| `pnpm check` | PASS | Rodado em cópia temporária limpa, sem `.env.local`: validação de política, API/Go, lint e typecheck Web, 79 testes Web, build Next.js e OpenAPI passaram. |
| `flutter pub get --enforce-lockfile`, `flutter analyze`, `flutter test` | NOT RUN | Flutter CLI não está instalado/disponível neste terminal (`command -v flutter` não encontrou executável). |
| `git diff --check`, `git diff --cached --check` | PASS | Sem whitespace inválido antes de criar este relatório; repetidos ao final. |

## Falha de entrega e ativação assistida

> A análise abaixo registrou o comportamento anterior à migration `20261013164451_phase_2g2_invitation_delivery_state.sql`. O estado atualizado após a correção está descrito ao final desta seção.

Revisão adicional do caminho real (sem chamadas à Brevo):

- O handler exige owner com AAL2 e valida membership/organização. Cria `athlete_profiles` como `pending_activation`, `activation_invitations` como `pending` e vínculo à turma; chama `SendEnrollment` ainda dentro de `WithMembershipContext`, antes do registro de idempotência/auditoria e do commit (`apps/api/internal/identity/invitation_handler.go`).
- Se `SendEnrollment` falha, a API responde `503 service_unavailable` com mensagem neutra. O callback da transação retorna erro; portanto os inserts, inclusive o convite pendente, são revertidos. Nenhum status de conta ativada é retornado. A membership organizacional é criada somente na conclusão da ativação (`activation_handler.go`), não existe membership pendente criada pelo convite.
- A Web também apresenta uma mensagem genérica, sem erro interno. Porém não há registro/ID para o Artur acompanhar nem botão seguro para reenviar. O BFF cria novo UUID `Idempotency-Key` por submissão; retry HTTP não conserva a chave. Se o provider aceitar o envio e ocorrer falha posterior antes do commit, uma nova submissão pode criar outro convite e enviar novamente. Não há hoje garantia de deduplicação de mensagem nem retry controlado.
- O adaptador de matrícula é somente e-mail e retorna `unsupported_delivery_channel` para SMS. `BrevoEmailProvider` recusa destino que não pareça e-mail. Seus testes podem e devem usar servidor HTTP fake; nenhum teste desta revisão chamou o endpoint real.
- A rota `POST /v1/athlete-invitations/{invitation_id}/resend` está no OpenAPI como `planned`, não está registrada no servidor Go e não tem handler/BFF correspondente. As regras de reenvio OTP existentes são para challenges de ativação, não para a entrega inicial de matrícula; não podem ser presumidas como política de reenvio de convite.

**Conclusão:** o comportamento de erro é seguro na resposta, mas não satisfaz o requisito de manter um cadastro pendente recuperável após falha. Para corrigir com segurança será necessário decidir/implementar persistência de estado de entrega, idempotência que sobreviva a retries, operação autorizada de reenvio e apresentação de status. Essa mudança não foi implementada nesta auditoria porque a rota correspondente está explicitamente planejada e as regras de retry da matrícula ainda não estão definidas. Não editar banco nem marcar status manualmente.

A ativação assistida presencial não é suportada pela arquitetura executável atual: não há endpoint owner/AAL2 para emitir desafio QR vinculado ao convite, nem mecanismo de consumo único por esse fluxo. Os proofs atuais pertencem ao fluxo de ativação iniciado pelo próprio atleta/responsável com matrícula e etapas OTP. Não criar QR, compartilhar matrícula/OTP do Artur, aceitar alteração de status ou permitir que o Artur defina/veja a senha. A alternativa exige contrato/API e testes de autorização, tenant, expiração e consumo atômico antes de qualquer interface.

Sem decisões adicionais, próximo corte recomendado: desenhar contrato de entrega/reenvio do convite em torno de estado persistido e operação explicitamente autorizada; manter a ativação assistida fora de escopo até haver mecanismo de desafio no dispositivo do responsável.

### Atualização após persistência de entrega

O código agora grava atleta, convite, profile, membership pendente, idempotência e auditoria antes de chamar o provider. A falha de e-mail não desfaz o convite e não o apresenta como ativado. O estado seguro e separado é `sending`, `sent`, `failed` ou `unknown`; envio aceito pela Brevo significa somente aceitação da requisição, não entrega na caixa postal. Timeout, falha de transporte e resposta ambígua viram `unknown`; nenhum reenvio automático ocorre.

Há endpoints implementados de leitura do estado e reenvio por owner AAL2 com membership ativo no tenant. O reenvio só parte de `failed`, usa lock transacional, chave idempotente própria, cooldown de 60 segundos e máximo de três reenvios em 24 horas; cada tentativa é auditada sem contato ou detalhe do provider. A Web conserva a chave de idempotência da criação após erro de transporte, exibe a conta como pendente e só apresenta reenvio para falha definitiva. Estado `unknown` instrui reconciliação e bloqueia botão de reenvio.

A migration faz backfill de profile/membership para convites pendentes antigos e conserva seu `expires_at`; ativação promove linhas pendentes ou mantém compatibilidade com convites antigos. OpenAPI, handlers e BFF agora expõem leitura/reenvio. Os testes adicionados cobrem classificação de resultado, persistência do pending após falha no teste PostgreSQL e comportamento visual seguro; a cobertura real de RLS e das rotas de reenvio ainda precisa ser executada com a stack parada. O canal real desta integração permanece apenas e-mail; SMS continua desabilitado e não há fallback.

A ativação assistida permanece não implementada: inexiste endpoint de desafio de uso único vinculado ao convite e fluxo de consumo pelo próprio responsável. Não usar QR improvisado, compartilhar matrícula/OTP com o Artur nem editar status diretamente no banco.

### Revisão advisory da Laya para esta atualização

`python3 -m unittest discover -s tools/laya_pr_gate -p 'test_*.py'`: PASS (4 testes). `python3 tools/laya_pr_gate/review_diff.py`: analisou o arquivo novo não rastreado, conforme o README; produziu a pergunta `[sensitive-data]` e nenhum sinal automático. Classificação: **não aplicável como finding / falso positivo se interpretada como vulnerabilidade** — é uma pergunta genérica, e a leitura manual confirmou que o relatório não contém credenciais, contatos, matrícula, tokens, OTP ou provas. Nenhum sinal de autorização/isolamento/ativação foi emitido pelo gate para este diff documental; os achados funcionais acima foram verificados diretamente no handler, router, OpenAPI, migration e testes citados.

Checks pós-Laya desta atualização: `pnpm openapi:lint` PASS; `pnpm check` PASS (cópia temporária sem `.env.local`, incluindo API, lint/typecheck, 79 testes Web, build, política e 41 testes de scripts); `pnpm security:secrets` PASS; `pnpm security:dependencies` FAIL pelo advisory de `braces` já descrito. A integração PostgreSQL não foi executada porque a stack local já estava ativa e o script exige stack parada. Não houve mudança de código/migration; Flutter não se aplica a este diff documental.

## Próximos cortes mínimos

1. Fechar decisões de responsável/menores, consentimento e modelo de turmas; criar endpoint/API real de turmas e alinhar owner/trainer.
2. Tornar convite e ativação realmente executáveis: provider SMS suportado, Web conectada às duas etapas, envio por outbox após commit e retries/idempotência seguros.
3. Corrigir o advisory transitivo `braces` pelo menor upgrade compatível, sem exceção de scanner, e repetir gates de segurança.
4. Definir e entregar calendário de treinos com dados persistidos e isolamento por organização.
5. Definir e entregar presença/falta, nota e observação com autorização, histórico e testes PostgreSQL.
6. Definir regras financeiras e então implementar mensalidade, confirmação manual e comprovante WhatsApp; não automatizar cobrança nesta etapa.
7. Completar dashboard apenas após haver dados confiáveis; implementar armazenamento seguro Mobile antes de lançamento.

## Decisões pendentes do Artur

- Para menores de 16 anos: campos e relação com responsável, consentimento exigido, quem pode confirmar e como atualizar/remover esses dados.
- Se trainer pode cadastrar atletas e quais permissões tem para turmas, presença e observações.
- Ciclo de vida das turmas e como transferências de atleta preservam histórico.
- Calendário: fuso horário, recorrência, cancelamento e prazo para alterações.
- Mensalidade: valor, dia de vencimento, regra exata dos três dias, exceções/acordos e responsáveis pela confirmação manual.
- Conteúdo e destino do botão WhatsApp; evitar incluir dados pessoais desnecessários em URL/mensagem.
- Provider homologado de SMS e procedimento de testes sem contatos em Git.

## Estado Git após criar este relatório

`git status --short --branch`:

```text
## mobileatleta...origin/mobileatleta
?? docs/mvp-1.0/
```

`git diff --stat` não imprime linhas: o único arquivo é novo e não está staged (nenhum `git add` foi executado). Não houve commit nem push.
