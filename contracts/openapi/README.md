# Contrato OpenAPI

`openapi.yaml` e a fonte de verdade da superficie HTTP do SysAP e usa OpenAPI
3.1. `GET /healthz` e `GET /readyz` permanecem implementados como na Fase 1.
As operacoes de identidade sob `/v1` sao contratos planejados na Subfase 2A e
estao marcadas com `x-sysap-implementation-status: planned`; elas ainda nao
representam funcionalidade disponivel.

O contrato de identidade e independente de fornecedor. Clientes falam somente
com a API Go, que aplica autorizacao, isolamento por organizacao, rate limiting
e mensagens seguras. Nao existem cadastro publico, login publico por telefone,
campo de fornecedor, credencial administrativa ou detalhe de entrega externa
no contrato.

O bearer e o padrao para operacoes protegidas. Endpoints publicos declaram
`security: []` explicitamente. No canal Web, o servidor Next.js consome tokens
server-side e mantem access/refresh em cookies seguros separados; o navegador
nao chama a API Go com esses cookies. No futuro mobile, refresh fica cifrado por
adapter nativo. JWT e `sub` sao credencial opaca, nunca ID de negocio; a API
tambem confirma `session_id` e estado local. Senha, OTP, TOTP, refresh token e
ticket recebidos em corpos sao `writeOnly`; tokens e tickets emitidos sao
`readOnly` e nao possuem exemplos.

O enrollment TOTP de staff usa ticket de primeiro fator, entrega URI `otpauth`
somente em resposta `no-store` e confirma o fator pelo mesmo endpoint de
verificacao. A URI contem segredo: nao possui exemplo, log ou replay
idempotente; resultado ambiguo reinicia o fluxo e remove fator nao verificado.

Quando uma operacao sem recurso na URL precisa de tenant, como criar convite,
`X-Organization-ID` apenas seleciona contexto. A API valida membership ativa e
permissao no PostgreSQL; o header nunca concede acesso. Idempotencia compara
somente projecao nao secreta e jamais persiste senha, OTP/TOTP, token, ticket ou
derivado.

Solicitacao de recuperacao sempre responde de forma generica, e autenticacao
nao diferencia matricula inexistente, segredo incorreto ou conta indisponivel.
Todo `429` documenta `Retry-After`, e respostas de identidade usam
`Cache-Control: no-store` e `X-Request-ID`.

Na raiz do repositorio, valide o contrato com:

```sh
pnpm openapi:lint
```

O Redocly valida estrutura, schemas e exemplos. Os testes Go comparam os corpos
HTTP aprovados byte a byte, enquanto a integracao exercita API e PostgreSQL
reais. Essa separacao evita adicionar uma biblioteca YAML ao backend apenas
para duplicar a validacao semantica feita pelo linter.

Esta subfase nao gera SDK ou servidor, nao publica documentacao automaticamente
e nao implementa os endpoints planejados.
