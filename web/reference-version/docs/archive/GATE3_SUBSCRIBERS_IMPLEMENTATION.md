# Gate 3 — Subscribers

**Data:** 11/09/2026. **Estado:** implementação e validação concluídas; PARAR PARA HUMAN REVIEW. Sem commit/push.

Foi integrado somente Subscriber na versão isolada `web/reference-version/`. Health/Telemetry do Gate 2 continuam com seus pollings e apresentação de estados. Devices, Sessions, Recent Events, Network, IPPool e Simulator não foram integrados ou alterados neste Gate.

## 1. Arquivos criados/modificados

Criados:

- [src/subscriber-api.ts](src/subscriber-api.ts): contrato e funções HTTP de Subscriber.
- [src/subscribers.tsx](src/subscribers.tsx): coleção compartilhada entre KPI e listagem.
- [src/SubscribersPage.tsx](src/SubscribersPage.tsx): lista, busca, detalhes, provisionamento e lifecycle.
- [src/subscribers.css](src/subscribers.css): estilos restritos à tela e seu diálogo.
- [tests/gate3.cjs](tests/gate3.cjs): teste de navegador usando API Go isolada.
- Evidências em [evidence/gate3](evidence/gate3) e este relatório.

Modificados:

- [src/App.tsx](src/App.tsx): provider de Subscribers, KPI real, roteamento para a tela real e retirada da composição mock de Subscribers baseada em sessões.
- [src/api.ts](src/api.ts): reutilização da leitura/validação de JSON e tratamento de erros existentes, aceitando opções HTTP para POST. Os contratos e comportamento das leituras de Health/Telemetry foram preservados.
- [vite.config.ts](vite.config.ts): proxy adicional somente para `/api/v1/subscribers` e seus caminhos.
- `dist/`: saída gerada por `npm run build`.

Não houve mudança em package.json, dependências, mapa, ReferenceArt, CSS base, código Go, migrações ou configuração PostgreSQL. A versão original fora de reference-version foi preservada.

## 2. Contratos utilizados

```ts
type SubscriberStatus =
  | 'PENDING_ACTIVATION'
  | 'ACTIVE'
  | 'SUSPENDED'
  | 'DEACTIVATED';

interface Subscriber {
  id: string;
  imsi: string;
  msisdn: string;
  status: SubscriberStatus;
  suspension_reason?: string;
  deactivation_reason?: string;
  created_at: string;
  updated_at: string;
}

interface ProvisionSubscriberRequest {
  imsi: string;
  msisdn: string;
}
```

Não foram acrescentados estados ou atributos de device, sessão, célula ou IP. As entidades são validadas em runtime antes de serem aceitas. UUID/IMSI/MSISDN permanecem strings completas. Datas permanecem strings do servidor nos dados; a formatação é apenas de apresentação.

## 3. Funções HTTP

| Função | Método/path | Resposta |
|---|---|---|
| getSubscribers | GET `/api/v1/subscribers` | Coleção completa; somente `null` bem-sucedido e validado vira `[]` |
| getSubscriber | GET `/api/v1/subscribers/{id}` | Subscriber único |
| getSubscriberByIMSI | GET `/api/v1/subscribers?imsi=...` | Subscriber único, não array |
| provisionSubscriber | POST `/api/v1/subscribers` | Subscriber retornado pelo servidor |
| activateSubscriber | POST `/api/v1/subscribers/{id}/activate` | Subscriber retornado |
| suspendSubscriber | POST `/api/v1/subscribers/{id}/suspend` | Subscriber retornado; body `{reason}` quando preenchido |
| deactivateSubscriber | POST `/api/v1/subscribers/{id}/deactivate` | Subscriber retornado; body `{reason}` quando preenchido |

Paths relativos, IDs/query codificados e fetch nativo. Sem Axios ou framework de repository. Requests de Subscribers têm timeout de 10 segundos; leituras também aceitam AbortSignal. Erros não são normalizados para coleção vazia. Escritas não têm retry automático.

Proxy: mantém target central do Gate 2 (`NEXUS_API_TARGET`, default `http://127.0.0.1:8080`) e acrescenta apenas a família Subscriber. Não foi criado proxy/consumo para Devices ou Sessions neste Gate.

