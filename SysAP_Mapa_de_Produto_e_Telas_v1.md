# SysAP — Mapa de Produto e Telas

**Versão:** 1.0  
**Data:** 26 de julho de 2026  
**Status:** direção funcional aprovada para orientar as fases futuras

---

## 1. Norte do produto

O SysAP é o sistema da Artur Performance para organizar treino específico de futebol, chamado **CrossFut**, acompanhar atletas e criar uma experiência de evolução esportiva.

Ele não será um sistema genérico de academia.

- O treino coletivo é CrossFut/futebol, realizado em turma e conduzido pelo Arthur.
- O plano individual é complementar: academia, crossfit, parque ou casa.
- O atleta vive sua evolução pelo **Modo Carreira AP**.
- O Arthur opera tudo pelo celular em uma Web App mobile-first; desktop apenas amplia o mesmo painel.

## 2. Pessoas e experiências

| Pessoa | Produto principal | Objetivo |
|---|---|---|
| Arthur / owner / trainer | Web App mobile-first | Criar turmas, conduzir sessões, fazer chamada, orientar planos e acompanhar o grupo |
| Atleta | App Android futuro | Ver o dia, confirmar presença, seguir plano, acompanhar Carreira AP e evolução |

O painel do Arthur não terá uma versão desktop separada. As mesmas rotas devem funcionar em celular, tablet e computador, começando por 360–430 px.

## 3. Turmas e grupos

Turma nunca será valor fixo no código. Arthur cria, edita, ativa e inativa livremente.

Exemplos iniciais:

```text
Artur Performance
├── AP 6:00–7:00
├── AP 7:00–8:00
└── Tropa AP
```

Cada turma/grupo terá:

- nome;
- tipo: CrossFut, futebol técnico, avaliação, grupo especial ou outro definido pelo Arthur;
- dias recorrentes;
- horário de início e término planejados;
- local;
- treinador responsável;
- atletas inscritos;
- posição no grupo quando necessário;
- status ativo/inativo;
- capacidade opcional;
- histórico preservado depois de inativada.

Um atleta pode pertencer a mais de uma turma. O perfil terá posição principal e poderá exibir posição/contexto específico dentro de uma turma.

**Tropa AP** é um grupo especial criado pelo Arthur, não uma regra fixa da aplicação. Pode ter agenda, atletas, campanhas, missões, ranking e eventos próprios.

## 4. CrossFut coletivo versus plano complementar

### Sessão coletiva CrossFut

É um treino de futebol em turma. O Arthur controla:

- agendamento;
- foco do dia;
- turma/local;
- início real;
- chamada;
- término real;
- ausências e justificativas;
- observações gerais.

Exemplo:

```text
CrossFut — Potência e agilidade
AP 7:00–8:00 · Areninha
Foco: explosão, desaceleração e mudança de direção
```

Não há obrigação de cadastrar séries e cargas individuais para cada atleta nessa sessão.

### Plano complementar individual

É uma ficha pessoal de treino que Arthur pode montar para academia, parque, casa ou crossfit.

Cada plano pode ter:

- período/semanas;
- objetivo;
- dias sugeridos;
- exercícios;
- séries, repetições, carga e descanso quando aplicável;
- instruções de execução;
- status: planejado, realizado, não realizado;
- feedback simples do atleta: concluído, difícil ou não consegui.

O sistema não fará adaptação médica automática. Arthur interpreta o feedback e ajusta o plano.

## 5. Navegação mobile-first do Arthur

```text
[ Hoje ] [ Turmas ] [ Treinos ] [ Atletas ] [ Mais ]
```

### Hoje

- próximo treino;
- treino em andamento;
- chamada rápida;
- confirmados, pendentes, ausentes e justificativas;
- alertas operacionais;
- mensalidade vencida apenas para quem possui permissão.

### Turmas

- listar turmas/grupos criados;
- criar turma;
- editar horário/local;
- adicionar/remover atleta;
- posição e status de inscrição;
- acesso à Tropa AP.

### Treinos

- calendário;
- criar sessão CrossFut;
- iniciar e encerrar;
- histórico;
- planos complementares individuais.

### Atletas

- busca;
- perfil;
- presença;
- plano complementar;
- avaliações;
- Carreira AP;
- situação de acesso, conforme papel do operador.

### Mais

- financeiro;
- campanhas e temporada;
- relatórios;
- equipe;
- configurações.

## 6. Tela operacional mais importante

```text
CrossFut — AP 7:00–8:00
Em andamento há 42 min

32 presentes
4 pendentes
2 ausentes

[ Fazer chamada ]
[ Encerrar treino ]
```

