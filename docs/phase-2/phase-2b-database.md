# Fase 2B - Banco de Identidade, Organizações e Isolamento

## Objetivo e Escopo
A Subfase 2B implementa a infraestrutura persistente e as restrições relacionais para identidade, controle de acesso e isolamento de tenants (organizações). Não inclui autenticação HTTP, JWT, endpoints ou interfaces de usuário. O foco é a estrutura do banco de dados, RLS e constraints fortes.

## Lista de Migrations
1. `20260722162001_phase_2b_identity_core.sql` - Core (organizations, profiles, organization_memberships, athletes, assignments)
2. `20260722162002_phase_2b_identity_workflows.sql` - Workflows (athlete_invitations, identity_operations, outbox_events, idempotency_records)
3. `20260722162003_phase_2b_identity_security.sql` - Security (security_audit_events, auditoria, RLS e triggers de proteção)

## Resumo das Tabelas
* `organizations`: Cadastro do tenant isolado.
* `profiles`: Informações básicas do usuário. Referencia globalmente `auth.users(id)`.
* `organization_memberships`: Vínculo entre Profile e Organization. Define o papel (owner, trainer, athlete) e estado de acesso.
* `athletes`: Detalhes específicos de matrícula.
* `trainer_athlete_assignments`: Atribuição de treinador a atleta.
* `athlete_invitations`: Fluxo de aceite de convite por atleta.
* `identity_operations`: Processamento assíncrono e tracking de operações de identidade.
* `outbox_events`: Eventos de domínio aguardando envio, garantindo integridade.
* `idempotency_records`: Proteção contra reprocessamento duplicado da mesma operação.
* `security_audit_events`: Tabela append-only para auditoria de eventos de segurança, rastreando acessos e mudanças estruturais.

## Matriz de Grants para sysap_api
As seguintes permissões foram concedidas ao papel `sysap_api`:

| Tabela                         | SELECT | INSERT | UPDATE | DELETE |
|--------------------------------|--------|--------|--------|--------|
| `organizations`                | Sim    | Não    | Sim    | Não    |
| `profiles`                     | Sim    | Não    | Sim    | Não    |
| `organization_memberships`     | Sim    | Sim    | Sim    | Não    |
| `athletes`                     | Sim    | Sim    | Sim    | Não    |
| `trainer_athlete_assignments`  | Sim    | Sim    | Sim    | Não    |
| `athlete_invitations`          | Sim    | Sim    | Sim    | Não    |
| `identity_operations`          | Sim    | Sim    | Sim    | Não    |
| `outbox_events`                | Sim    | Sim    | Sim    | Não    |
| `idempotency_records`          | Sim    | Sim    | Sim    | Não    |
| `security_audit_events`        | Sim    | Sim    | Não    | Não    |

* Nenhuma tabela possui INSERT, UPDATE ou DELETE global para organizações ou profiles sem restrições.
* As tabelas usam apenas fechamento ou inativação lógica. Nenhuma possui acesso DELETE.
* Roles públicos (`public`, `anon`, `authenticated`, `service_role`) tiveram todos os privilégios revogados via `REVOKE ALL`.

## Estratégia RLS
O banco adota a diretriz `Fail Closed`. `ENABLE ROW LEVEL SECURITY` e `FORCE ROW LEVEL SECURITY` foram aplicados a todas as tabelas. Para interagir com dados, o papel `sysap_api` obrigatoriamente depende do valor local da transação, estabelecido por `SET LOCAL app.current_organization_id = '<uuid>'`. Se o contexto estiver ausente ou mal formatado, as consultas silenciosamente falharão por causa das policies exigindo igualdade de UUID.

## Contexto Transacional
A API sempre setará a variável local no PostgreSQL no início da transação: `SET LOCAL app.current_organization_id = ...`. Como é local, a persistência morre instantaneamente no rollback ou commit. A função utilitária do PostgreSQL `app.current_tenant_id()` recupera esse valor. Não usamos funções de stateful caching ou globais de pool.

## Proteção do Último Owner
O banco previne a remoção de "active owners" (caso chegue a zero) com um `FOR UPDATE` da tabela `organizations`, e só permite o `UPDATE` (suspendendo, rebaixando) ou `DELETE` se o `COUNT` de owners ativos na organização não for zerado, evitando condições de corrida simultâneas que orfanariam a organização.

## Comandos de Validação
Os cenários da fase foram testados via testes de integração automatizados integrados ao CI e rodando local:
* `pnpm db:start` / `pnpm db:reset` (subida da stack local limpa).
* `pnpm test:api` (cobertura go da fundação e subfase 2B).
* `pnpm check` (tipagem geral).

## Estratégia de Rollback Destrutivo
Um rollback manual completo requer desconstruir dependências estritamente na ordem inversa.

### Rollback manual ordenado:
1. Revoke grants:
```sql
REVOKE ALL ON ALL TABLES IN SCHEMA app FROM sysap_api;
```
2. Drop policies e tables, e tipos se aplicável:
```sql
DROP TABLE app.security_audit_events;
DROP TABLE app.idempotency_records;
DROP TABLE app.outbox_events;
DROP TABLE app.identity_operations;
DROP TABLE app.athlete_invitations;
DROP TABLE app.trainer_athlete_assignments;
DROP TABLE app.athletes;
DROP TABLE app.organization_memberships;
DROP TABLE app.profiles;
DROP TABLE app.organizations;
```
3. Drop triggers e functions:
```sql
DROP FUNCTION app.check_audit_metadata_secrets() CASCADE;
DROP FUNCTION app.check_outbox_payload_secrets() CASCADE;
DROP FUNCTION app.check_identity_operations_transition() CASCADE;
DROP FUNCTION app.check_athlete_invitation_transition() CASCADE;
DROP FUNCTION app.protect_last_owner() CASCADE;
DROP FUNCTION app.contains_forbidden_keys(jsonb, integer) CASCADE;
DROP FUNCTION app.current_tenant_id() CASCADE;
```

### Impacto Destrutivo e Backups
Este rollback apaga estruturalmente todos os dados de cliente criados em Fases posteriores.
Em produção, rollback destrutivo (via DROP) é estritamente proibido de ser executado para retroceder estado, a não ser que uma migração compensatória mova os dados primeiro. Sempre execute back-up do DB (pg_dump) e back-up das instâncias gerenciadas do `auth.users` antes de migrar ou retroceder em ambiente de produção.

### Identidades Auth
Existe constraint `ON DELETE RESTRICT` nas tabelas em relação ao `auth.users`. Deletar uma linha do `auth.users` que tem referência ativa falhará. No processo de rollback, identidades criadas no `auth.users` associadas via `profile` no ambiente de teste ficarão órfãs na auth sem link, exigindo um flush (o `db:reset` local limpa inteiramente, então para testes locais não há risco. `db:reset` é proibido para produção). Em ambientes reais, eventuais identidades órfãs precisam de limpeza ou expiração manual, não sendo tratadas isoladamente neste rollback estrutural de Phase 2B.
