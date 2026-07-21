# Contrato OpenAPI

`openapi.yaml` e a fonte de verdade da superficie HTTP implementada na Fase 1.
Ele usa OpenAPI 3.1 e documenta somente `GET /healthz` e `GET /readyz`.

Na raiz do repositorio, valide o contrato com:

```sh
pnpm openapi:lint
```

O Redocly valida estrutura, schemas e exemplos. Os testes Go comparam os corpos
HTTP aprovados byte a byte, enquanto a integracao exercita API e PostgreSQL
reais. Essa separacao evita adicionar uma biblioteca YAML ao backend apenas
para duplicar a validacao semantica feita pelo linter.

Esta fase nao publica documentacao automaticamente e nao descreve endpoints
futuros.
