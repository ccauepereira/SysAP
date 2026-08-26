# SysAP — Fase 2C

## Arquitetura de identidade, autenticação, autorização e segurança

**Versão:** 1.0  
**Data:** 26 de julho de 2026  
**Status:** proposta pronta para implementação em cortes  
**Responsável pela decisão:** Cauê / SysAP  
**Escopo:** arquitetura completa da Fase 2 e limites executáveis da Subfase 2C

---

## 1. Decisão executiva

A Fase 2C será o núcleo de confiança do SysAP. Ela não criará ainda a tela de login nem enviará SMS real. Sua responsabilidade será provar que a API Go consegue:

1. receber um access token emitido pelo Supabase Auth;
2. verificar assinatura e claims localmente por JWKS, sem consultar o Auth a cada requisição;
3. exigir uma sessão local registrada e não revogada;
4. resolver o usuário e suas permissões exclusivamente no banco;
5. selecionar uma organização explicitamente e impedir vazamento entre organizações;
6. bloquear imediatamente usuário, vínculo ou sessão, mesmo se o JWT ainda estiver dentro da validade;
7. executar uma primeira rota protegida real, `GET /v1/me`;
8. falhar de forma segura e observável, sem registrar token, senha, OTP ou segredo.

O Supabase Auth será o provedor de identidade e ficará responsável por credenciais, OTP, MFA, refresh tokens e emissão de JWT. A API Go continuará sendo a autoridade para regras de negócio, sessões permitidas, papéis, status, organização e escopo de acesso.

> Regra central: **um JWT válido prova que o Supabase autenticou uma identidade; ele não prova que essa identidade continua autorizada no SysAP.**

---

## 2. Resultado final pretendido da Fase 2

Ao término de toda a Fase 2, e não apenas da 2C, o sistema deverá oferecer:

- acesso do atleta por matrícula de 10 dígitos e senha escolhida por ele;
- matrícula no formato `AAAA######`, usando o ano de cadastro e seis dígitos gerados com aleatoriedade criptográfica;
- ativação inicial por OTP enviado ao telefone previamente cadastrado;
- senha escolhida pelo atleta, sem composição previsível baseada em nome ou nascimento;
- autenticação da equipe administrativa por e-mail e senha;
- MFA TOTP obrigatório para `owner` e `trainer`;
- sessões curtas, renováveis e revogáveis;
- logout da sessão atual e logout de todos os dispositivos;
- recuperação segura sem revelar se matrícula, e-mail ou telefone existem;
- suspensão imediata por perfil, vínculo organizacional ou sessão;
- trilha de auditoria segura para operações sensíveis;
- isolamento comprovado entre organizações;
- Web usando BFF e cookies seguros;
- futuro aplicativo mobile sem acesso direto às tabelas de negócio.

---

## 3. Limite exato da Subfase 2C

### 3.1 Dentro da 2C

- contratos de domínio para identidade e autorização;
- verificador JWT local com JWKS;
- validação rígida de header, assinatura e claims;
- cache limitado e seguro de JWKS;
- registro local de sessões autorizadas;
- revogação e suspensão verificadas em toda requisição protegida;
- contexto autenticado e contexto de organização;
- matriz central de permissões para `owner`, `trainer` e `athlete`;
- isolamento por organização em duas camadas: autorização da aplicação e RLS;
- `GET /v1/me` como primeiro corte vertical protegido;
- migration estritamente necessária para sessão/contexto;
- contratos OpenAPI diretamente afetados;
- logs, métricas e auditoria mínimos;
- testes unitários, integração PostgreSQL, concorrência controlada, fuzz e casos adversariais;
- documentação e runbook operacional do verificador.

### 3.2 Fora da 2C

Permanecem para cortes seguintes:

- criação de atleta e convite completo;
- provisionamento do usuário no Supabase Auth;
- envio de SMS real;
- ativação por OTP;
- login por matrícula e senha;
- login administrativo;
- matrícula automática;
- recuperação de senha;
- refresh e rotação de refresh token;
- cookies, BFF e telas Web de autenticação;
- configuração e desafio TOTP;
- integração Flutter/Android;
- teste volumétrico de 10 mil requisições;
- deploy e Supabase remoto.

A arquitetura desses fluxos está definida neste documento, mas eles **não devem ser implementados silenciosamente na 2C**.

---

## 4. Fronteiras de confiança

```text
┌──────────────────────────────────────────────────────────────┐
│ Cliente Web / futuro app mobile                              │
│ Não contém service_role, senha de banco nem regra de acesso  │
└──────────────────────────────┬───────────────────────────────┘
                               │ HTTPS
                               ▼
┌──────────────────────────────────────────────────────────────┐
│ API Go do SysAP                                              │
│ 1. limita requisição                                         │
│ 2. valida JWT por JWKS                                       │
│ 3. confirma sessão local                                     │
│ 4. resolve perfil e organização                              │
│ 5. aplica RBAC/ABAC                                          │
│ 6. abre transação e define contexto RLS                      │
└───────────────┬──────────────────────────────┬───────────────┘
                │ conexão direta               │ HTTPS fixo
                ▼                              ▼
┌─────────────────────────────┐   ┌────────────────────────────┐
│ PostgreSQL                  │   │ Supabase Auth              │
│ Fonte de autorização        │   │ Fonte de autenticação      │
│ sessões, perfis, vínculos,  │   │ senha, OTP, MFA, refresh,  │
│ papéis, status e auditoria  │   │ JWT e JWKS                 │
└─────────────────────────────┘   └────────────────────────────┘
```

