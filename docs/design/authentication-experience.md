# Experiência de autenticação — Web 2G.1

## Linguagem visual

A autenticação usa azul-marinho profundo (`#07162C`), superfície branco-quente
(`#FBFAF6`), dourado AP (`#D4AE29`) e texto azul-escuro sobre ações douradas.
Manrope permanece como fonte funcional e títulos usam uma serifada local do
sistema para criar contraste editorial. A textura tática é um SVG estrutural de
baixa opacidade; a única marca renderizada é
`assets/brand/artur-performance-logo.png`, servida pela cópia canônica já
existente em `apps/web/public/brand/`.

A abertura do login combina aparecimento da textura, uma linha dourada breve e
entrada da logo. Ela não captura interação e desaparece em pouco mais de dois
segundos. Com `prefers-reduced-motion: reduce`, a abertura não é renderizada.

## Rotas

- `/login`: matrícula e senha, conectado ao BFF same-origin da 2G.0.
- `/ativar`, `/ativar/verificar`, `/ativar/senha`, `/ativar/concluida`:
  apresentação controlada do fluxo de ativação.
- `/recuperar-acesso`, `/recuperar-acesso/verificar`,
  `/recuperar-acesso/nova-senha`, `/recuperar-acesso/concluida`: apresentação
  controlada da recuperação.
- `/verificar-identidade` e `/verificar-identidade/erro`: estados visuais de
  MFA administrativo.
- `/estado/carregando`, `/estado/sessao-expirada`,
  `/estado/conta-suspensa`, `/estado/servico-indisponivel` e
  `/estado/mfa-obrigatorio`: estados seguros reutilizáveis.

## Limite de integração

Somente o login chama `/api/auth/login`. O BFF atual não expõe ativação,
recuperação ou MFA ao browser; por isso essas páginas exibem um aviso de prévia,
apagam o conteúdo dos formulários antes de navegar e não enviam nem persistem
matrícula, senha, OTP ou prova. A conexão com os endpoints definitivos,
autorização AAL2 e tratamento autoritativo dos estados pertence à 2G.2.

## Acessibilidade e responsividade

Os formulários têm labels reais, associação de erro, foco no primeiro campo
inválido, navegação por teclado, foco visível e autocomplete apropriado. O campo
OTP aceita digitação, colagem, backspace e setas. A composição é mobile-first,
não bloqueia zoom, respeita safe areas e reorganiza espaçamento sem criar
rolagem horizontal entre 360 e 1440 pixels. Textos sobre dourado permanecem
escuros e a animação respeita preferência por movimento reduzido.
