# ADR 0003 — Contexto de identidade e sessão local da Subfase 2C.0

- **Status:** aprovado para implementação incremental
- **Data:** 26 de julho de 2026
- **Escopo:** contrato, schema, RLS e contexto transacional; sem autenticação HTTP

## Contexto

O Supabase Auth autentica credenciais e emite JWTs, mas um JWT
criptograficamente válido não demonstra que o acesso ainda está ativo no SysAP.
Uma membership pode ser suspensa e o sign-out no provider não invalida um access
token já emitido antes de sua expiração. Papel e seleção de organização também
são estado de negócio do PostgreSQL, não claims aceitas do cliente.

## Decisão

- Supabase Auth permanece responsável por senha, OTP, MFA, refresh e emissão
  de JWT. A API Go validará futuramente esses JWTs e decidirá autorização a
  partir do PostgreSQL do SysAP.
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

## Consequências

O contexto transacional reduz o risco de vazamento entre requisições reutilizando
uma conexão. A revogação local introduz uma consulta adicional em rotas
protegidas, necessária para suspensão e logout imediatos. A migration é
forward-only: qualquer reversão operacional será uma migration compensatória
após verificar dependências e dados existentes.

## Limite desta subfase

Esta 2C.0 não implementa login, Bearer middleware, JWT/JWKS, cookies, refresh,
SMS, OTP, MFA, handler ou endpoint funcional de `/v1/me`, nem escrita real em
sessões.

As próximas entregas da 2C permanecem separadas: 2C.1 selecionará e provará o
verificador JWT/JWKS; 2C.2 validará os claims técnicos permitidos, inclusive
`sub`, `session_id` e AAL; 2C.3 integrará essa validação ao contexto
transacional de rotas protegidas; 2C.4 implementará registro, logout, revogação
e suspensão imediata; 2C.5 implementará `/v1/me` e a matriz de autorização
correspondente. Cada uma exige revisão própria antes de alterar o fluxo.
