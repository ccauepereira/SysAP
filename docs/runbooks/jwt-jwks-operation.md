# Runbook: Operação Segura de JWT / JWKS (Subfase 2C.5)

## 1. Propósito e Arquitetura

O SysAP valida tokens JWT de forma estritamente local na API Go, utilizando o conjunto de chaves públicas obtido via JWKS (*JSON Web Key Set*) do provedor de identidade (Supabase Auth).

A validação criptográfica do JWT na API é a primeira camada de defesa de autenticação. A segunda camada é a resolução e conferência da sessão local e do perfil no banco de dados PostgreSQL (`app.auth_sessions` e `app.organization_memberships`).

> [!IMPORTANT]
> A revogação ou suspensão local de sessão é **sempre conferida no PostgreSQL**, mesmo que o JWT apresentado pela requisição ainda seja válido criptograficamente e não tenha atingido seu tempo de expiração (`exp`).

---

## 2. Validações Obrigatórias do Token

Para qualquer requisição recebida com cabeçalho `Authorization: Bearer <token>`, o verificador de tokens exige e valida:

1. **Estrutura e Formato**: Cabeçalho de proteção e formato compacto de JWT de 3 partes base64url, dentro do limite máximo de tamanho.
2. **Algoritmo Permitido**: Exclusivamente **`ES256`** (ECDSA utilizando curva P-256 e SHA-256). É sumariamente rejeitado qualquer token que especifique `alg: none`, `HS256`, `RS256` ou qualquer outro algoritmo fora da allowlist.
3. **Identificador de Chave (`kid`)**: O token deve conter um `kid` válido que corresponda a uma chave pública ativa do conjunto JWKS.
4. **Emissor (`iss`)**: Deve ser idêntico ao configurado na variável de ambiente `SYSAP_AUTH_JWT_ISSUER`.
5. **Audiência (`aud`)**: Deve conter o valor configurado em `SYSAP_AUTH_JWT_AUDIENCE` (ex: `authenticated`).
6. **Expiração (`exp`)**: O instante atual da validação deve ser estritamente anterior à data/hora de expiração do token.
7. **Not Before (`nbf`)**: Se presente, o instante atual deve ser posterior ou igual ao campo `nbf`.
8. **Identificador de Perfil (`sub`)**: Deve ser um UUID V4 sintaticamente válido e não nulo.
9. **Sessão Local (`session_id`)**: Claim customizada obrigatória, contendo um UUID V4 válido de sessão.
10. **Nível de Garantia (`aal`)**: Se presente, deve ser `aal1` ou `aal2`. Valores ausentes são normalizados para `aal1`.
11. **Papel de Sistema (`role`)**: Se a claim `role` estiver presente no token, seu valor deve ser obrigatoriamente `authenticated`. Claims não alteram nem concedem permissões de negócio na API.

---

## 3. Comportamento do Cache e Rotação de Chaves

O verificador gerencia o conjunto de chaves JWKS em memória via um mecanismo seguro de cache (`jwksCache`), desenhado para alta concorrência e resiliência:

* **Cache Hit**: Se a chave pública correspondente ao `kid` for encontrada no cache e o tempo de vida do cache (`SYSAP_AUTH_JWKS_CACHE_TTL`) estiver válido, a busca é efetuada sem nenhuma chamada HTTP externa.
* **KID Desconhecido**: Quando um token apresenta um `kid` não localizado no conjunto em memória, é disparado no máximo um recarregamento (*refresh*) síncrono e controlado da URL de JWKS.
* **Refresh Único sob Concorrência**: Múltiplas requisições simultâneas requisitando atualização do JWKS usam o mecanismo de coalescência (`singleflight`) para garantir a execução de no máximo 1 requisição HTTP remota ao provedor de JWKS por vez.
* **Proteção contra Tempestades de Rede (DoS)**: Para evitar tempestades de requisições disparadas por clientes maliciosos enviando `kid`s fictícios continuamente, o cache aplica um intervalo mínimo (rate-limit de 5 segundos) entre buscas ativas no provedor remoto.
* **Expiração e Política *Fail-Closed***: Se o cache expirar e o provedor remoto de JWKS estiver inacessível, retornando erro HTTP (!= 200), resposta vazia, corpo corrompido, redirect ou *timeout*, a autenticação falha imediatamente fechada (*fail-closed*), recusando o acesso com o status HTTP 401 seguro (`ErrTokenVerificationUnavailable` / `ErrInvalidToken`).

---

## 4. Classificação de Dados e Segredos

* **Público**: O documento JWKS (conjunto de chaves públicas) obtido da URL configurada.
* **Segredo Crítico**: Tokens JWT emitidos, cabeçalhos `Authorization`, credenciais, chaves privadas de assinatura e tokens de serviço (como `service_role` do Supabase). **Nunca** devem ser commitados no Git, expostos em logs ou incluídos em exemplos de documentação.

---

## 5. Variáveis de Ambiente

As seguintes variáveis configuram o comportamento do verificador de tokens. Nomes de ambiente padrão:

