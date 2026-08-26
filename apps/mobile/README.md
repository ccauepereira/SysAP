# SysAP — Aplicativo Mobile

Aplicativo nativo Expo / React Native do SysAP para Android e iOS.

---

## 1. Configuração de Ambiente (`EXPO_PUBLIC_API_BASE_URL`)

O aplicativo consome exclusivamente a API REST do SysAP via HTTP/HTTPS.

Para definir o endereço da API, configure a variável de ambiente:

```bash
# Exemplo para emulador Android padrão (conecta ao loopback da máquina host):
export EXPO_PUBLIC_API_BASE_URL="http://10.0.2.2:8080"

# Exemplo para emulador iOS / Web:
export EXPO_PUBLIC_API_BASE_URL="http://127.0.0.1:8080"

# Exemplo para aparelho físico na mesma rede Wi-Fi local (RFC 1918):
export EXPO_PUBLIC_API_BASE_URL="http://192.168.1.150:8080"

# Exemplo em ambiente de produção (obrigatório HTTPS):
export EXPO_PUBLIC_API_BASE_URL="https://api.sysap.app"
```

> ⚠️ **Aviso Importante sobre Variáveis Públicas**:
> `EXPO_PUBLIC_API_BASE_URL` é compilada publicamente no bundle do cliente.
> **Nunca coloque chave Brevo, service_role, senha, OTP, token de servidor ou qualquer segredo em variáveis `EXPO_PUBLIC_*`**.

---

## 2. Como Iniciar o App no Android / Emulador

### Pré-requisitos
1. Iniciar a API SysAP localmente:
   ```bash
   pnpm dev:api
   ```
2. Garantir que o emulador Android esteja em execução (`adb devices`).

### Executando o aplicativo
```bash
# Na raiz do monorepo:
pnpm --filter @sysap/mobile dev

# Ou para abrir diretamente no Android:
pnpm --filter @sysap/mobile android
```

---

## 3. Como Usar em Aparelho Físico em Rede Local

1. Conecte o computador e o smartphone na **mesma rede Wi-Fi local**.
2. Descubra o IP local da sua máquina (ex.: `192.168.1.150`).
3. Inicie a API escutando no endereço local.
4. Defina a variável antes de iniciar o Expo:
   ```bash
   export EXPO_PUBLIC_API_BASE_URL="http://192.168.1.150:8080"
   pnpm --filter @sysap/mobile dev
   ```
5. Abra o aplicativo **Expo Go** no smartphone Android e leia o QR Code gerado no terminal.

---

## 4. Como Rodar os Testes

```bash
# Executar suíte de testes unitários e de integração do mobile:
pnpm --filter @sysap/mobile test

# Executar checagem de tipos TypeScript:
pnpm --filter @sysap/mobile typecheck
```

---

## 5. Fluxo Já Entregue (Fase 2.5.2)

```text
Usuário abre o app
→ entra com matrícula e senha
→ app chama a API real existente (/v1/auth/login)
→ sessão é armazenada com segurança no dispositivo (expo-secure-store)
→ app confirma perfil e papel na API (/v1/me)
→ Owner vai para a área (owner)
→ Athlete vai para a área (athlete)
→ logout encerra a sessão local e remota (/v1/auth/logout)
```

---

## 6. Fluxos Não Entregues Nesta Fase (Previstos para Fase 3)

- Cadastro de atletas e envio de convites (Fase 3.1)
- Ativação de conta por desafio seguro OTP (Fase 3.2)
- Confirmação de e-mail e criação de senha (Fase 3.3)
- Recuperação de senha por desafio seguro (Fase 3.4)
- Integrações transacionais de e-mail/SMS (Brevo/Twilio via Supabase Auth)
- Painel esportivo e métricas de desempenho

---

## 7. Diretrizes de Segurança

- **Credenciais**: Senhas nunca são persistidas em disco ou memória de longa duração.
- **Sessão**: O par `access_token` e `refresh_token` é armazenado exclusivamente no `expo-secure-store` (Android Keystore / iOS Keychain).
- **Sem WebView / Sem AsyncStorage**: Nenhum token ou dado sensível é mantido em armazenamento inseguro.
- **Autorização**: O cliente mobile nunca toma decisões de permissão baseado apenas no estado local; a API Go é a fonte de verdade autoritativa via `GET /v1/me`.
