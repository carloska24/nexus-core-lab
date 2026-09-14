# Executar frontend e backend

Na pasta `web/reference-version`, execute:

```bat
npm run dev
```

Requisitos: Node/npm, Go e dependências frontend já instaladas.

O comando compila a API em uma pasta temporária, inicia o backend, aguarda `/health` e inicia o Vite com o proxy apontando para essa API. Abra o endereço informado pelo Vite. Ctrl+C encerra ambos.

- `PORT`: porta da API, padrão 8080. Porta ocupada gera mensagem; nenhum processo externo é encerrado.
- `DATABASE_URL`: se definida, é preservada. Sem ela, armazenamento em memória; dados desaparecem ao encerrar a API.
- `NEXUS_API_TARGET`: no comando combinado é definido automaticamente para a API iniciada.
- `npm run dev -- --port 5177`: escolhe a porta frontend.
- `npm run dev:frontend`: inicia somente Vite, para quem já executa backend separadamente; nesse caso, respeita NEXUS_API_TARGET.

Nenhuma migration ou serviço PostgreSQL é iniciado automaticamente.

Validação: execução combinada com API 18085 e Vite 5185; HTTP 200 na página, `/health` via proxy e `/api/v1/subscribers`; porta ocupada rejeitada; Ctrl+C encerrou ambas as portas. Nenhuma alteração em Go ou no contrato do Gate 4A.