Ela deve exigir poucos toques, funcionar com uma mão e continuar legível sob sol forte na areninha.

## 7. Modo Carreira AP

O Modo Carreira deve motivar consistência, pertencimento e evolução, não dar diagnóstico médico ou expor atleta.

```text
Carreira AP
├── temporada atual
├── XP de carreira
├── nível
├── pontos de temporada
├── sequência
├── missões
├── conquistas
└── rankings autorizados
```

### Duas pontuações

| Pontuação | Uso |
|---|---|
| XP de Carreira | nível, marcos e conquistas acumuladas |
| Pontos de Temporada | ranking/campanha do período atual |

Uma temporada nova reinicia apenas os pontos de temporada; não apaga a trajetória do atleta.

### Pontos permitidos

- presença real;
- confirmação de presença;
- conclusão de plano complementar;
- sequência de semanas;
- participação em avaliação;
- missão criada pelo Arthur;
- comportamento/comprometimento registrado com critério claro.

### Nunca pontuar ou punir por

- lesão;
- prontidão baixa;
- informação médica;
- falta justificada;
- mensalidade atrasada;
- GPS, frequência cardíaca ou smartwatch de maneira automática.

Arthur define a temporada, regras, missões, visibilidade do ranking e grupos participantes. Regras de pontuação precisam ser determinísticas, versionadas e auditáveis.

## 8. Inspirações adaptadas

### Do Strava

- resumo pós-sessão;
- histórico cronológico de atividades;
- comparação do atleta com seu próprio histórico;
- desafios coletivos;
- perfil de evolução;
- mapa/heatmap pessoal somente após dados GPS reais e consentimento.

GPS não será usado para concluir gols, passes, toques ou qualidade técnica.

### Do adidas Running

- planos guiados;
- check-in em evento;
- metas e desafios;
- pontos, níveis e conquistas;
- comunidade local;
- feedback após o treino;
- recompensas de participação definidas pela organização.

Não copiar layout, marca, textos ou mecanismos de recompensa dos produtos de referência.

## 9. Telas estimadas

### Web App do Arthur

1. autenticação e ativação administrativa;
2. Hoje;
3. Turmas;
4. criar/editar turma;
5. detalhe da turma;
6. Atletas;
7. perfil do atleta;
8. agenda;
9. criar/editar sessão CrossFut;
10. treino em andamento/chamada;
11. histórico de sessões;
12. planos complementares;
13. editor de plano;
14. avaliações;
15. Carreira AP/temporadas;
16. missões e rankings;
17. financeiro/acessos;
18. equipe/configurações.

### App futuro do atleta

1. ativação;
2. login;
3. início/meu dia;
4. detalhe do treino;
5. confirmar presença;
6. plano semanal;
7. detalhe do plano/exercício;
8. evolução;
9. Carreira AP;
10. missões/ranking permitido;
11. avisos;
12. perfil/configurações.

Estados como sessão expirada, acesso bloqueado, MFA obrigatório, vazio, offline e erro não contam como telas principais, mas precisam de design próprio.

## 10. Identidade visual

- azul-marinho muito escuro como base;
- branco claro para leitura e superfícies secundárias;
- dourado como destaque de ação, conquista e marca;
- verde/amarelo/vermelho exclusivamente para estados;
- tipografia forte e objetiva;
- foco em contraste, espaço e leitura; pouco brilho e poucos gradientes;
- nenhuma alteração na logo oficial.

## 11. Roadmap funcional

| Fase | Entrega |
|---|---|
| 2C–2G | identidade, segurança, ativação, login, MFA e painel autenticado |
| 3A | turmas, atletas, inscrições, posições e agenda recorrente |
| 3B | sessões CrossFut, iniciar/encerrar e chamada |
| 3C | planos complementares individuais |
| 3D | Carreira AP, temporadas, XP, missões e ranking |
| 3E | financeiro e bloqueio manual de acesso |
| 4 | app Android do atleta |
| 5 | Health Connect, smartwatch e importação de colete |
| 6 | GPS, mapas, heatmaps e relatórios avançados |

## 12. Regras de honestidade

- prontidão e carga não são diagnóstico médico;
- GPS não mede fundamentos de futebol;
- dados de dispositivo entram após consentimento;
- heatmap e rota são privados por padrão;
- ranking é configurável e não pode punir saúde, falta justificada ou condição financeira;
- dados reais de atleta nunca entram em seed, screenshot de teste ou Git.
