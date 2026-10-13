# Convite: persistência e recuperação de entrega

## Estados

O convite, perfil e membership `pending_activation`, vínculo à turma, chave de idempotência e auditoria são gravados antes do envio de matrícula. O envio externo é síncrono após o commit local; a criação não é revertida por falha do provider.

`delivery_status` é independente do status de ativação:

- `sending`: tentativa foi reservada antes da chamada externa; em leitura posterior é tratada como `unknown` para impedir retry cego;
- `sent`: o provider confirmou a aceitação da requisição, não a entrega na caixa de entrada;
- `failed`: o adaptador classificou uma rejeição definitiva ou o canal não é suportado;
- `unknown`: timeout, falha de transporte ou resposta ambígua; verificar com a equipe/provider antes de qualquer nova tentativa.

Não existe garantia “exactly once”: a API não controla o processamento final da Brevo. Repetir uma criação com a mesma `Idempotency-Key` devolve o convite já persistido e não chama o provider novamente. A Web conserva a chave enquanto uma resposta estiver incerta.

## Reenvio autorizado

Somente owner autenticado com AAL2 e membership ativo na organização do convite pode consultar estado ou reenviar. A consulta não expõe contatos nem matrícula. Reenvio é permitido apenas quando a tentativa anterior terminou como `failed`, convite continua pendente e não expirou. Há intervalo mínimo de 60 segundos e máximo de três reenvios por janela de 24 horas. Reserva e limites são transacionais sob lock do convite; cada tentativa é auditada sem contato, matrícula ou erro de provider. `sent`, `sending` e `unknown` não permitem novo envio.

Um registro que permaneça `sending` após crash/processo interrompido não é reprocessado automaticamente. A equipe deve reconciliar o resultado do provider por procedimento operacional; não editar o banco nem transformar status manualmente.

## Migração e compatibilidade

A migration é forward-only. Convites pendentes existentes são preservados, recebem estado inicial seguro e ganham profile/membership pendentes ausentes. Convites não são substituídos nem têm expiração ampliada; continuam usando `expires_at` original. A conclusão da ativação promove os registros pendentes existentes, com suporte às linhas antigas sem profile/membership.

## Ativação assistida

Ativação presencial via QR/desafio não está implementada: não existe endpoint autorizado que gere desafio de uso único vinculado ao convite nem fluxo no dispositivo do responsável para consumir o desafio e definir senha. Os proofs de ativação existentes pertencem ao fluxo iniciado pelo atleta e não são reaproveitáveis para esse caso. Portanto a alternativa presencial permanece fora de escopo; não compartilhar matrícula/OTP com Artur, definir senha pelo administrador ou alterar status diretamente.

## Homologação

Os testes usam provider fake ou `httptest`; nenhum teste deve chamar a Brevo real. O canal implementado nesta integração é e-mail; SMS permanece desabilitado até existir adaptador server-side próprio. Credenciais de homologação ficam fora do Git e contatos fictícios/`example.test` são usados nos testes.
