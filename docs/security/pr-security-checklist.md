# Checklist de segurança para PR

## Contexto e escopo

- [ ] A alteração tem objetivo e limites explícitos.
- [ ] Não introduz feature posterior nem altera contrato sem atualização
  correspondente.
- [ ] Não há alterações inesperadas em lockfile, workflow, migration ou
  configuração global.

## Segredos, PII e logs

- [ ] `.env.local` e valores reais não foram lidos, copiados ou alterados.
- [ ] Não há senha, token, refresh token, OTP, proof, segredo TOTP, e-mail,
  telefone, matrícula, IP bruto, DSN ou URL autenticada em código, testes,
  exemplos, logs ou respostas.
- [ ] Novas variáveis possuem placeholders vazios somente em `.env.example`.

## Autorização e persistência

- [ ] A API continua sendo a fronteira entre clientes e dados/Auth.
- [ ] Perfil, organização, membership e sessão são validados no estado local.
- [ ] Cross-organization, perfil suspenso e sessão revogada foram considerados.
- [ ] Migration é forward-only, com RLS e grants mínimos; não há policy ampla.

## Cliente e integrações

- [ ] Browser/mobile não armazena token em armazenamento comum ou log.
- [ ] Integração externa permanece atrás de adaptador e não ganhou fallback
  silencioso.
- [ ] Dados de saúde/localização recebem minimização e consentimento explícito.

## Evidência

- [ ] Gates relevantes passaram ou há limitação documentada com comando e erro.
- [ ] Laya advisory foi lido quando o diff toca superfície sensível.
- [ ] Revisão humana confirmou os riscos relevantes antes do merge.
