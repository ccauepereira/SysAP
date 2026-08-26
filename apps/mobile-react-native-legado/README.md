# SysAP — Base Legada React Native / Expo (Preservada)

Este diretório contém a base móvel inicial desenvolvida em **React Native / Expo** durante as fases anteriores do projeto.

---

## 1. Motivo da Substituição pelo Flutter

Conforme deliberado e aprovado no **ADR 0004** (`docs/architecture/decisions/0004-flutter-mobile-android-ios.md`):
1. **Controle Nativo e Plataforma**: O Flutter compila diretamente para código de máquina nativo e oferece integração tipada com APIs do sistema operacional via Platform Channels (Pigeon), essencial para os futuros adaptadores de sensores esportivos (HealthKit no iOS e Health Connect no Android).
2. **Segurança de Armazenamento**: Acesso auditável e seguro ao Apple Keychain e Android Keystore sem intermediários ou pontes JavaScript transitivas.
3. **Estabilidade de Dependências**: Eliminação de vulnerabilidades recorrentes da árvore transitiva do ecossistema Node.js/npm no cliente móvel.

---

## 2. Localização e Preservação

* **Base Oficial Atual**: O cliente mobile oficial do SysAP reside exclusivamente em `apps/mobile/` (desenvolvido em Flutter/Dart).
* **Base Legada**: Todo o código-fonte, componentes visuais e testes do protótipo React Native estão preservados integralmente neste diretório (`apps/mobile-react-native-legado/`) apenas para referência histórica.

---

## 3. Próximos Passos (Fase 3.4 em Diante)

* Os fluxos de login nativo, sessão segura com tokens/refresh tokens no Keychain/Keystore e roteamento por perfil (`Owner` vs `Athlete`) serão reimplementados diretamente no Flutter a partir da Fase 3.4, consumindo os contratos oficiais da API SysAP.