### 4.1 Autoridades

| Decisão | Autoridade |
|---|---|
| Senha correta, OTP correto, MFA concluído | Supabase Auth |
| Assinatura e validade criptográfica do JWT | API, usando JWKS confiável |
| Usuário continua habilitado | PostgreSQL do SysAP |
| Sessão é aceita pelo produto | PostgreSQL do SysAP |
| Organização selecionada | Requisição + validação no PostgreSQL |
| Papel `owner`, `trainer` ou `athlete` | PostgreSQL do SysAP |
| Trainer pode acessar determinado atleta | PostgreSQL do SysAP |
| Operação exige AAL2 | Matriz de permissão da API |
| RLS permite consultar o registro | PostgreSQL, com contexto transacional |

Nenhum papel de negócio será aceito a partir de claim customizada do JWT.

---

## 5. Arquitetura modular da API Go

Estrutura recomendada:

```text
apps/api/internal/
├── identity/
│   ├── domain/
│   │   ├── assurance_level.go
│   │   ├── principal.go
│   │   ├── role.go
│   │   └── session.go
│   ├── application/
│   │   ├── authenticate_request.go
│   │   ├── authorize.go
│   │   └── get_current_user.go
│   ├── ports/
│   │   ├── token_verifier.go
│   │   ├── session_registry.go
│   │   ├── access_repository.go
│   │   └── security_audit_writer.go
│   ├── postgres/
│   │   ├── access_repository.go
│   │   └── session_registry.go
│   └── http/
│       ├── middleware.go
│       ├── organization.go
│       └── me_handler.go
├── integrations/
│   └── supabaseauth/
│       └── tokenverifier/
│           ├── verifier.go
│           ├── jwks_cache.go
│           └── config.go
└── platform/
    └── database/
        └── transaction_context.go
```

Não criar `utils`, `common` genérico ou uma hierarquia de abstrações sem uso concreto.

### 5.1 Portas centrais

```go
type TokenVerifier interface {
    Verify(ctx context.Context, rawToken string) (VerifiedToken, error)
}

type SessionRegistry interface {
    RequireActive(
        ctx context.Context,
        subjectID uuid.UUID,
        sessionID uuid.UUID,
    ) (Principal, error)
}

type AccessRepository interface {
    ListActiveMemberships(
        ctx context.Context,
        profileID uuid.UUID,
    ) ([]Membership, error)

    ResolveTenantAccess(
        ctx context.Context,
        profileID uuid.UUID,
        organizationID uuid.UUID,
    ) (TenantAccess, error)
}
```

Os nomes finais podem ser ajustados, mas os limites devem permanecer: verificação criptográfica, sessão local e autorização organizacional são responsabilidades distintas.

### 5.2 Objetos de domínio

```go
type VerifiedToken struct {
    SubjectID     uuid.UUID
    SessionID     uuid.UUID
    Assurance     AssuranceLevel
    IssuedAt      time.Time
    NotBefore     *time.Time
    ExpiresAt     time.Time
}

type Principal struct {
    ProfileID uuid.UUID
    SubjectID uuid.UUID
    SessionID uuid.UUID
    Assurance AssuranceLevel
}

type TenantAccess struct {
    OrganizationID uuid.UUID
    MembershipID   uuid.UUID
    Role           Role
    AthleteID      *uuid.UUID
}
```

O domínio não deve importar `net/http`, `pgx`, Supabase SDK ou tipos do framework.

---

## 6. Pipeline de uma requisição protegida

1. Rejeitar requisição acima dos limites HTTP.
2. Recuperar exatamente um header `Authorization`.
3. Exigir formato `Bearer <token>`, sem espaços adicionais ou múltiplos valores.
4. Limitar o bearer completo a 8 KiB.
5. Validar assinatura e claims do JWT.
6. Usar `sub` e `session_id` verificados para procurar sessão local ativa.
7. Confirmar que a sessão pertence ao mesmo perfil/subject.
8. Para rota global como `/v1/me`, operar apenas no escopo do próprio perfil.
9. Para rota de organização:
   - exigir `X-Organization-ID`;
   - validar UUID canônico;
   - resolver vínculo ativo;
   - derivar papel do banco;
   - exigir AAL compatível;
   - abrir transação;
   - definir o contexto RLS de forma parametrizada;
   - executar o caso de uso e concluir a transação.
10. Produzir resposta segura com `Cache-Control: no-store`.
11. Registrar apenas metadados permitidos.

### 6.1 Regra de transação

Contexto de identidade e organização deve ser definido com `set_config(..., true)` dentro da mesma transação da consulta:

```sql
select set_config('app.current_auth_subject_id', $1, true);
select set_config('app.current_auth_session_id', $2, true);
select set_config('app.current_organization_id', $3, true);
```

Não concatenar valores em SQL. Não usar contexto persistente de sessão. O rollback/commit deve apagar o contexto antes de devolver a conexão ao pool.

---

## 7. Política definitiva de JWT e JWKS

### 7.1 Biblioteca

Usar uma biblioteca JOSE/JWK madura com suporte nativo a JWKS. A escolha arquitetural recomendada é `github.com/lestrrat-go/jwx/v3`.

Antes da implementação:

- fixar uma versão exata;
- verificar licença, manutenção, advisories e dependências transitivas;
- não instalar a versão “latest” sem auditoria;
- registrar a decisão em ADR.

