# SysAP Mobile — Fundação Flutter (Android e iOS)

Aplicativo mobile oficial do SysAP desenvolvido em **Flutter + Dart**, projetado para alta performance e suporte nativo a **Android** e **iOS**.

---

## 1. Escopo Desta Fase (Fase 3.3)

Esta entrega compreende a **fundação técnica** do aplicativo:
* Projeto Flutter configurado para Android e iOS;
* Arquitetura em 4 camadas desacopladas (`apresentação`, `aplicação`, `domínio`, `infraestrutura`);
* Configuração pública segura por `--dart-define` com validação de HTTPS e exceção restrita a desenvolvimento local;
* Tema Material 3 com suporte a modo claro e escuro;
* Ponto de entrada limpo e tela de inicialização (bootstrap) acessível;
* Testes unitários, de widget e de integração.

> [!NOTE]
> **Funcionalidades deliberadamente agendadas para as próximas fases:**
> * Autenticação, sessão segura em Keystore/Keychain e rotas por perfil: **Fase 3.4**;
> * Sensores, Health Connect, HealthKit, biometria e relógios: **Fase 3.5**;
> * Validação nativa final e empacotamento: **Fase 3.6**.

---

## 2. Pré-requisitos

* **Flutter SDK**: `>= 3.13.1` (canal `stable`)
* **Dart SDK**: `>= 3.0.0`
* **Android**: Android Studio e Android SDK configurados para desenvolvimento Android.
* **iOS**: macOS com Xcode configurado para desenvolvimento e compilação de iOS.

---

## 3. Comandos de Desenvolvimento e Validação

Dentro do diretório `apps/mobile/`:

```bash
# Obter dependências
flutter pub get

# Formatar código
dart format --set-exit-if-changed .

# Análise estática de código
flutter analyze

# Executar suíte de testes unitários e de widget
flutter test
```

---

## 4. Configuração Segura por Ambiente (`--dart-define`)

O aplicativo recebe exclusivamente parâmetros de configuração **públicos** e não sensíveis em tempo de compilação ou execução via `--dart-define`:

### Execução em Desenvolvimento Local (Android Emulator)
```bash
flutter run -d android \
  --dart-define=SYSAP_API_URL=http://10.0.2.2:8080 \
  --dart-define=SYSAP_ENV=desenvolvimento
```

### Execução em Produção
```bash
flutter run -d android \
  --dart-define=SYSAP_API_URL=https://api.sysap.com.br \
  --dart-define=SYSAP_ENV=producao
```

> [!WARNING]
> **Aviso de Segurança:**
> 1. Valores passados por `--dart-define` ficam embutidos no binário compilado. **Nunca passe senhas, chaves privadas, tokens ou segredos via `--dart-define`**.
> 2. O ambiente de produção **exige obrigatoriamente** o protocolo seguro `HTTPS`. Conexões `HTTP` são rejeitadas pelo validador do aplicativo.
> 3. Conexões `HTTP` são toleradas estritamente em ambiente de desenvolvimento local (`desenvolvimento`).

---

## 5. Estrutura Arquitetural

```text
apps/mobile/
  android/          # Projeto nativo Android
  ios/              # Projeto nativo iOS
  lib/
    main.dart       # Ponto de entrada mínimo
    aplicacao/      # Casos de uso de bootstrap e configuração
      configuracao/
      inicializacao/
    dominio/        # Tipos puros de falha e contratos sem dependência de framework
      falhas/
    infraestrutura/ # Adaptadores técnicos desacoplados
      plataforma/
    apresentacao/   # Telas, temas e componentes acessíveis
      componentes/
      inicializacao/
      navegacao/
      tema/
  test/             # Testes unitários e de widget
  integration_test/ # Testes de integração de fluxo
  pubspec.yaml      # Manifesto Flutter
  README.md
```

---

## 6. Limitações de Plataforma e Ambiente

* **Android**: Pode ser compilado e executado em Linux/macOS/Windows com o Android SDK instalado.
* **iOS**: A compilação nativa, assinatura de código e validação de iOS **dependem obrigatoriamente de macOS com Xcode**. Em ambientes Linux, o código Dart/Flutter de iOS é mantido e validado estaticamente, mas o build nativo é executado exclusivamente em macOS/CI macOS.
