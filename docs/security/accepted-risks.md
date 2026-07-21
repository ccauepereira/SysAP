# Riscos aceitos da fundacao

As excecoes abaixo sao decisoes temporarias e revisaveis. Elas nao autorizam
ignorar novos advisories nem ampliar o fluxo de dados sem nova analise.

## PostCSS 8.4.31

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