### 7.2 Algoritmo e chave

- Produção aceitará apenas `ES256`.
- A chave pública deve ser EC P-256.
- `alg=none`, HMAC, RSA e qualquer algoritmo fora da allowlist serão rejeitados.
- Não haverá fallback para o antigo segredo JWT simétrico.
- A chave deve possuir `kid` único, `use=sig` ou operação de verificação compatível.

### 7.3 Origem do JWKS

A URL do JWKS será derivada de configuração confiável, nunca de dado do token.

Regras:

- HTTPS obrigatório fora de desenvolvimento/teste;
- HTTP permitido apenas para `localhost`, `127.0.0.1` e `::1`;
- sem userinfo, fragmento ou query;
- redirects rejeitados;
- DNS/host não são fornecidos pelo usuário;
- timeout de conexão e leitura;
- status HTTP 200 obrigatório;
- `Content-Type` JSON;
- corpo máximo de 64 KiB;
- máximo de 8 chaves;
- `kid` sem duplicidade.

Headers `jku`, `x5u`, `jwk` ou extensões críticas não reconhecidas serão rejeitados. Eles jamais poderão alterar o destino de rede.

### 7.4 Header JOSE

Exigir:

- token JWS compacto com três segmentos;
- `typ=JWT`;
- `alg=ES256`;
- `kid` presente;
- `kid` com 1 a 128 caracteres em `[A-Za-z0-9._-]`;
- ausência de campos ambíguos ou duplicados.

### 7.5 Claims obrigatórias

| Claim | Regra |
|---|---|
| `iss` | igualdade exata com issuer configurado |
| `aud` | contém exatamente a audiência aceita, inicialmente `authenticated` |
| `exp` | obrigatório, futuro e dentro do limite de vida aceito |
| `iat` | obrigatório, não muito à frente do relógio da API |
| `nbf` | opcional, mas validado quando presente |
| `sub` | UUID canônico, não zero |
| `session_id` | UUID canônico, não zero |
| `role` | valor técnico `authenticated`; não usado como papel de negócio |
| `aal` | `aal1` ou `aal2` |

Leeway máximo: 30 segundos. O access token deverá ter vida curta; objetivo operacional de 10 minutos e teto aceito de 15 minutos.

Claims com tipo incorreto, números fora da faixa, arrays inesperados ou campos duplicados devem falhar.

### 7.6 Cache e rotação

- fazer warm-up do JWKS na inicialização sem impedir o processo de subir;
- se nenhuma chave confiável tiver sido carregada, rotas protegidas retornam 503;
- chaves já armazenadas podem continuar verificando tokens durante falha temporária do Auth;
- respeitar cache do provedor, com TTL máximo local de 10 minutos;
- aceitar stale controlado por no máximo 20 minutos durante indisponibilidade;
- `kid` desconhecido dispara no máximo um refresh coordenado;
- refresh concorrente usa `singleflight`;
- após falha, aplicar cooldown de 30 segundos;
- valores aleatórios de `kid` não podem produzir uma requisição de rede por tentativa;
- rotação deve aceitar chave antiga enquanto tokens legítimos ainda puderem estar válidos;
- remoção emergencial exige runbook para expurgar cache/reiniciar instâncias.

O cache de JWKS é permitido. Cache positivo de autorização e suspensão não será criado na 2C.

---

## 8. Sessão local revogável

### 8.1 Motivo

O logout do provedor remove a sessão/refresh token, mas um access token já emitido pode permanecer criptograficamente válido até `exp`. Por isso o SysAP terá um registro local de sessões autorizadas.

### 8.2 Tabela proposta

```sql
create table app.auth_sessions (
    session_id uuid primary key,
    profile_id uuid not null references app.profiles(id),
    assurance_level text not null
        check (assurance_level in ('aal1', 'aal2')),
    registered_at timestamptz not null default now(),
    revoked_at timestamptz,
    revocation_reason text,
    created_by_operation_id uuid,
    check (
        (revoked_at is null and revocation_reason is null)
        or
        (revoked_at is not null and revocation_reason is not null)
    )
);
```

Decisões:

- não armazenar access token nem refresh token;
- não armazenar senha, OTP ou segredo TOTP;
- não atualizar `last_seen_at` em toda requisição;
- não aceitar sessão existente no Supabase que não esteja registrada no SysAP;
- registrar a sessão somente após um fluxo de login aprovado pela API;
- na 2C, testes de integração podem criar sessões fictícias diretamente no banco local.

### 8.3 Suspensão imediata

Toda requisição protegida deve confirmar:

1. sessão existe;
2. sessão pertence ao `sub`;
3. sessão não foi revogada;
4. perfil está ativo;
5. vínculo com a organização está ativo.

Efeito:

- revogar uma sessão bloqueia o próximo request;
- suspender um perfil bloqueia todas as organizações;
- suspender um vínculo bloqueia somente aquela organização;
- remover atribuição trainer–athlete bloqueia o acesso ao atleta no próximo request;
- JWT ainda válido não contorna nenhuma dessas decisões.

---

## 9. Organizações, papéis e autorização

### 9.1 Seleção de organização

Toda rota vinculada a uma organização deve exigir `X-Organization-ID`, inclusive rotas cujo path contém um identificador opaco.

O header é apenas um seletor não confiável. Ele não concede acesso.

Processo:

