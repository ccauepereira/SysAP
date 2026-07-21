# Prompt inicial para implementar o SysAP com Codex

Use este texto como primeira tarefa do Codex no repositório. Ele implementa somente a **Fase 1 — Fundação executável**. Não tente gerar o produto completo em uma única execução.

---

## Prompt para copiar

```text
Quero que você implemente a Fase 1 — Fundação executável do SysAP neste repositório.

Objetivo

Transformar o repositório de documentação em uma base monorepo executável e testada, seguindo a arquitetura aprovada. Ao final, a API, o painel web e o PostgreSQL local devem subir por comandos documentados, e a CI deve verificar os principais gates.

Contexto obrigatório

1. Leia integralmente `AGENTS.md` e `docs/architecture.md` antes de propor qualquer alteração.
2. Inspecione todo o repositório e confirme o estado atual.
3. A fonte de verdade do escopo é a seção “Fase 1 — Fundação executável” da arquitetura.
4. Leia `assets/brand/README.md` e use `assets/brand/artur-performance-logo.png` como a única logo oficial.
5. O projeto é conduzido por um estudante que precisa entender o código. Prefira soluções simples, explícitas e fáceis de explicar.

Escopo desta execução

- Criar a estrutura do monorepo descrita na arquitetura.
- Inicializar `apps/api` como módulo Go em versão estável suportada, com:
  - configuração por variáveis de ambiente;
  - encerramento gracioso;
  - logs estruturados sem segredos;
  - `GET /healthz` sem dependências externas;
  - `GET /readyz` verificando o PostgreSQL;
  - envelope de erro consistente e `request_id`;
  - testes dos endpoints e da configuração.
- Inicializar `apps/web` com Next.js App Router, TypeScript e gerenciador pnpm, com:
  - layout inicial acessível;
  - identidade visual escura baseada nos tokens oficiais da marca;
  - uso da logo original, copiada para o local público apropriado sem alteração visual;
  - página “Estado do sistema” consumindo a API;
  - estados de carregamento, indisponibilidade e sucesso;
  - lint, typecheck, testes essenciais e build.
- Preparar PostgreSQL local e `infra/supabase/migrations` com a migration mínima de fundação:
  - schema privado de negócio;
  - tabela de controle/versão necessária ao bootstrap;
  - nenhuma tabela completa das fases futuras.
- Criar `contracts/openapi/openapi.yaml` apenas com `/healthz`, `/readyz` e envelopes comuns.
- Criar `.env.example`, `.gitignore`, Makefile e/ou scripts raiz para setup, desenvolvimento, teste, lint e build.
- Criar GitHub Actions para API, web, contrato e migrations que já existirem nesta fase.
- Atualizar o README com pré-requisitos e comandos reais que você validou.
- Preparar `apps/android/README.md` explicando que o scaffold Android será feito na Fase 5; não gerar aplicativo Android agora.

Limites obrigatórios

- Não implemente autenticação, atletas, turmas, presença, prontidão, feedback, GPS, heatmap, Health Connect ou dashboard de negócio nesta execução.
- Não crie dados falsos apresentados como reais.
- Não redesenhe, recolora, distorça, recorte ou aplique efeitos à logo oficial.
- Não use texto branco sobre o dourado principal quando isso reprovar contraste; siga o guia da marca.
- Não invente formato de arquivo de colete GPS.
- Não adicione rastreamento ao vivo, IA, microsserviços, Redis, Kafka, Kubernetes ou mensageria externa.
- Não acesse Supabase remoto, não faça deploy e não solicite segredos reais.
- Não use tags `latest`; fixe versões verificadas nos arquivos de build e CI.
- Não faça commit ou push sem minha autorização explícita.
- Não troque Go, Next.js, PostgreSQL/Supabase ou a estrutura arquitetural sem parar e pedir decisão.

Forma de trabalho

1. Antes de editar, apresente:
   - resumo do que entendeu;
   - decisões pequenas que ainda precisa tomar;
   - plano de arquivos e comandos;
   - riscos ou bloqueios encontrados.
2. Se uma escolha alterar arquitetura, custo, segurança ou escopo, pare e pergunte. Para escolhas locais e reversíveis, use seu melhor julgamento e documente.
3. Implemente em passos pequenos. Após cada passo, rode a verificação relevante.
4. Quando algo falhar, investigue a causa; não remova o gate para fazer a CI ficar verde.
5. Revise o diff completo antes de finalizar.

Critérios de aceite

- Um desenvolvedor novo consegue seguir o README em ambiente limpo.
- A API inicia, responde `/healthz` e diferencia corretamente readiness sem/com banco.
- O painel web exibe o estado real da API e trata indisponibilidade.
- O PostgreSQL local aplica as migrations do zero sem intervenção manual escondida.
- O OpenAPI valida e representa os endpoints implementados.
- Nenhum segredo ou dado pessoal foi incluído no Git.
- Formatação, lint, typecheck, testes e builds relevantes passam.
- A CI usa os mesmos comandos documentados para desenvolvimento local.
- O escopo não ultrapassa a Fase 1.

Entrega final

Ao terminar, responda em português com:

1. resumo do que foi implementado;
2. árvore dos principais arquivos;
3. explicação simples do fluxo web -> API -> PostgreSQL;
4. comandos executados e resultado de cada um;
5. testes adicionados;
6. decisões e dependências escolhidas com justificativa;
7. limitações, pendências ou comandos que não conseguiu executar;
8. confirmação explícita de que revisou o diff e permaneceu na Fase 1.
```

---

## Como usar

1. Abra o repositório `SysAP` no Codex.
2. Entre em Plan mode para a primeira leitura.
3. Cole o prompt acima.
4. Revise o plano antes de autorizar a implementação.
5. Ao final, confira o diff e peça uma explicação dos trechos que você não entende.
6. Só depois de executar localmente e revisar, faça commit da Fase 1.

O próximo prompt deve ser criado apenas depois dessa fundação passar nos testes. Ele tratará a Fase 2 como um corte vertical: autenticação, organização, primeiro atleta e primeira tela real.