## 4. Fonte compartilhada entre KPI/lista

Um `SubscribersProvider` envolve a aplicação e mantém a única coleção completa usada pelo KPI e pela tela. O KPI é `collection.data.length`, sem usar quantidade filtrada ou uma lista independente. O número 128 e o percentual +12% foram retirados desse card.

Carregamento inicial, revalidação após sucesso e refresh manual. Não há polling periódico de Subscribers. Navegar entre Overview e Subscribers não recria a fonte. Requests de lista anteriores são cancelados/invalidados antes de uma nova revalidação, evitando que uma resposta antiga sobrescreva uma entidade confirmada.

Após escrita bem-sucedida, a entidade retornada é inserida/substituída por UUID no snapshot completo conhecido, e a coleção é revalidada. Se ainda não havia coleção completa, uma única entidade não é tratada como total global: aguarda-se a leitura completa.

## 5. Listagem

Colunas: **MSISDN, IMSI, Status, Created At, Updated At, Details**. A coluna Device mock foi removida da versão real.

Badges usam exatamente os quatro estados do domínio: pending em azul, active em verde, suspended em âmbar e deactivated em tom neutro. Ordenação local por created_at e UUID para estabilidade, sem paginação server-side inexistente.

Lista vazia real mostra “No subscribers provisioned yet”. Filtro sem resultados mostra “No matching subscribers”. Erro inicial mostra indisponibilidade, sem fingir ausência de registros. Em falha posterior, os registros conhecidos permanecem visíveis com aviso stale.

## 6. Busca

- Filtro local por IMSI, MSISDN e status, sem requests a cada tecla.
- “Find exact IMSI” habilitado para 15 dígitos: usa a consulta HTTP exata, abre o UUID encontrado e revalida a coleção.
- Falha da busca exata mantém a coleção carregada e apresenta o erro; não substitui a coleção global por um objeto ou lista vazia.
- Filtro visual não altera o KPI nem o total da coleção.

## 7. Details

O diálogo consulta GET por UUID. Exibe UUID, IMSI, MSISDN, status, created_at, updated_at e motivos opcionais quando presentes. Os timestamps completos aparecem no detalhe e em tooltips na tabela.

Não apresenta atributos inventados de outros domínios. Leituras em andamento são canceladas ao desmontar o diálogo. Uma falha na leitura oferece Retry details; lifecycle fica indisponível enquanto a leitura está pendente ou com erro.

O diálogo nativo mantém foco/modalidade e suporta fechamento por Escape quando não há operação de escrita pendente.

## 8. Provisionamento

“Provision subscriber” abre formulário IMSI/MSISDN. Validação básica:

- IMSI: exatamente 15 dígitos.
- MSISDN: `^\+?[1-9]\d{9,14}$`, conforme contrato auditado.

Os campos usam strings; não há conversão numérica ou truncamento interno. O formulário espera o POST e só mostra sucesso após resposta válida. A entidade retornada aparece no detalhe; coleção/KPI são atualizados/revalidados. Durante a requisição, submissão e fechamento do diálogo são desabilitados.

**Débito existente mantido:** regex pode aceitar `+` e 15 dígitos (16 caracteres), enquanto a coluna PostgreSQL auditada é VARCHAR(15). Não foi corrigido Go/migration nem reduzido silenciosamente o contrato. Os testes utilizaram MSISDNs de 15 caracteres totais, fora da borda ambígua.

## 9. Lifecycle

| Estado atual | Ações oferecidas |
|---|---|
| PENDING_ACTIVATION | Activate; Deactivate permanently |
| ACTIVE | Suspend; Deactivate permanently |
| SUSPENDED | Activate; Suspend (atualizar motivo); Deactivate permanently |
| DEACTIVATED | Nenhuma; aviso de estado terminal |

Suspend/Deactivate têm campo de motivo opcional. Status não muda otimisticamente: somente a entidade retornada altera o estado mostrado. Reativação usa o estado retornado, incluindo remoção de suspension_reason quando feita pelo servidor.

O backend continua autoridade. Em conflito, a mensagem 409 permanece visível e o detalhe/coleção são reconsultados para reconciliar alteração feita por outro operador.

