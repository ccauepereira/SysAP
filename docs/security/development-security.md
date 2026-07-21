# Seguranca no desenvolvimento

## Segredos e dados

- Chaves, tokens, JWTs, senhas, DSNs, certificados e dados pessoais nunca vao
  para o Git.
- `.env.local` e local, ignorado e protegido; `.env.example` contem somente
  nomes e valores ficticios.
- Se um valor vazar, revogue ou rotacione imediatamente. Apagar no commit
  seguinte nao limpa o historico Git.
- Antes do commit, revise nomes e conteudo staged e execute
  `pnpm security:secrets`.
- Fixtures e seeds usam somente dados claramente ficticios. Logs nao carregam
  PII, saude, feedback, URL de banco ou segredo.

Testes de seguranca e carga executam apenas localmente ou em staging autorizado.
Nunca rode carga, scanner ou migration experimental contra producao. Operacoes
Supabase remotas, incluindo login, link e push, exigem autorizacao explicita e
nao fazem parte da Fase 1.

## Dependencias

Dependencias diretas usam versoes exatas. Antes de adicionar ou atualizar:

1. confirme necessidade e origem oficial;
2. escolha release estavel publicada ha pelo menos sete dias;
3. revise licenca, manutencao e lifecycle scripts;
4. instale com scripts desabilitados quando possivel;
5. execute auditoria, testes e build;
6. revise manifest e lockfile integralmente.

`pnpm security:dependencies` falha para qualquer advisory novo e aceita somente
o risco PostCSS exato documentado. Ferramentas nao viram dependencias de
producao da API.

## Banco e menor privilegio

Migrations passam por revisao de grants, RLS, ownership, rollback e exposicao da
Data API. A API usa papel restrito; clientes nao acessam tabelas de negocio.
Dados de teste sao ficticios e o reset local nunca deve apontar para host remoto.

## GitHub Actions

Actions sao oficiais, fixadas por SHA completo com a tag revisada em comentario
e executadas em runner hospedado fixo. Checkout nao persiste credencial, as
permissoes globais sao somente `contents: read`, nao ha contexto `secrets.*`,
runner proprio, cache, deploy ou gatilho privilegiado. Atualizacoes de Action
seguem a mesma quarentena e revisao de licenca das dependencias.

Ferramentas revisadas nesta fase:

| Ferramenta | Versao | Origem | Licenca |
|---|---:|---|---|
| Redocly CLI | 2.39.0 | `Redocly/redocly-cli` / npm | MIT |
| Gitleaks | 8.30.1 | `gitleaks/gitleaks` / GHCR | MIT |
| Supabase CLI | 2.109.1 | `supabase/cli` / npm | MIT |

O Gitleaks usa a imagem oficial fixada pelo digest OCI completo e roda sem rede
durante a varredura. Relatorios sao temporarios, redigidos e nunca enviados como
artefato.

Actions revisadas e fixadas na CI:

| Action | Tag | SHA | Licenca |
|---|---:|---|---|
| `actions/checkout` | v4.2.2 | `11bd71901bbe5b1630ceea73d27597364c9af683` | MIT |
| `actions/setup-node` | v4.4.0 | `49933ea5288caeca8642d1e84afbd3f7d6820020` | MIT |
| `actions/setup-go` | v5.5.0 | `d35c59abb061a4a6fb18e82ac0862c26744d6ab5` | MIT |
| `pnpm/action-setup` | v4.1.0 | `7088e561eb65bb68695d245aa206f005ef30921d` | MIT |