| Variável de Ambiente | Descrição | Exemplo de Placeholder |
| :--- | :--- | :--- |
| `SYSAP_AUTH_JWT_ISSUER` | Emissor esperado nas claims do token (`iss`) | `https://auth.example.local/auth/v1` |
| `SYSAP_AUTH_JWT_AUDIENCE` | Audiência esperada nas claims do token (`aud`) | `authenticated` |
| `SYSAP_AUTH_JWKS_URL` | URL HTTP(S) absoluta para o endpoint de chaves públicas | `https://auth.example.local/auth/v1/.well-known/jwks.json` |
| `SYSAP_AUTH_JWKS_TIMEOUT` | Timeout para requisição HTTP de busca do JWKS | `2s` |
| `SYSAP_AUTH_JWKS_CACHE_TTL` | Tempo de vida (TTL) do cache de chaves JWKS em memória | `5m` |
| `SYSAP_AUTH_JWKS_MAX_BODY_LENGTH` | Limite máximo de bytes lidos do corpo da resposta JWKS | `65536` |
| `SYSAP_AUTH_JWKS_MAX_KEYS` | Limite máximo de chaves públicas aceitas no JWKS | `16` |

> [!NOTE]
> Em ambiente não local (`SYSAP_ENV` diferente de `development`, `local` ou `test`), a `SYSAP_AUTH_JWKS_URL` exige obrigatoriamente o esquema HTTPS.

---

## 6. Validação Local

A validação local deve ser executada sem dependência de serviços externos ou de rede remota:

1. **Iniciar banco de dados local**:
   ```bash
   pnpm db:start
   ```
2. **Executar testes de unidade do verificador de autenticação**:
   ```bash
   cd apps/api
   go test -v ./internal/platform/auth
   ```
3. **Executar testes de integração contendo a validação completa de `/v1/me`**:
   ```bash
   pnpm test:integration
   ```

---

## 7. Checklist de Rotação Planejada de Chaves

Ao realizar rotação de chaves de assinatura no provedor de autenticação (Supabase Auth):

1. [ ] **Publicação**: Adicionar a nova chave pública ao JWKS no provedor mantendo a chave antiga ativa (ambos os `kid`s presentes no JSON do JWKS).
2. [ ] **Verificação de Capacidade**: Garantir que o número total de chaves ativas no JWKS não excede `SYSAP_AUTH_JWKS_MAX_KEYS`.
3. [ ] **Transição de Emissão**: Configurar o provedor de autenticação para assinar novos JWTs utilizando a nova chave (`kid`).
4. [ ] **Janela de Coexistência**: Aguardar o tempo decorrido do TTL do cache (`SYSAP_AUTH_JWKS_CACHE_TTL`) mais o tempo máximo de expiração dos JWTs antigos emitidos.
5. [ ] **Limpeza**: Remover a chave antiga do JWKS no provedor após o encerramento da janela de coexistência.

---

## 8. Procedimentos de Incidente

### 8.1. Chave Privada Comprometida no Provedor
1. Promover rotação imediata de chaves no provedor de autenticação.
2. Revogar todas as sessões ativas no PostgreSQL (`app.auth_sessions`). A verificação local de sessão invalidará imediatamente o acesso, mesmo que tokens com a chave antiga/nova sejam apresentados.
3. Forçar o recarregamento do JWKS na API (reiniciando a aplicação ou aguardando o TTL).

### 8.2. JWT Suspeito ou Aumento Anormal de Erros 401
1. Verificar os logs estruturados da API Go para identificar se a recusa ocorre por assinatura inválida, claim divergente ou revogação de sessão no PostgreSQL.
2. Confirmar que nenhuma credencial ou token foi vazado em logs ou tráfego.
3. Se houver falha de comunicação persistente com a URL do JWKS, verificar conectividade da rede local/DNS ou disponibilidade da porta do provedor Auth.

---

## 9. Diretrizes de Logging e Segurança

Para prevenir vazamento de dados sensíveis ou informações de identificação pessoal (PII):

> [!CAUTION]
> É estritamente **PROIBIDO** registrar em logs, mensagens de erro, respostas HTTP ou saídas de debug:
> - O cabeçalho `Authorization` ou qualquer Bearer token / JWT na íntegra ou parcial;
> - Refresh tokens, senhas, OTPs, códigos MFA ou segredos TOTP;
> - Strings de conexão de banco de dados (DSN) contendo senhas;
> - O corpo bruto do documento JWKS ou requisições HTTP autenticadas.

Apenas erros genéricos e sanitizados (como `ErrInvalidToken` ou `ErrTokenVerificationUnavailable`) devem ser retornados aos clientes HTTP na API (mapeados para o envelope padrão HTTP 401).

---

## 10. Limitações Conhecidas do Escopo Atual

O escopo da subfase 2C.5 limita-se ao hardening e operação do verificador de tokens JWT/JWKS local. Não fazem parte deste escopo e não devem ser implementados nesta camada:

- Fluxos de login, cadastro, troca ou recuperação de senha;
- Emissão ou renovação de *refresh tokens*;
- Gerenciamento de cookies de sessão / suporte a BFF;
- Desafios de MFA, envio ou validação de SMS / OTP;
- Aplicação de *rate limiting* por IP/cliente na API HTTP.

---

## 11. Comandos de Verificação

Para validar a integridade da documentação, contratos e suíte de testes do repositório:

```bash
# Validação de contratos, tipos, lints e políticas do repositório
pnpm check

# Varredura de segredos
pnpm security:secrets

# Execução da suíte completa de testes da API
cd apps/api
go test -race ./...
```