## 10. Tratamento de erros

Mensagens preservam HTTP status, code e message quando fornecidos pelo Go. Isso inclui `SUBSCRIBER_ALREADY_EXISTS`, `SUBSCRIBER_ALREADY_DEACTIVATED`, `INVALID_STATE_TRANSITION` e erros internos. Não há conversão de 409 em sucesso.

Falhas de rede/timeout informam que a operação não pôde ser confirmada e orientam refresh antes de nova tentativa. Não há retry automático de provisionamento: um timeout não prova que o servidor deixou de gravar.

HTTP 400 foi validado no backend real com IMSI inválido; 409 foi validado via UI com IMSI/MSISDN duplicados e conflito terminal; 500 foi injetado no navegador para verificar o estado do formulário sem causar falha no banco.

## 11. Loading/error/stale

| Situação | KPI | Lista |
|---|---|---|
| Carregamento inicial | `—`, Loading | Loading subscribers |
| Sucesso vazio | 0 | Vazia de verdade |
| Sucesso com registros | Total completo | Coleção real filtrável |
| Falha sem snapshot | `—`, Unavailable | Erro; não “nenhum subscriber” |
| Falha após sucesso | Último total, Stale | Última coleção + aviso de desatualização |
| Recuperação por Refresh | Total revalidado | Coleção real atualizada |

Durante refresh com snapshot, ele continua visível e a tela indica Refreshing. Health/Telemetry permanecem independentes, com seus intervalos do Gate 2. A coleção não é consultada a cada 5 segundos.

## 12. Testes executados

`node tests/gate3.cjs`: **PASS — 13 grupos de verificação**, registrados em [results.json](evidence/gate3/results.json).

Cobertura:

1. Lista inicialmente vazia real e KPI 0.
2. Validação UX de IMSI inválido.
3. Provisionamento real em PENDING_ACTIVATION e identidades completas.
4. Activate → suspend com reason → reactivate → deactivate; estado terminal.
5. Busca local IMSI/MSISDN/status, busca exata HTTP e detalhes por UUID.
6. Duplicidade IMSI e duplicidade MSISDN, ambas 409 reais visíveis.
7. Provisionamento 500 sem sucesso ou aumento artificial do total.
8. IMSI inválido rejeitado por Go com 400 INVALID_TELECOM_IDENTITY.
9. Conflito terminal concorrente, com 409 visível e reconciliação do detalhe.
10. Falha de rede: stale, erro inicial sem lista vazia e recuperação manual.
11. Ausência de polling periódico da lista; Health/Telemetry continuam ativos.
12. Contrato `null` bem-sucedido normalizado; erro não normalizado.
13. Zero requests Devices/Sessions, nenhum erro JS não tratado e canvas nos viewports-alvo.

Chrome real via Playwright existente no runtime, sem instalar pacote. API Go iniciada com `PORT=18082`, `DATABASE_URL=''`, confirmando `storage_backend=memory`. Vite :5177 usa proxy para essa instância. As escritas reais foram feitas por meio da própria UI, exceto os requests pontuais para simular outro operador e testar rejeição de entrada pelo backend.

As falhas de rede, o 500 e a resposta `null` foram injetados no navegador para testar os contratos sem acessar ou danificar PostgreSQL. A lista vazia inicial e o lifecycle foram reais.

## 13. Evidência real do ciclo

Subscriber principal: `3b066839-20d5-4237-8a72-3cbe6bd24284`.

| Operação confirmada | Status retornado | Motivo |
|---|---|---|
| Provision | PENDING_ACTIVATION | Ausente |
| Activate | ACTIVE | Ausente |
| Suspend | SUSPENDED | `Gate 3 administrative hold` |
| Activate | ACTIVE | suspension_reason removido |
| Deactivate | DEACTIVATED | `Gate 3 contract ended` |

As respostas reais completas e requests observados estão no [results.json](evidence/gate3/results.json). Um segundo subscriber foi criado para validar uma desativação concorrente e o erro SUBSCRIBER_ALREADY_DEACTIVATED ao tentar ativar a partir de um detalhe ainda antigo.

## 14. Coerência KPI/lista

