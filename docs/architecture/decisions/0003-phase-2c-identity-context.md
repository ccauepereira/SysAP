# ADR 0003 — Contexto de identidade e sessão local da Fase 2C

- **Status:** aprovado para implementação incremental
- **Data:** 26 de julho de 2026
- **Escopo:** contrato, schema, RLS, JWT/JWKS e middleware HTTP; sem endpoint de identidade

## Contexto

O Supabase Auth autentica credenciais e emite JWTs, mas um JWT
criptograficamente válido não demonstra que o acesso ainda está ativo no SysAP.
Uma membership pode ser suspensa e o sign-out no provider não invalida um access
token já emitido antes de sua expiração. Papel e seleção de organização também
são estado de negócio do PostgreSQL, não claims aceitas do cliente.

## Decisão

- Supabase Auth permanece responsável por senha, OTP, MFA, refresh e emissão
  de JWT. A API Go valida JWTs e decide autorização a partir do PostgreSQL do
  SysAP.
- `app.auth_sessions` registra somente `session_id`, perfil, AAL, instante de
  registro e eventual revogação. Não contém bearer token, senha, OTP ou segredo
  TOTP. Isso permite negar localmente uma sessão ainda coberta por um JWT não
  expirado.
- A API usará `set_config` parametrizado e `is_local = true` para definir
  subject, session e organização somente dentro da transação. As funções
  `app.current_auth_subject_id()` e `app.current_auth_session_id()` tratam GUC
  ausente, vazio ou inválido como `NULL`; não há contexto persistente no pool.
- O papel de negócio e o estado de acesso são lidos de
  `app.organization_memberships`. Uma claim JWT nunca promove papel nem escolhe
  tenant. Toda rota vinculada a tenant exigirá `X-Organization-ID` como seletor
  não confiável, validado contra a membership autoritativa.
- `/v1/me` é uma leitura global do próprio perfil e de suas memberships; não
  seleciona uma organização e não recebe `X-Organization-ID`.
- `sysap_api` recebe somente `USAGE` no schema já privado e `SELECT` em
  `app.auth_sessions`. Escrita em sessões fica negada até o fluxo de emissão,
  logout e revogação estar implementado e revisado.
- RLS é defesa em profundidade. Ela permite ao papel limitado resolver apenas
  o perfil do subject, suas memberships e sua própria sessão; sem contexto,
  retorna zero linhas. Não substitui a autorização da API.
- A policy 2B de leitura de perfis por tenant é removida nesta fundação porque
  ela consultava memberships, enquanto a leitura das próprias memberships
  precisa consultar profiles para ligar `profile_id` ao subject. Manter ambas
  cria recursão de RLS no PostgreSQL. A 2C.0 fica deliberadamente mais
  restritiva: só permite a leitura do próprio perfil. Projeções de perfil de
  terceiros exigirão policy específica de uma entrega futura, sem reintroduzir
  esse ciclo.
- A API aceita apenas JWT ES256 com `kid`, assinatura conferida contra o JWKS
  configurado no servidor, issuer, audience, expiração, `nbf` quando presente,
  `sub` UUID e `session_id` UUID. `aal` ausente é normalizado para `aal1`; só
  `aal1` e `aal2` são aceitos. Claims de papel ou organização não autorizam
  domínio; `role`, quando presente, precisa ser `authenticated`.
- O middleware extrai exclusivamente `Authorization: Bearer <token>`, valida o
  JWT e resolve a sessão local dentro de `WithAuthenticatedContext`. Sessão
  inexistente, revogada, associada a outro subject, AAL divergente ou perfil
  com `suspended_at` resulta na mesma resposta segura de autenticação.
- AAL é transportado no contexto imutável da requisição, mas não concede papel
  nem acesso administrativo. Uma fase posterior aplicará AAL2 às ações que o
  exigirem.

## Consequências

O contexto transacional reduz o risco de vazamento entre requisições reutilizando
uma conexão. A revogação local introduz uma consulta adicional em rotas
protegidas, necessária para suspensão e logout imediatos. A migration é
forward-only: qualquer reversão operacional será uma migration compensatória
após verificar dependências e dados existentes.

## Limite desta fase

Esta fase não implementa login, cookies, refresh, SMS, OTP, MFA, handler ou
endpoint funcional de `/v1/me`, nem escrita real em sessões.

As próximas entregas permanecem separadas: 2C.4 implementará `/v1/me`, 2C.5
implementará cache e rotação de JWKS e 2F aplicará AAL2 às ações sensíveis.
