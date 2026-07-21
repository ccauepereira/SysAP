# Fundação do dashboard Web

## Objetivo e pessoa principal

A Subfase 1C cria a fundação responsiva do painel do SysAP para Artur, treinador
que usa principalmente o celular durante a rotina. A tela é uma demonstração
visual e técnica: ainda não administra atletas, presença ou treinos e não
persiste nenhuma ação.

Um único aplicativo Next.js atende celular, tablet e computador. Não existe um
dashboard mobile separado, o que mantém semântica, dados, acessibilidade e
evolução do produto em uma única base de código.

## Direção visual

A direção “Laboratório de Performance + Centro de Comando” combina superfícies
obsidianas, painéis grafite, bordas finas e linhas táticas discretas. O dourado
da Artur Performance sinaliza identidade e ações prioritárias sem dominar a
interface. Verde, amarelo e vermelho são reservados a estados, sempre
acompanhados de texto.

A logo em `apps/web/public/brand/artur-performance-logo.png` é uma cópia byte a
byte do original canônico. Ela é exibida inteira, sem recorte, filtro, sombra,
recoloração ou mudança de proporção. Os mockups fornecidos são apenas referência
de direção e não são assets da aplicação.

## Hierarquia e responsividade

No celular, a ordem é saudação, data, indicadores 2 × 2, chamada futura,
alertas, atletas em lista, distribuição, gráfico simplificado, próximo treino e
estado técnico. A navegação inferior é fixa, respeita a safe area e o conteúdo
reserva espaço para não ficar coberto.

Entre 768 e 1199 px, a navegação vira uma barra lateral compacta e o conteúdo
usa uma ou duas colunas conforme o espaço. A tabela só aparece a partir de
1024 px; abaixo disso, a lista evita comprimir informações operacionais.

A partir de 1200 px, a sidebar mostra rótulos, os quatro indicadores ocupam uma
linha e a área principal separa atletas/gráfico de alertas/distribuição/agenda.
O conteúdo tem largura máxima para preservar leitura em monitores grandes.

## Tokens e tipografia

Os tokens de marca documentados foram mantidos:

- `--brand-gold: #D4AE29`, com texto escuro `--text-on-gold`;
- `--brand-gold-hover: #EAC855`;
- `--background: #080A0C`;
- `--surface: #0D1117`;
- `--text-primary: #F7F7F5`.

Tokens exclusivos da interface completam o sistema sem alterar a marca:
`--surface-elevated`, `--border`, `--border-brand`, `--text-secondary`,
`--status-good`, `--status-warning`, `--status-critical`, `--focus-ring`, escala
de espaçamento, raios de 8 a 12 px e `--shadow-panel`. As combinações de texto e
superfície priorizam contraste WCAG AA.

Manrope é usada na interface; Barlow Condensed enfatiza métricas e números.
Ambas são empacotadas localmente por Fontsource, sem CDN.

## Dados demonstrativos

O marcador “Dados demonstrativos” aparece antes do título. Métricas, atletas,
alertas, distribuição, série histórica e próximo treino vêm de fixture
TypeScript isolada e são explicitamente fictícios. Avatares usam apenas
iniciais. Não existem imagens remotas, rostos inventados, dados pessoais, seed,
armazenamento local ou simulação de persistência.

Alertas recomendam revisão humana e verificação de dados. Eles não diagnosticam,
não prognosticam e não alegam risco de lesão.

## Integração técnica

O painel consulta exclusivamente `GET /healthz` e `GET /readyz` no servidor. A
variável `SYSAP_API_BASE_URL` não possui prefixo público e não vai para o bundle
do navegador. O adapter usa `fetch` nativo, timeout, `cache: "no-store"`,
validação mínima de status e JSON e retorna somente quatro estados seguros:

- API online e banco pronto;
- API online e banco indisponível;
- API indisponível;
- resposta inesperada.

Erros internos, URL completa e `request_id` não chegam à interface. Qualquer
indisponibilidade preserva o dashboard demonstrativo.

Datas partem de um instante UTC e são formatadas somente na apresentação em
português do Brasil e `America/Fortaleza`. A função aceita uma data injetada
para testes determinísticos.

## Acessibilidade

A estrutura fornece skip link, `main`, `aside` e `nav` nomeados, um único `h1`,
headings hierárquicos, tabela com cabeçalhos e lista mobile semântica. O gráfico
SVG possui título, descrição e alternativa textual. Ícones decorativos são
ocultos; status incluem texto; foco é visível; alvos principais têm ao menos
44 × 44 px; zoom e navegação por teclado permanecem disponíveis. Animações são
curtas e desativadas com `prefers-reduced-motion`.

Ações futuras ficam realmente desabilitadas e explicam “Em breve”. A navegação
futura não usa links falsos.

## Segurança

O Next.js remove `X-Powered-By` e envia `nosniff`, política de referência,
negação de framing e `Permissions-Policy` restritiva. Não há HSTS local, CSP
improvisada, conteúdo remoto, analytics, telemetria, segredo público ou variável
`NEXT_PUBLIC_*`.

## Fora de escopo

Continuam fora desta subfase autenticação, Supabase Auth, domínio e persistência
de atletas/turmas/presenças/treinos, chamada funcional, criação de treino,
feedback, GPS, mapas, heatmap, smartwatch, Health Connect, upload, relatórios,
notificações, Android, OpenAPI, CI e deploy. PWA, service worker, modo offline e
push notification ficam deliberadamente para etapa posterior, quando houver um
caso de uso e estratégia de atualização/cache definidos.
