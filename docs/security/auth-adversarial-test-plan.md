# Plano de testes adversariais de identidade — Subfase 2H

- **Status:** planejado; nenhuma carga ou integração externa executada na 2A
- **Data:** 22 de julho de 2026
- **Ambientes permitidos:** local e staging expressamente autorizado

## Objetivo e critérios

Comprovar as invariantes da arquitetura de identidade sob entrada hostil,
concorrência e falha de dependência. Os testes não demonstram segurança
completa: fornecem evidência reproduzível para riscos conhecidos.

São condições de aprovação:

- nenhuma criação pública de usuário, staff ou organização;
- nenhuma leitura/escrita entre organizações;
- conta suspensa bloqueada apesar de JWT ainda válido;
- senha, OTP/TOTP, segredo de fator, ticket MFA, token, telefone completo,
  derivado enumerável e erro de provider ausentes de saída, log e artefato;
- idempotência e reconciliação sem conta órfã ignorada;
- proteção retorna `401`, `403`, `404`, `409` ou `429` conforme contrato sem
  virar oráculo de existência;
- serviço se recupera depois de falha e carga sem intervenção oculta.

## Regras de segurança do ambiente

- Nunca executar contra produção ou Supabase remoto.
- Testes automatizados e CI usam Supabase local, `auth.sms.test_otp`, apenas o
  número reservado fictício do contrato e egress externo bloqueado.
- Provider SMS real e credenciais ficam ausentes. Um número não mapeado deve
  falhar sem rede.
- Testes de carga usam somente provider falso/local; Twilio trial é proibido em
  carga e automação.
- Fixtures usam organizações, nomes, emails, telefones, tokens e IPs
  exclusivamente fictícios/reservados.
- Relatórios não guardam corpos de autenticação nem headers sensíveis.
- A ferramenta de carga será escolhida na 2H após revisão de manutenção,
  licença e dependências; esta subfase não adiciona ferramenta.

## Pirâmide de testes

### Unitários de domínio e aplicação

| Área | Casos mínimos | Evidência esperada |
|---|---|---|
| Matrícula | formato, ano Fortaleza em fronteira de UTC, CSPRNG injetado, colisões e limite de retry | Dez dígitos, unicidade e falha segura. |
| Estados | máquinas separadas de convite/operação/membership, terminais, suspensão/reativação e `activation_finalizing` | Erro de domínio estável, owner correto do estado e nenhuma mutação parcial. |
| Autorização | matriz completa por papel, tenant, estado e AAL | Negação por padrão e contexto derivado do servidor. |
| Idempotência | mesma chave/projeção, namespace HMAC de matrícula existente/inexistente, desafio real apenas após autenticar e comandos com senha/OTP | Replay determinístico ou `409`, nenhuma nova chamada externa/oráculo e nenhum segredo/derivado persistido. |
| Rate limiting | IP, HMAC de matrícula/telefone/convite, janelas, backoff e `Retry-After` | Limite combinado e sem lockout permanente. |
| Respostas seguras | mapeamento de erros Auth/PostgreSQL/SMS | Nenhum detalhe de provider ou existência. |
| Auditoria | allowlist por evento e redaction | Apenas metadados aprovados em UTC. |

Usar fakes determinísticos para `IdentityProvider`, `TokenVerifier`,
`SessionRegistry`, `MFAChallengeStore`, `EnrollmentNumberGenerator`, `Clock`,
`RateLimiter`, `SecurityAuditWriter` e `TransactionManager`. A aplicação nunca
precisa de fake Twilio direto.

### Integração PostgreSQL

- aplicar migrations da Fase 2 do zero em banco local limpo;
- provar constraints de matrícula, idempotência, estados e referências;
- provar grants e RLS com os papéis reais, inclusive `FORCE ROW LEVEL
  SECURITY` onde a futura modelagem decidir usá-lo;
- provar que `anon`, `authenticated`, `service_role` e cliente não acessam o
  schema de negócio;
- provar que o executor da API não é owner nem `BYPASSRLS`;
- concorrência de convite, ativação, suspensão e consumo da outbox;
- auditoria aceita `INSERT` pelo papel apropriado e nega `UPDATE`/`DELETE`;
- rollback documentado ou irreversibilidade explicitamente revisada.

### Integração Supabase Auth local

- provisionar somente identidade pré-cadastrada pelo adapter administrativo;
- confirmar signup global, email e anônimo desligados quando não usados,
  `auth.sms.enable_signup = false` e flag equivalente a
  `shouldCreateUser: false` em cada pedido de OTP;
- confirmar que OTP para sujeito inexistente não cria `auth.users`;
- confirmar no Auth local seis dígitos, política local de expiração, uso único,
  limite de tentativa e cooldown, sem atribuir o resultado ao Twilio Verify;