1. validar formato;
2. localizar vínculo ativo do perfil;
3. definir contexto RLS;
4. buscar o recurso dentro do tenant;
5. aplicar permissão de papel e, quando necessário, vínculo trainer–athlete.

### 9.2 Papéis

| Papel | Escopo |
|---|---|
| `owner` | administração completa da própria organização |
| `trainer` | operações esportivas e atletas atribuídos |
| `athlete` | apenas seus próprios dados e ações permitidas |

Papéis vêm de `app.organization_memberships`, nunca do JWT.

### 9.3 Matriz inicial

| Capacidade | owner | trainer | athlete | AAL |
|---|---:|---:|---:|---|
| Ler próprio perfil e vínculos | sim | sim | sim | AAL1 |
| Ler configuração da organização | sim | sim, limitada | não | AAL1 |
| Convidar atleta | sim | sim, se permitido | não | AAL2 |
| Alterar papel/status de membro | sim | não | não | AAL2 |
| Suspender acesso | sim | não | não | AAL2 |
| Ler atleta atribuído | sim | sim | próprio | AAL1 |
| Alterar mensalidade/configuração comercial | sim | não | não | AAL2 |
| Configurar MFA da própria conta | sim | sim | opcional | sessão reforçada |

A matriz deve ser código central testável, não condicionais espalhadas em handlers.

### 9.4 Defesa em profundidade

Cada acesso a dado de negócio terá:

1. autorização de caso de uso na API;
2. query com tenant explícito;
3. contexto transacional;
4. RLS `FORCE`;
5. papel de banco sem `BYPASSRLS`, `SUPERUSER` ou propriedade das tabelas.

---

## 10. MFA administrativo

### 10.1 Política

- `owner` e `trainer` só podem operar com AAL2.
- O primeiro login pode concluir senha, mas permanece limitado até cadastrar TOTP.
- Operações privilegiadas exigem AAL2 atual, mesmo se a sessão iniciou em AAL1.
- O atleta usa AAL1 no MVP; MFA opcional pode ser avaliado depois.
- SMS não será segundo fator administrativo.

### 10.2 Fluxo futuro

1. credencial primária validada no Supabase;
2. API identifica vínculo de equipe;
3. se não houver TOTP, emitir autorização temporária de enrollment com uso único e vida curta;
4. usuário cadastra e confirma TOTP;
5. Supabase emite sessão AAL2;
6. API registra a sessão local;
7. rotas administrativas continuam verificando `aal=aal2`.

Nenhum “ticket de MFA” será um JWT de acesso comum. Ele deverá ter audiência, finalidade, expiração e consumo únicos.

---

## 11. Matrícula, senha e ativação por SMS

### 11.1 Matrícula

- exatamente 10 dígitos;
- primeiros 4 dígitos representam o ano do cadastro;
- últimos 6 dígitos são gerados com CSPRNG;
- unicidade verificada no banco;
- geração tenta novamente em colisão;
- matrícula é identificador público de login, não segredo;
- matrícula não revela data de nascimento, turma ou sequência previsível.

Não usar contador incremental.

### 11.2 Senha

- escolhida pelo atleta;
- mínimo 15 e máximo 128 caracteres;
- permitir passphrases e Unicode;
- rejeitar senha comprometida ou muito comum;
- não exigir padrão baseado em nome, matrícula ou nascimento;
- não armazenar nem registrar senha na API;
- não criar senha provisória previsível.

Se a política comercial exigir composição, ela deve ser avaliada por usabilidade. A arquitetura recomenda comprimento e bloqueio de senhas comprometidas em vez de exigir exatamente “dois especiais e quatro números”.

### 11.3 Ativação

1. owner/trainer cadastra o atleta e telefone verificado operacionalmente;
2. API gera matrícula e convite de uso único;
3. API solicita OTP ao adaptador Supabase/Twilio Verify;
4. resposta externa permanece genérica;
5. atleta envia OTP e cria sua senha;
6. operação idempotente associa `auth.users` ao perfil;
7. convite é consumido;
8. sessão local é registrada;
9. evento de auditoria é gravado.

OTP:

- seis dígitos;
- nunca salvo em claro pelo SysAP;
- nunca incluído em URL;
- vida curta;
- limite de tentativas;
- reenvio não invalida segurança nem permite spam;
- produção usa provedor real; CI usa OTP local documentado.

---

## 12. Rate limiting e antiabuso

O rate limiting deve combinar conta/matrícula, sessão, IP com prefixo truncado e finalidade. Não depender apenas de IP.

Valores iniciais:

| Fluxo | Limite |
|---|---|
| Login por matrícula | 5 falhas / 15 min por matrícula |
| Login por origem | 30 tentativas / 15 min por IP |
| Envio de OTP | 1 / 60 s, 5 / 30 min e 10 / dia por destino |
| Verificação de OTP | 5 tentativas por desafio |
| Recuperação | 3 / hora por conta e 10 / hora por IP |
| Refresh | limite por sessão e proteção contra replay |
| Endpoint protegido comum | orçamento por sessão/organização |

Respostas devem usar 429 e `Retry-After` quando seguro. A resposta externa não pode distinguir matrícula inexistente, suspensa, sem telefone ou convite expirado.

Testes de 10 mil requisições pertencem à 2H, em ambiente isolado e com objetivos definidos. Na 2C haverá apenas concorrência controlada para provar que o cache JWKS não amplifica chamadas externas.

---

