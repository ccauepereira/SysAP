# Laya PR gate experimental

`review_diff.py` gera perguntas e sinais de revisão para um diff Git local.
Ele é deliberadamente **advisory**: não chama rede, não usa modelo externo, não
lê arquivos de ambiente e sempre retorna código zero. Não substitui revisão
humana, scanner de segredos, testes ou checks obrigatórios.

## Uso

```bash
python3 tools/laya_pr_gate/review_diff.py
python3 tools/laya_pr_gate/review_diff.py --base origin/main
```

Sem `--base`, analisa alterações não commitadas contra `HEAD`. Com `--base`,
analisa `base...HEAD`, como em um pull request.

Arquivos `.env` e `.env.*` (inclusive exemplos), diretórios de segredos e
credenciais, material de chave e links simbólicos são excluídos antes da leitura
de conteúdo. O relatório contém somente perguntas e mensagens fixas, nunca
trechos do diff. Cada sinal examina apenas os arquivos dos seus caminhos.

## Regras

`questions.json` é a fonte versionada das perguntas contextuais e dos sinais
heurísticos. Sinais não são achados de segurança nem bloqueiam merge; servem
para orientar revisão humana. Não coloque exemplos reais, segredos ou padrões
que imprimam conteúdo sensível no relatório.