- usar apenas `auth.sms.test_otp`; confirmar zero tentativa de rede;
- ativar, definir senha, autenticar, refresh rotacionado, logout e logout global;
- provar que login/refresh negado revoga a sessão técnica e não entrega token;
- TOTP de staff: bootstrap/enrollment, AAL1, ticket de cinco minutos, challenge,
  verify, AAL2, uso único do ticket e limpeza do token AAL1. Repetição do código
  TOTP dentro da janela é comportamento do Auth a medir, não premissa; repetir
  também após restart e entre duas instâncias com a mesma chave versionada;
- indisponibilidade e timeout sem erro bruto; reconciliação após retorno;
- provar que senha/código e seus derivados não chegam ao PostgreSQL/log do
  SysAP; ticket MFA deixa somente digest HMAC e metadados permitidos.

O Auth local/test OTP não reproduz `twilio_verify`, e o adapter upstream usa a
origem externa do Verify. A aplicação é testada com `IdentityProvider` falso;
validade de cinco minutos, reenvio/reutilização, entrega e erros do provider só
podem ser validados em teste manual/staging expressamente autorizado na 2E.
Nunca fazem parte do gate automatizado ou da carga.

## Matriz de testes adversariais

| Categoria | Ataques/casos | Oráculo seguro esperado |
|---|---|---|
| Autorização cruzada | UUID de outro tenant em convite/membership, `X-Organization-ID` ausente/alheio e filtragem das memberships em `/me`; owner/trainer/athlete cruzados | Mesmo `404` para recurso alheio/inexistente; header não concede acesso e `/me` não vaza tenant. |
| Suspensão/logout | Suspender ou encerrar sessão entre duas requisições com access token válido e caches quentes | Próxima rota protegida nega pelo estado/`session_id`, mesmo antes de `exp`. |
| Credential stuffing | Senhas fictícias distribuídas por IP e matrículas | Limites por conta e rede; `401`/`429`; sem confirmação de conta. |
| Timing/enumeração | Amostras existentes/inexistentes, ativas/suspensas e senha correta/incorreta | Mesma mensagem/código; distribuição de tempo dentro do limiar aprovado na 2H. |
| OTP | inválido, expirado, repetido, de outro propósito/sujeito e tentativas paralelas | Falha genérica, consumo único e contador preservado. |
| Ticket MFA | fraco, expirado, repetido, concorrente, de outro sujeito/session/fator/finalidade e tentativas excedidas | Bearer >=256 bits, digest apenas, consumo atômico, erro genérico e sessão AAL1 revogada. |
| Tokens | truncado, assinatura alterada, `alg=none`, confusão HS/RS, claims ausentes/duplicados, `iss`/`aud`/`sub`/tipo errados | `401` sem parse/error interno. |
| Bypass Auth | sessão válida criada diretamente no Auth e `session_id` nunca registrado pela API | `401`; nenhum acesso de negócio, embora pumping continue coberto por controles externos. |
| `kid`/JWKS | `kid` desconhecido, enorme, URL, path traversal, muitos valores e rotação durante requisição | Um refresh controlado da origem fixa e falha fechada. |
| Refresh | replay fora/dentro da janela, single-flight, retry concorrente e token de outra sessão | Troca atômica sem cache idempotente; rotação conforme Auth e árvore revogada quando reuse é ataque. |
| CSRF | Origin ausente/hostil, token ausente/repetido, método simples e SameSite | BFF nega antes da mutação; API bearer não aceita cookie do browser como atalho. |
| XSS/sessão | payloads em nome/razão/erro; tentativa de ler cookies/tokens no HTML | Escape; cookie HttpOnly; nenhum token renderizado ou persistido em Web Storage. |
| SQL/mass assignment | metacaracteres, payload profundo, campos `role`, `organization_id`, `auth_user_id`, estado e provider | Query parametrizada, schema fechado e `422` sem ecoar valor. |
| Concorrência | cem pedidos iguais/diferentes para convite, resend, activate, recovery e suspend | Uma transição/comando efetivo; conflitos determinísticos. |
| Falhas externas | timeout/5xx/malformed antes/depois de OTP, senha e commits, inclusive `activation_finalizing` | Estado intermediário, nenhum segredo para retry, login reparador/novo desafio conforme resultado, reconciliação e `503` genérico. |
| Logs | CR/LF, Unicode de controle, payload grande e segredo em erro fake | JSON válido, tamanho limitado e redaction completa. |

### Fuzzing

- fuzz dos decoders HTTP e DTOs com limite de corpo/profundidade;
- matrícula, UUID, `X-Organization-ID`, Idempotency-Key, email, telefone E.164,
  senha Unicode, OTP/TOTP, ticket MFA e `reason_code`;
