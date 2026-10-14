# Riscos aceitos da fundacao

As excecoes abaixo sao decisoes temporarias e revisaveis. Elas nao autorizam
ignorar novos advisories nem ampliar o fluxo de dados sem nova analise. O
processo esta em `docs/security/dependency-management.md`.

## braces 3.0.3 (ativa, temporaria)

- **Advisory:** `GHSA-vfj7-8cjw-p6xm`. O CVE `CVE-2026-93687` foi informado na
  tarefa e nao foi conferido na fonte oficial (acesso a `api.github.com` negado).
- **Severidade:** alta (relatorio do `pnpm audit`).
- **Origem:** somente desenvolvimento:
  `apps/web (devDependencies) > eslint-config-next@16.3.8 >
  @next/eslint-plugin-next@16.3.8 > fast-glob@3.3.1 > micromatch@4.0.8 >
  braces@3.0.3`. O `pnpm audit` marca o achado com `dev: true`.
- **Problema (descricao fornecida, nao verificada no advisory):** esgotamento de
  pilha por recursao profunda ao expandir padroes muito aninhados (CWE-674),
  causando negacao de servico do processo que expande o padrao.
- **Versao corrigida:** o relatorio indica `>=3.0.4`, mas o registry npm publica
  somente ate `braces@3.0.3` (`latest`) em 14/10/2026.
- **Decisao:** aceita temporariamente em 14 de outubro de 2026.
- **Responsavel:** Caue (mantenedor do SysAP).
- **Expira:** fim de 13 de novembro de 2026 em America/Fortaleza
  (`2026-11-14T03:00:00Z`). Depois disso `pnpm security:dependencies` falha.

Por que nao ha correcao segura publicada nesta data:

- `eslint-config-next`/`@next/eslint-plugin-next` `16.4.0` (`latest`) e
  `16.5.0-canary.10` (`canary`) continuam fixando `fast-glob@3.3.1`;
- `fast-glob@3.3.3` (`latest`) depende de `micromatch@^4.0.8`, e
  `micromatch@4.0.8` (`latest`) depende de `braces@^3.0.3`;
- remover `eslint-config-next` ou `@next/eslint-plugin-next` eliminaria as
  regras `@next/next/*` do lint;
- trocar `fast-glob` por outro pacote via alias de override colocaria codigo nao
  equivalente sob o nome de outra biblioteca, sem suporte nem teste upstream.

Evidencia de risco delimitado (comandos registrados na entrega):

- `pnpm --filter @sysap/web list --prod --depth Infinity` nao contem `braces`,
  `micromatch` nem `fast-glob`; o `next` empacota apenas `glob`, nao `braces`;
- o plugin so chama `fast-glob` em `get-root-dirs.js` quando
  `settings.next.rootDir` esta configurado; a configuracao atual nao o define;
- sonda de carregamento: durante `eslint .`, `braces` e carregado por
  `micromatch`, mas recebe zero chamadas; durante `next build`, nao e carregado.
  O controle positivo da mesma sonda registra a chamada;
- o padrao que chegaria a `braces` viria da configuracao versionada do
  repositorio, nao de entrada de usuario, atleta ou rede.

A excecao fica invalida, mesmo antes do vencimento, se `settings.next.rootDir`
for configurado, se `braces` aparecer em outro caminho ou em producao, se a
severidade mudar ou se surgir versao corrigida compativel. O gate confere
advisory, pacote, versao, severidade, `dev: true` e caminho exatos, e nunca
combina a excecao com outro advisory. Quando a versao corrigida for publicada,
atualize respeitando a quarentena, remova a excecao e este registro.

## PostCSS 8.4.31 (encerrada)

A excecao foi removida do gate em 26 de julho de 2026 (commit `955bb26`); o
grafo atual resolve `postcss@8.5.26`. O registro abaixo e mantido como historico.

- **Advisory:** `GHSA-qx2v-qp2m-jg93`.
- **Severidade:** moderada.
- **Origem:** dependencia transitiva `apps/web > next@16.2.10 > postcss@8.4.31`.
- **Problema:** XSS quando uma serializacao CSS nao escapa `</style>` antes de
  inserir resultado em um contexto HTML `style`.
- **Versao corrigida conhecida:** PostCSS 8.5.10 ou superior.
- **Decisao:** aceita provisoriamente em 21 de julho de 2026.

O fluxo atual nao recebe CSS do usuario, nao transforma entrada de usuario em
CSS e nao injeta HTML inseguro. Estilos sao fontes estaticas do repositorio e o
dashboard nao usa APIs de HTML bruto. Assim, o caminho vulneravel nao e
alcancavel na implementacao atual.

E proibido introduzir CSS controlavel por usuario, `dangerouslySetInnerHTML` ou
outra insercao HTML insegura enquanto essa versao permanecer. Qualquer um desses
fluxos invalida imediatamente a aceitacao e bloqueia a entrega.

A correcao deve vir por uma versao oficial e compativel do Next.js que atualize
sua dependencia, respeitando a quarentena, testes e regressao visual. Nao sera
feito override isolado de PostCSS. A excecao e conferida por package, versao,
advisory e caminho exatos e deve ser revista em toda atualizacao do Next.js. Se
o advisory desaparecer, o gate informa que a excecao nao e mais necessaria.

## Content Security Policy

CSP ainda nao foi implementada. Uma politica improvisada com `unsafe-inline`
seria falsa protecao e nao deve ser adicionada apenas para marcar um requisito.
Antes de exposicao publica ou autenticacao, os scripts, estilos, fontes, imagens
e conexoes realmente usados serao inventariados; entao a politica sera criada,
testada e endurecida sem liberar origens desnecessarias. Ela deve ser reavaliada
sempre que esses recursos mudarem.
