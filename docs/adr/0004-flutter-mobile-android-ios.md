# ADR 0004 — Cliente Mobile Oficial em Flutter para Android e iOS

- **Status:** Aprovado como base técnica da Fase 3
- **Data:** 26 de agosto de 2026
- **Decisores:** Time de Engenharia SysAP

## Contexto

O SysAP necessita de uma aplicação móvel nativa de alto desempenho, confiabilidade e segurança para atender atletas e gestores desportivos (Owner e Athlete) em plataformas **Android e iOS**. O aplicativo precisa operar de modo offline-first para métricas locais, gerenciar credenciais de acesso sensíveis com segurança de hardware (Keystore no Android e Keychain no iOS), integrar-se a fontes de dados de saúde e sensores de relógios (Health Connect e Apple HealthKit) e consumir a API REST canônica em Go/Supabase.

## Decisão

```text
Flutter + Dart será o cliente mobile oficial do SysAP, com Android e iOS como plataformas-alvo.
```

O projeto do aplicativo móvel residirá estritamente no diretório:

```text
apps/mobile/
```

Isso consolida a arquitetura monorepo do SysAP. A totalidade da interface, regras de negócio da aplicação, domínio e comunicação com a API serão escritas em Dart. Recursos estritamente dependentes de APIs de baixo nível do sistema operacional (Health Connect no Android e HealthKit no iOS) serão encapsulados por adaptadores em Kotlin e Swift conectados via **Platform Channels com contratos tipados gerados por Pigeon, quando for necessário integrar Kotlin e Swift**.

## Motivos e Matriz de Avaliação Tecnológica Ponderada

| Critério | Peso | Flutter + Dart | React Native / Expo | Web / PWA |
| :--- | :---: | :---: | :---: | :---: |
| **Segurança e Armazenamento de Sessão** | 30% | **9,5** | **8,0** | **4,0** |
| **Suporte Consistente a Android e iOS** | 25% | **9,5** | **8,0** | **6,5** |
| **Manutenção de Longo Prazo** | 20% | **9,0** | **7,0** | **7,5** |
| **Integração com Recursos Nativos Esportivos** | 15% | **9,0** | **8,5** | **3,0** |
| **Curva de Aprendizado e Produtividade** | 10% | **8,5** | **8,5** | **9,0** |
| **Nota Final Ponderada** | **100%** | **9,23** | **7,93** | **5,68** |

### Justificativas Técnicas:
1. **Segurança (30%)**: Flutter compila para binários nativos, mas nenhum código distribuído ao dispositivo deve conter segredos, pois aplicativos compilados ainda podem ser analisados. O armazenamento de credenciais usará integrações de plataforma auditáveis, protegidas pelo Android Keystore e Apple Keychain. Tokens nunca serão gravados em armazenamento comum, logs ou variáveis públicas.
2. **Consistência Visual e Comportamento (25%)**: A renderização própria favorece interfaces fluidas e consistentes; o desempenho final dependerá do dispositivo, da tela e da complexidade da interface.
3. **Manutenção e Tipagem (20%)**: A adoção de Flutter reduz a dependência do ecossistema Node.js no cliente mobile, mas não elimina riscos de dependências: pacotes Dart/Flutter e código nativo também exigem auditoria e atualização contínuas.
4. **Recursos Esportivos e Smartwatches (15%)**: Health Connect e HealthKit não significam conexão direta com qualquer smartwatch. Eles serão a camada principal para ler dados autorizados que relógios e apps sincronizam; integrações Bluetooth ou SDK proprietário só entram depois, se algum relógio realmente exigir.
5. **Web/PWA Rejeitado**: Não atende de forma adequada ao modelo necessário de integração com HealthKit, sensores em segundo plano, armazenamento seguro de sessão e experiência nativa consistente nas duas plataformas (nota final ponderada 5,68).

## Limitação Real

> [!WARNING]
> O desenvolvimento Android pode ocorrer no ambiente Linux atual.
> Build, assinatura e validação nativa final de iOS exigirão macOS com Xcode ou uma infraestrutura de CI compatível com macOS em etapa futura.

## Consequências

### Positivas
* Código de interface, casos de uso e cliente HTTP unificados para Android e iOS no monorepo (`apps/mobile/`).
* Redução de dependências JavaScript no cliente móvel.
* Performance e renderização fluidas para dashboards de métricas esportivas.
* Separação clara entre domínio puro em Dart e infraestrutura nativa em Kotlin/Swift com Pigeon.

### Negativas / Mitigações
* Exigência de ambiente macOS para compilação final e testes nativos de iOS (mitigado por pipeline de CI com runners macOS).
* Necessidade de manter contratos e testes de integração tipados com Pigeon para Platform Channels nativos.
