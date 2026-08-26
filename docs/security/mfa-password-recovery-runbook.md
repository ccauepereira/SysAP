# MFA e recuperação de senha — operação

## MFA de staff

Owner e trainer usam TOTP mantido pelo Supabase Auth. O SysAP guarda somente o
identificador opaco do fator e seu estado; segredo TOTP, URI/QR, challenge,
código, access token e refresh token nunca são persistidos ou registrados.

Uma ação administrativa exige sessão AAL2 e fator local verificado. A consulta
de estado MFA reconcilia a remoção do fator no provider e desabilita o estado
local antes de permitir nova elevação. Em indisponibilidade do provider, a API
nega a elevação e não cria um caminho alternativo.

## Recuperação de senha

`start` devolve sempre `202` neutro. A API resolve internamente uma identidade
e entrega o pedido ao Supabase Auth sem expor matrícula, canal ou resultado de
entrega. O SysAP não persiste OTP: após verificação pelo provider, persiste
somente HMAC de uma proof aleatória, curta e de uso único.

Na conclusão, a senha passa apenas ao adaptador do Supabase Auth; depois do
sucesso, a função SQL restrita consome a proof e revoga todas as sessões locais
do perfil. Falha do provider não é contornada nem transforma a proof em acesso.

## Recuperação administrativa fora de escopo

Owner ou trainer com MFA ativo não pode concluir recuperação somente com a
proof de recuperação. A API responde `403 mfa_required`. A recuperação
administrativa é assistida e manual, com confirmação operacional de identidade
e reinicialização segura do fator pelo processo do Supabase Auth; não há bypass
local nesta subfase.