## 13. Logout, revogação e recuperação

### 13.1 Logout atual

1. autenticar request;
2. revogar a sessão local em transação;
3. solicitar revogação ao Supabase;
4. limpar cookies no BFF;
5. resposta idempotente;
6. access token restante é bloqueado pela sessão local.

### 13.2 Logout de todos os dispositivos

- revogar todas as sessões locais do perfil;
- revogar sessões do provedor;
- registrar evento de segurança;
- não expor lista de tokens.

### 13.3 Refresh

- refresh token fica somente em cookie `HttpOnly`, `Secure` e `SameSite` apropriado no BFF;
- nunca em `localStorage`;
- rotação e uso único;
- replay revoga a família de sessão;
- access token não aparece no HTML nem em logs;
- CSRF protegido em endpoints mutáveis do BFF.

### 13.4 Recuperação

- resposta genérica;
- token/OTP de uso único;
- expiração curta;
- rate limiting;
- recuperação administrativa exige trilha de auditoria e autenticação forte;
- troca de senha revoga sessões existentes;
- não usar perguntas de segurança.

---

## 14. Bootstrap seguro do primeiro proprietário

Não haverá endpoint público de “criar owner”.

Estratégia:

1. comando administrativo one-shot executado por operador autorizado;
2. segredo operacional vindo de secret manager ou stdin, nunca argumento de shell;
3. somente em organização nova ou mediante procedimento break-glass explícito;
4. proibir criação se já existir owner ativo, salvo modo de recuperação auditado;
5. criar convite administrativo idempotente;
6. owner escolhe senha;
7. owner configura TOTP antes de ativar privilégios;
8. registrar ator, organização, timestamp, request/operation ID e resultado;
9. não usar seed de produção, senha padrão ou `SECURITY DEFINER` exposta.

A 2C apenas documenta e impede atalhos. O comando será implementado em corte específico após o fluxo de staff/MFA estar pronto.

---

## 15. Contratos HTTP e erros

### 15.1 Envelope

```json
{
  "error": {
    "code": "authentication_required",
    "message": "authentication is required",
    "request_id": "..."
  }
}
```

### 15.2 Códigos

| HTTP | Código | Uso |
|---:|---|---|
| 400 | `invalid_request` | header organizacional ou payload malformado |
| 401 | `authentication_required` | token ausente, inválido, expirado, sessão inexistente ou revogada |
| 403 | `access_denied` | identidade ativa sem papel, vínculo, atribuição ou AAL suficiente |
| 404 | `resource_not_found` | recurso inexistente ou pertencente a outro tenant |
| 409 | `operation_conflict` | estado concorrente ou operação já concluída de forma incompatível |
| 422 | `validation_failed` | campos válidos sintaticamente, mas rejeitados pelo contrato |
| 429 | `rate_limit_exceeded` | orçamento excedido |
| 503 | `identity_verification_unavailable` | nenhuma chave confiável disponível para verificar tokens |

Regras:

- 401 inclui `WWW-Authenticate: Bearer`;
- respostas de autenticação usam `Cache-Control: no-store`;
- erro interno, URL, DSN, claim, token, `kid`, matrícula ou telefone não aparecem;
- matrícula inexistente e existente produzem resposta equivalente em fluxos públicos;
- recurso de outro tenant e recurso inexistente produzem 404 equivalente;
- `/healthz` continua independente;
- o contrato atual de `/readyz` não muda na 2C sem revisão explícita do Web e OpenAPI.

### 15.3 `GET /v1/me`

Prova vertical da 2C:

```json
{
  "data": {
    "profile": {
      "id": "uuid",
      "display_name": "Cauê"
    },
    "memberships": [
      {
        "organization_id": "uuid",
        "role": "athlete",
        "status": "active"
      }
    ]
  }
}
```

Não retornar:

- `auth_user_id`;
- `session_id`;
- token;
- telefone completo;
- dados de outra organização;
- segredo ou informação interna do Supabase.

---

## 16. Logs, auditoria e métricas

### 16.1 Logs técnicos permitidos

- timestamp UTC;
- nível;
- evento estável;
- request ID;
- rota normalizada;
- status HTTP;
- duração;
- resultado por categoria;
- organização interna somente quando necessário e permitido.

### 16.2 Nunca registrar

- access token;
- refresh token;
- header Authorization;
- cookie;
- senha;
- OTP;
- segredo TOTP;
- URL autenticada;
- corpo bruto do provedor;
- matrícula/telefone/e-mail em logs comuns;
- `kid` bruto controlado por atacante.

### 16.3 Eventos de segurança

- `authentication_succeeded`;
- `authentication_failed`;
- `session_registered`;
- `session_revoked`;
- `authorization_denied`;
- `membership_suspended`;
- `otp_requested`;
- `otp_verification_failed`;
- `mfa_enrolled`;
- `mfa_required`;
- `recovery_started`;
- `owner_bootstrap_attempted`.

Eventos devem conter identificadores internos mínimos, ator, organização, ação, resultado e request ID. Payload adicional passa por allowlist e limite de profundidade/tamanho.

### 16.4 Métricas

- validações JWT por resultado;
- idade do cache JWKS;
- refresh de JWKS por resultado;
- `kid` desconhecido agregado;
- sessões revogadas;
- negativas de autorização por categoria;
- rate limit acionado;
- latência do banco;
- falhas do provedor Auth.