Estado inicial: 0. Após primeiro provisionamento: 1. Lifecycle e tentativas duplicadas/500 não aumentaram o total. Após segundo provisionamento: 2.

O teste navegou entre Overview e Subscribers e conferiu o mesmo total. Após falha de refresh, ambos mantiveram o último total conhecido; após reload com API inacessível, ambos ficaram sem snapshot (`—`), e o refresh recuperou 2. Não houve consultas N+1.

## 15. Screenshot Overview

[Overview 1920×1080](evidence/gate3/overview-1920x1080.png), capturado no Chrome e inspecionado visualmente. Total Subscribers real = 2; métricas de telemetry continuam reais. Mapa, containers, sidebar, diagonal, topbar e footer foram preservados.

## 16. Screenshots Subscribers

- [Subscribers 1920×1080](evidence/gate3/subscribers-1920x1080.png).
- [Detalhe SUSPENDED com motivo](evidence/gate3/details-suspended-1920x1080.png).
- [Lista stale](evidence/gate3/subscribers-stale-1920x1080.png).
- [Subscribers 1440×900](evidence/gate3/subscribers-1440x900.png).
- [Subscribers 1366×768](evidence/gate3/subscribers-1366x768.png).

Listagem e diálogo em 1920×1080 foram inspecionados visualmente. O teste também confirmou que o canvas cabe nos dois viewports menores. As adições de formulário, toolbar e badges seguem o tema existente; não houve redesenho do NEXUS.

## 17. npm run build

**PASS**, TypeScript + Vite:

```text
vite v6.4.3 building for production...
✓ 1598 modules transformed.
dist/index.html                   0.70 kB │ gzip: 0.40 kB
dist/assets/index-JmB0Tgze.css     27.87 kB │ gzip: 7.32 kB
dist/assets/index-B8kF3xoy.js     192.35 kB │ gzip: 60.79 kB
✓ built in 30.01s
```

## 18. go test ./...

**PASS**. Subscriber, Device e Session passaram em execução; outros pacotes de testes retornaram cache. cmd/api, cmd/migrate e platform/postgres não possuem testes.

`TEST_DATABASE_URL` foi mantido vazio no processo de teste. Testes PostgreSQL condicionais foram pulados; não se afirma validação contra banco real neste Gate. Nenhum código Go foi alterado.

## 19. Git e como testar

Na raiz do repositório, `git status --short`:

```text
?? web/
```

`web/` já estava untracked antes do Gate. `git diff --stat` não apresenta saída para esses arquivos não rastreados; isso não significa ausência de trabalho frontend. Não foi feito git add, commit ou push.

Frontend de teste: [http://127.0.0.1:5177/#subscribers](http://127.0.0.1:5177/#subscribers), enquanto os processos desta sessão estiverem ativos. Dados de teste residem apenas na memória da instância :18082.

Para reiniciar o ambiente, terminal na raiz do repositório:

```powershell
$env:PORT='18082'
$env:DATABASE_URL=''
go run ./cmd/api
```

Outro terminal, em `web/reference-version`:

```powershell
$env:NEXUS_API_TARGET='http://127.0.0.1:18082'
npm run dev -- --host 127.0.0.1 --port 5177 --strictPort
```

Teste automatizado, nessa mesma pasta: `node tests/gate3.cjs`. O teste exige a instância isolada indicada; não executar contra banco de produção. É possível configurar o módulo Playwright já instalado via `PLAYWRIGHT_MODULE` em outro ambiente.

## 20. Confirmação de isolamento e revisão

**ZERO alterações Go/backend, PostgreSQL, migrations, compose.yaml, go.mod ou go.sum.** Somente arquivos de `web/reference-version/` foram escritos. Nenhum endpoint foi criado; nenhum acesso direto ao DB, autenticação, WebSocket, SSE, pacote ou state manager foi acrescentado.

Não foram consultados Devices ou Sessions durante a validação de navegador deste Gate. Health/Telemetry permanecem integrados, e os outros domínios não foram avançados. O débito de borda do MSISDN permanece documentado, sem correção de backend/migration.

**PARAR PARA HUMAN REVIEW. Nenhum próximo Gate está autorizado por esta entrega. NÃO COMMIT.**
