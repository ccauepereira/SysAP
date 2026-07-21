# PostgreSQL local do SysAP

Esta pasta contém somente a fundação local do PostgreSQL gerenciado pelo
Supabase CLI. Ela não possui vínculo com projeto remoto e não contém
credenciais.

## Uso local

Na raiz do repositório:

```sh
pnpm db:start
pnpm db:env
pnpm db:reset
pnpm db:lint
pnpm db:status
pnpm db:stop
```

Os scripts ocultam a saída sensível da CLI. `db:env` trata a URL recebida como
entrada não confiável e valida protocolo, host de loopback, porta, credenciais,
banco e caracteres proibidos com o parser de URL da biblioteca padrão do
Node.js. O comando grava exclusivamente as duas variáveis de conexão
necessárias em `.env.local`, usando sintaxe dotenv sem expansão shell. O arquivo
é ignorado pelo Git, não pode ser um link simbólico, é preparado no mesmo
diretório com permissão `0600` e substituído atomicamente somente após todas as
validações.

`db:start` cria, quando necessário, a rede Docker dedicada `sysap-loopback` e
inicia a stack nessa rede. A rede é identificada pelas labels versionadas
`com.sysap.managed=true`, `com.sysap.purpose=local-supabase` e
`com.sysap.version=1`. Antes do uso, o script confirma nome, driver `bridge`,
labels, bind padrão em `127.0.0.1` e quantidade de containers. Uma rede
incompatível não é usada nem removida. A única migração automática admitida é
a da rede legada sem labels, com configuração local exata e nenhum container
conectado. A configuração da bridge mantém as portas publicadas em loopback sem
alterar o daemon, o firewall ou as permissões do host. O mesmo identificador de
rede é passado ao `db:reset`, pois esse comando recria o container PostgreSQL.

O schema `app` não é exposto pela Data API. A conexão local da API assume o
papel `sysap_api`, com `SET ROLE`, sem login e com acesso somente de leitura à
metadata de bootstrap. `NOLOGIN` impede que esse papel de autorização seja
usado como credencial direta; a migration reaplica incondicionalmente seus
atributos restritivos e remove memberships herdadas.

O PostgreSQL só permite remover o `EXECUTE` implícito de `PUBLIC` em funções
futuras no default global do papel executor; por isso esse `REVOKE` se aplica a
todas as funções que `postgres` criar depois da migration. Tabelas e sequências
continuam com defaults restritos no schema `app`.

Como a migration ainda não foi implantada, o rollback local consiste em
remover `app.bootstrap_metadata`, remover o schema `app` e revogar o grant do
papel para `postgres`. `sysap_api` só deve ser removido se tiver sido criado por
essa migration e não possuir outras dependências; se já existia, deve ser
preservado e tratado explicitamente.

`SET ROLE` é uma defesa adicional, não uma credencial de produção. Um ambiente
de produção ainda exigirá um usuário `LOGIN` dedicado e restrito, criado fora
do Git, com senha armazenada em um gerenciador de segredos. Esta fundação não
está pronta para produção.