Evitar labels de alta cardinalidade com matrícula, usuário, token, telefone ou IP completo.

---

## 17. Threat model

### 17.1 Ativos

- credenciais e fatores de autenticação;
- access/refresh tokens;
- identidades Supabase;
- sessões locais;
- vínculos e papéis;
- dados pessoais e esportivos;
- dados comerciais;
- chaves de assinatura/JWKS;
- trilha de auditoria;
- disponibilidade de login e OTP.

### 17.2 Adversários

- pessoa externa automatizando login/OTP;
- atleta tentando acessar outro atleta;
- trainer tentando ampliar escopo;
- usuário suspenso com token antigo;
- atacante com token roubado;
- dependência ou upstream comprometido;
- operador interno com privilégio excessivo;
- erro de configuração de Auth, RLS ou chave;
- cliente Web comprometido por XSS/CSRF futuro.

### 17.3 Ameaças, controles e testes

| # | Ameaça | Severidade | Controle principal | Prova exigida |
|---:|---|---|---|---|
| 1 | JWT forjado | crítica | ES256 + assinatura por JWKS | token com assinatura inválida retorna 401 |
| 2 | `alg=none` ou confusão HMAC/EC | crítica | allowlist exata | variantes são rejeitadas |
| 3 | issuer/audience incorretos | alta | igualdade/containment estritos | matriz de claims adversariais |
| 4 | token expirado ou futuro | alta | `exp`, `iat`, `nbf`, leeway 30 s | testes de fronteira com relógio injetado |
| 5 | `kid` usado para SSRF | crítica | JWKS fixo; rejeitar `jku/x5u` | destino atacante recebe zero requests |
| 6 | tempestade de refresh JWKS | alta | singleflight + cooldown | 100 tokens/kids geram refresh limitado |
| 7 | JWKS gigante/malformado | alta | timeout, 64 KiB, máximo de chaves | reader cancelado e 503/401 seguro |
| 8 | chave rotacionada/removida | alta | cache limitado + runbook | testes old/new/outage |
| 9 | access token roubado | alta | vida curta + sessão local | revogação bloqueia próximo request |
| 10 | login direto no Auth contorna API | crítica | sessão não registrada é negada | JWT válido, session desconhecida = 401 |
| 11 | papel adulterado no JWT | crítica | papel vem do banco | claim `owner` não amplia acesso |
| 12 | IDOR entre organizações | crítica | tenant explícito + RBAC + RLS | matriz com duas organizações |
| 13 | trainer acessa atleta não atribuído | alta | relação de atribuição obrigatória | acesso cruzado retorna 404 |
| 14 | suspensão demora a valer | alta | consulta DB em cada request | JWT antigo falha após suspensão |
| 15 | contexto RLS vaza pelo pool | crítica | `SET LOCAL` transacional | reuso de conexão não herda tenant |
| 16 | papel do banco ignora RLS | crítica | sem owner/BYPASSRLS/SUPERUSER | catálogo e query negativa real |
| 17 | SQL injection no contexto | crítica | `set_config` parametrizado | entradas maliciosas não alteram SQL |
| 18 | mass assignment de papel/status | alta | DTO allowlist | campos proibidos retornam 422 |
| 19 | enumeração de matrícula | alta | respostas/timing equivalentes | conta existente/inexistente indistinguíveis |
| 20 | brute force de senha | alta | rate limit composto | bloqueio e recuperação controlada |
| 21 | OTP bombing/replay | alta | cooldown, cotas, uso único | reenvio/tentativas/concorrência |
| 22 | refresh token replay | crítica | rotação e família revogada | segundo uso encerra família |
| 23 | bypass de MFA | crítica | AAL2 em toda ação privilegiada | owner/trainer AAL1 recebem 403 |
| 24 | vazamento em logs | alta | redaction e allowlist | scanner captura logs e respostas |
| 25 | CSRF no BFF | alta | SameSite + token/origin | testes Web futuros |
| 26 | XSS rouba sessão | alta | HttpOnly + CSP futura | testes Web futuros |
| 27 | DoS por token/corpo | média/alta | limites antes do parse | inputs grandes cancelados cedo |
| 28 | clock incorreto | alta | UTC/NTP + métricas | skew além do limite falha |
| 29 | bootstrap cria owner indevido | crítica | processo one-shot, AAL2, auditoria | segunda criação é recusada |
| 30 | segredo no Git | crítica | scanners estado/histórico/CI | gate fail-closed |

### 17.4 Riscos residuais

- comprometimento completo do projeto Supabase ou da chave privada exige resposta operacional;
- access token furtado permanece útil até a revogação local ou expiração;
- rate limiting distribuído precisará de armazenamento consistente antes de múltiplas instâncias;
- SMS está sujeito a SIM swap e disponibilidade da operadora;
- conta administrativa depende da segurança do TOTP e dos códigos de recuperação;
- erros de política RLS continuam sendo risco crítico e exigem auditoria Sol independente.

---

## 18. Estratégia de testes da 2C

### 18.1 Verificador JWT

Gerar chaves ES256 efêmeras em teste. Cobrir:

- token válido;
- assinatura inválida;
- `none`, HS256, RS256 e algoritmo desconhecido;
- issuer errado;
- audience ausente, errada, string e array;
- `exp` ausente, expirado e no limite;
- `iat` futuro;
- `nbf` futuro;
- `sub` e `session_id` ausentes, zero ou malformados;
- `role` técnico incorreto;
- `aal` inválido;
- tipo incorreto de claims;
- claims/headers duplicados;
- token vazio, truncado, quatro segmentos e maior que 8 KiB;
- `kid` ausente, longo, inválido e desconhecido;
- `jku`, `x5u`, `jwk` e `crit`.

