# SysAP

Plataforma de acompanhamento de atletas do Artur Performance.

O SysAP reúne gestão de atletas, presença, prontidão, feedback do treinador e dados de treino vindos de smartwatch e colete GPS. O objetivo do MVP é transformar dados reais em histórico, métricas, caminho percorrido e mapa de calor, sem prometer recursos que os dispositivos não conseguem medir.

## Estado atual

Projeto em fase de arquitetura e preparação do repositório.

## Documentos principais

- [Arquitetura do sistema](docs/architecture.md)
- [Prompt inicial de implementação para o Codex](docs/codex/IMPLEMENTATION_PROMPT.md)
- [Instruções permanentes para agentes](AGENTS.md)

## Stack aprovada para o MVP

- API: Go, monólito modular e REST/OpenAPI.
- Painel do treinador: Next.js com TypeScript.
- Aplicativo do atleta: Android nativo com Kotlin e Jetpack Compose.
- Dados e identidade: PostgreSQL, Supabase Auth e Supabase Storage.
- Smartwatch: integração indireta por Health Connect.
- Colete GPS: importação de arquivo exportado pelo fabricante.

## Regra de produto

O MVP não terá GPS em tempo real, diagnóstico médico, previsão de lesão, xG, detecção automática de toques na bola ou integração inventada com um fabricante. Cada nova integração de colete só começa depois da obtenção de um arquivo real e de sua documentação.