- headers JWT e claims com tipos inesperados, arrays, duplicatas e tamanhos
  extremos;
- transições e sequências de comandos geradas por estado;
- parsers nunca entram em panic, loop ilimitado ou alocação descontrolada;
- corpus versionado contém só dados fictícios e nenhum token válido.

### Timing

Executar aquecimento e amostras intercaladas de login/recovery para sujeitos
existentes e inexistentes no mesmo host isolado. Medir distribuição, não apenas
média, e definir o limiar estatístico na 2H antes de observar os resultados.
Ruído de rede e banco deve ser registrado. Não adicionar atraso fixo como única
defesa; igualar trabalho e limitar abuso são os controles primários.

### Auditoria e logs

Para cada evento esperado — pré-cadastro, matrícula, provisionamento, OTP,
ativação, login, refresh, logout, logout global, recuperação, alteração de
telefone, suspensão, reativação, papel e acesso negado — verificar:

1. exatamente um evento lógico apesar de retries;
2. `request_id`, timestamp UTC, resultado e IDs internos corretos;
3. nenhuma senha, OTP/TOTP, segredo de fator, ticket, sessão AAL1, token,
   cookie, Authorization, telefone/email completo, DSN, derivado enumerável ou
   erro bruto;
4. entradas não podem ser atualizadas ou removidas pelo papel da aplicação;
5. conteúdo hostil não quebra a estrutura do log.

## Teste de carga controlado

### Perfil

- **Total:** 10.000 requisições na campanha inteira, distribuídas entre os
  cenários e endpoints documentados.
- **Degraus sugeridos:** 1, 5, 10, 25 e 50 requisições concorrentes, com
  estabilização curta entre degraus e orçamento total fixo.
- **Mistura:** login inválido, OTP inválido, refresh, `/me`, convite idempotente
  e acesso cruzado, em proporções registradas no relatório.
- **Dados:** organizações/identidades fictícias pré-criadas e provider falso.
- **Proteção bem-sucedida:** `401`, `403` ou `429` pode ser resultado correto;
  contá-los separadamente de falha interna.

### Medições

- latência p50, p95 e p99 por operação e código;
- throughput e códigos `2xx`, `4xx`, `429`, `5xx`;
- memória, CPU, goroutines e pausas relevantes da API;
- conexões PostgreSQL ativas/em espera, contenção e deadlocks;
- filas/outbox, retries, idade de comandos e tamanho da auditoria;
- volume de OTP solicitado/aprovado no fake e zero acesso de rede externo;
- custo calculado deve permanecer zero no ambiente de teste.

### Critérios após carga

1. cessar tráfego e confirmar que rate limit/janelas se recuperam como
   documentado;
2. outbox/reconciliação converge sem comando preso ou conta órfã;
3. conexões, goroutines, memória e CPU retornam ao patamar definido antes do
   teste;
4. API aceita login legítimo e `/me` após o cooldown;
5. invariantes e auditoria continuam corretas;
6. nenhum SMS real, dado sensível ou artefato excessivo foi produzido.

Dez mil requisições não provam segurança, capacidade de produção ou
entregabilidade SMS. O objetivo é encontrar regressões e validar recuperação.

## Falhas injetadas e reconciliação

Em cada caso, interromper antes/depois do commit PostgreSQL e antes/depois da
resposta do Auth fake:

- PostgreSQL confirma e Supabase falha;
- Supabase confirma e PostgreSQL não finaliza;
- recovery deixa `identity_operations(account_recovery)` em `processing` após a
  senha mudar; login reparador deve concluí-la sem reter o segredo;
- SMS fake falha depois do provisionamento;
- processo reinicia com comando em andamento;
- mesma mensagem de outbox é processada duas vezes;
- JWKS fica indisponível durante rotação;
- relógio avança/retrocede no fake sem alterar o relógio real do host.

O estado final deve ser recuperável e explicável. Compensação destrutiva exige
prova de propriedade do recurso externo e auditoria; não é aceita limpeza
silenciosa.

## Relatório da 2H

O relatório deve registrar commit testado, ambiente, configuração não sensível,
quantidades, distribuição, versões, comandos, resultados, gráficos agregados,
falhas e risco residual. Não inclui números, nomes, emails, tokens, códigos ou
segredos. Toda exceção ganha owner, prazo e gate de revalidação.

Referências: [OWASP Web Security Testing Guide](https://owasp.org/www-project-web-security-testing-guide/),
[Authentication Cheat Sheet](https://cheatsheetseries.owasp.org/cheatsheets/Authentication_Cheat_Sheet.html),
[REST Security Cheat Sheet](https://cheatsheetseries.owasp.org/cheatsheets/REST_Security_Cheat_Sheet.html)
e [Supabase Auth rate limits](https://supabase.com/docs/guides/auth/rate-limits).