### 18.2 Servidor JWKS hostil

- redirect;
- HTTP 500;
- timeout;
- content type incorreto;
- JSON vazio/malformado;
- corpo acima de 64 KiB;
- mais de 8 chaves;
- `kid` duplicado;
- chave com curva/algoritmo/uso errado;
- rotação old/new;
- indisponibilidade com cache;
- refresh concorrente;
- cooldown após falha;
- confirmação de zero request para URL controlada pelo token.

### 18.3 PostgreSQL real

Com duas organizações e dados fictícios:

- sessão ativa, revogada, inexistente e pertencente a outro subject;
- perfil ativo e suspenso;
- vínculo ativo e suspenso;
- owner/trainer/athlete;
- trainer com e sem atribuição;
- `/me` retorna apenas vínculos próprios;
- tenant A não lê tenant B;
- sem contexto RLS não há dado;
- rollback remove contexto;
- conexão reutilizada não herda identidade/tenant;
- `sysap_api` sem atributos perigosos;
- funções e tabelas com privilégios exatos.

### 18.4 HTTP

- corpos de erro comparados byte a byte;
- `WWW-Authenticate`;
- `Cache-Control: no-store`;
- request ID;
- nenhuma enumeração;
- nenhum token/claim/DSN em resposta ou log;
- mesmos códigos para recurso ausente/alheio;
- API sobe sem Auth/JWKS, mas rota protegida falha com 503 seguro.

### 18.5 Qualidade

- `gofmt`;
- `go vet ./...`;
- `go test -race ./...`;
- `go test` com PostgreSQL real;
- fuzz para parser de Authorization/JWT e seleção de organização;
- OpenAPI validado;
- migration aplicável do zero duas vezes;
- scanner de segredos;
- auditoria de dependências fail-closed.

---

## 19. Cortes implementáveis e priorização

### 2C.0 — Fechar contrato e schema

**Objetivo:** eliminar ambiguidades antes do código.

Entregas:

- ADR JWT/JWKS;
- OpenAPI de `/v1/me`, erros e `X-Organization-ID`;
- migration de `app.auth_sessions`;
- contexto autenticado no banco;
- políticas RLS e privilégios;
- testes SQL negativos.

Responsáveis:

- Sol alto: aprova arquitetura e threat model;
- Terra alto: implementa migration/contrato;
- Antigravity 3.1 Pro alto: executa reset, catálogo e matriz repetitiva;
- Sol alto: revisão read-only apenas do SQL/RLS.

### 2C.1 — Verificador JWT/JWKS

**Objetivo:** transformar bearer em `VerifiedToken` confiável.

Entregas:

- adaptador Supabase Auth;
- cache, limites, singleflight e clock injetável;
- suíte adversarial sem banco;
- logs e métricas seguros.

Responsáveis:

- Terra alto: implementação principal;
- Antigravity 3.6 Flash médio: testes mecânicos, matrizes e execução;
- Sol alto: revisar algoritmo, claims, SSRF e cache.

### 2C.2 — Sessão e identidade local

**Objetivo:** bloquear sessão não registrada, revogada ou incompatível.

Entregas:

- repositório de sessões;
- resolução `sub → profile`;
- contexto autenticado transacional;
- testes com PostgreSQL real.

Responsáveis:

- Terra alto: implementação;
- Antigravity 3.1 Pro alto: integração e cenários de banco;
- Sol alto: revisar revogação e RLS.

### 2C.3 — Organização e autorização

**Objetivo:** provar isolamento e matriz de papéis.

Entregas:

- seletor de organização;
- permission matrix;
- `owner`, `trainer`, `athlete`;
- atribuição trainer–athlete;
- AAL2 para operação privilegiada;
- 404 anti-IDOR.

Responsáveis:

- Terra alto: código de domínio/aplicação;
- Antigravity 3.1 Pro alto: matriz de testes;
- Sol alto: auditoria de cross-tenant e privilege escalation.

### 2C.4 — Corte vertical `/v1/me`

**Objetivo:** primeiro endpoint protegido ponta a ponta.

Entregas:

- handler;
- OpenAPI final;
- integração token → sessão → perfil → resposta;
- testes exatos e degradação segura.

Responsáveis:

- Terra alto: integração;
- Antigravity 3.6 Flash médio: gates, documentação e execução.

### 2C.5 — Hardening e encerramento

**Objetivo:** provar que a 2C pode sustentar os fluxos posteriores.

Entregas:

- fuzz e concorrência;
- threat model atualizado com evidências;
- runbook JWKS/rotação;
- auditoria de segredos e dependências;
- relatório final.

Responsáveis:

- Antigravity 3.1 Pro alto: execução volumosa;
- Terra alto: correções;
- Sol alto: auditoria final read-only;
- GitHub Actions: confirmação reproduzível.

---

## 20. Uso econômico dos agentes

| Trabalho | Executor recomendado |
|---|---|
| Decisão criptográfica, RLS, tenant isolation, threat model | Sol 5.6 alto |
| Código de domínio, JWT, sessão, autorização | Terra alto |
| Refatoração comum e correção localizada | Terra médio |
| Execução de comandos, matrizes repetitivas, docs e screenshots | Antigravity 3.6 Flash médio |
| Migration complexa e integração extensa | Antigravity 3.1 Pro alto |
| Ajuste simples e mecânico | Antigravity 3.5 Flash baixo/médio |
| Auditoria crítica final | Sol 5.6 alto, read-only |

Sol não deve escrever toda a feature. Ele define invariantes e audita trechos críticos. Terra implementa o núcleo. Antigravity absorve volume e repetição.

---

## 21. Critérios de segurança obrigatórios

A 2C só recebe PASS quando:

- apenas ES256 é aceito;
- JWKS nunca é escolhido pelo token;
- issuer, audience, exp, iat, nbf, sub, session_id, role técnico, aal e kid são validados;
- sessão desconhecida/revogada é negada;
- suspensão tem efeito no request seguinte;
- nenhum papel de negócio vem do JWT;
- toda rota tenant-bound exige organização explícita;
- duas organizações não conseguem ler ou alterar dados entre si;
- trainer não acessa atleta não atribuído;
- contexto RLS não vaza pelo pool;
- papel do banco não possui privilégios perigosos;
- `/v1/me` retorna apenas dados próprios;
- respostas e logs não vazam segredo ou identificador sensível;
- cache JWKS é limitado e resistente a tempestade;
- migrations aplicam do zero;
- testes adversariais e race detector passam;
- OpenAPI corresponde ao comportamento;
- CI e scanner de segredos passam;
- não houve Supabase remoto, SMS real, deploy ou funcionalidade fora do corte.

---

## 22. Bloqueadores de produção

Mesmo após a 2C, o sistema não estará pronto para usuários reais até:

- configurar signing key assimétrica no projeto de produção;
- comprovar claims reais emitidas pelo Supabase local e remoto controlado;
- implementar 2D–2F;
- configurar TOTP e recuperação;
- implementar rate limiting distribuído;
- definir provedor SMS, DPA e retenção;
- configurar secret manager;
- definir rotação de chave e resposta a incidente;
- revisar LGPD, consentimento e retenção;
- realizar pentest independente;
- configurar CSP, cookies e CSRF no BFF;
- definir backup/restore e auditoria operacional.

---

## 23. Como explicar esta arquitetura em uma entrevista

> “O SysAP separa autenticação de autorização. O Supabase Auth valida senha, OTP e MFA e emite um JWT curto. A API Go verifica esse JWT localmente com uma chave pública obtida por JWKS, usando uma allowlist de algoritmo e validação rígida de claims. Mesmo com token válido, a API consulta uma sessão local e o estado do perfil e do vínculo organizacional. Assim, uma suspensão ou revogação vale imediatamente. O papel do usuário não vem do token; vem do banco. Para cada requisição de organização, a API valida o vínculo, aplica a matriz de permissões e executa a consulta numa transação com contexto RLS. Isso cria duas barreiras independentes contra vazamento entre organizações.”

Pontos que Cauê deve conseguir demonstrar:

- diferença entre autenticação e autorização;
- motivo do JWT não bastar para suspensão imediata;
- função de `iss`, `aud`, `exp`, `nbf`, `sub`, `session_id`, `kid` e `alg`;
- por que `jku` e `x5u` podem causar SSRF;
- por que papel de negócio não deve vir do JWT;
- por que `SET LOCAL` precisa estar na mesma transação;
- diferença entre 401, 403 e 404 anti-IDOR;
- papel do RLS como defesa em profundidade;
- por que senha previsível baseada em nome/nascimento é insegura;
- por que testes de carga não substituem threat model e testes adversariais.

---

## 24. Referências técnicas

- [Supabase — JWT Signing Keys](https://supabase.com/docs/guides/auth/signing-keys)
- [Supabase — JSON Web Tokens](https://supabase.com/docs/guides/auth/jwts)
- [Supabase — Sessions](https://supabase.com/docs/guides/auth/sessions)
- [Supabase — Multi-Factor Authentication](https://supabase.com/docs/guides/auth/auth-mfa)
- [Supabase — Phone Login](https://supabase.com/docs/guides/auth/phone-login)
- [OWASP — Authentication Cheat Sheet](https://cheatsheetseries.owasp.org/cheatsheets/Authentication_Cheat_Sheet.html)
- [OWASP — Forgot Password Cheat Sheet](https://cheatsheetseries.owasp.org/cheatsheets/Forgot_Password_Cheat_Sheet.html)
- [OWASP — REST Security Cheat Sheet](https://cheatsheetseries.owasp.org/cheatsheets/REST_Security_Cheat_Sheet.html)
- [NIST SP 800-63B — Authentication and Lifecycle Management](https://pages.nist.gov/800-63-4/sp800-63b.html)
- [lestrrat-go/jwx v3](https://pkg.go.dev/github.com/lestrrat-go/jwx/v3)

---

## 25. Próxima ação recomendada

Não iniciar toda a 2C em um único prompt.

Ordem:

1. abrir branch `feat/phase-2c-identity-core`;
2. executar **2C.0**;
3. auditoria read-only do SQL/RLS;
4. executar **2C.1**;
5. auditoria do verificador;
6. executar **2C.2** e **2C.3**;
7. implementar `/v1/me`;
8. executar hardening e auditoria final;
9. somente então fazer commit/push/PR mediante autorização.

Cada corte deve começar com working tree limpo, origem sincronizada, diff de escopo e gates locais aprovados.
