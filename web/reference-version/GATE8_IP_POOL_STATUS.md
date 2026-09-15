# Gate 8 — Authoritative IP Pool Status

Entrega em 14/09/2026, aprovada pelo Human Review em 15/09/2026. Este relatório preserva o estado da entrega anterior ao checkpoint. Validações Go e build frontend repetidos para o checkpoint; PostgreSQL integration tests not executed (TEST_DATABASE_URL indisponível).

## 1. Baseline Git

Branch `feat/demo-ui`; HEAD `8cb9e0f6db3d1d5db1faaa712c71bac35e7f26f9`. Working tree confirmado clean antes da implementação. HEAD permanece inalterado.

## 2. Arquivos alterados

Backend: `cmd/api/main.go`, `internal/network/ippool.go`; novos `internal/network/ippool_status.go` e `internal/network/ippool_status_test.go`.
Frontend e documentação: arquivos listados nos itens 23 e 36. Alterações incidentais de finais de linha em arquivos Go fora do escopo foram removidas, conferindo conteúdo contra HEAD.

## 3–6. Implementação original e capacidade confirmada

O IPPool contém mutex, cursor `nextHost`, conjunto `allocated` e pilha `recycled`. Allocate prioriza endereços reciclados; Release remove do conjunto e coloca na pilha. A política existente de reutilização é LIFO e foi preservada.

- CIDR: `10.45.0.0/16`.
- Reservados: `10.45.0.0`, `10.45.0.1`, `10.45.255.255`.
- Utilizáveis: `10.45.0.2` até `10.45.255.254`.
- Capacidade: `65534 - 2 + 1 = 65533` endereços.

Confirmado nas constantes e validações do código, antes da implementação.

## 7–8. Dívida reproduzida e correção local

O teste determinístico `TestIPPoolWarmupHighAddressLeavesHolesReachable` criou um pool, marcou `10.45.255.254` e tentou Allocate. Antes da correção falhou:

```text
65532 usable holes remain after warm-up, Allocate failed: ip pool exhausted
```

MarkAllocated avançava o cursor para depois do endereço marcado, tornando endereços inferiores livres inalcançáveis. Agora MarkAllocated não avança o cursor; Allocate percorre os candidatos e ignora os que já estão alocados. Endereços liberados anteriormente continuam na pilha de reutilização. Todo endereço utilizável livre permanece elegível.

MarkAllocated também normaliza a representação IPv4 antes da verificação de duplicidade, evitando duas representações do mesmo endereço no conjunto. Não houve redesign, nova configuração produtiva ou alteração das regras de Session.

## 9–13. Snapshot e contrato HTTP

`IPPoolSnapshot` expõe somente `cidr`, `capacity`, `allocated`, `available`, `utilization_percent`.

`GET /api/v1/network/ip-pool` retorna HTTP 200, JSON e `Cache-Control: no-store`. Exemplo real após dois Attach:

```json
{"cidr":"10.45.0.0/16","capacity":65533,"allocated":2,"available":65531,"utilization_percent":0.0030518975172813697}
```

O handler recebe o MESMO ponteiro de IPPool entregue ao SessionService e utilizado pelo startup warm-up. Snapshot lê sob o mutex existente; não altera cursor, conjunto ou pilha. Não reserva nem libera IPs. POST, PUT e DELETE foram testados e retornam 405. Não foram criadas rotas públicas de alocação, liberação ou listagem de IPs.

`available = capacity - allocated`; percentual = `allocated / capacity * 100`. A correção dos holes permite que essa disponibilidade represente capacidade utilizável real. O snapshot é atômico no allocator, não uma transação conjunta com o repositório de Sessions: pode refletir reservas de operações ainda em andamento.

## 14–18. Fluxo real de Sessions

Teste de navegador usando API Go isolada, repositórios em memória e operações reais:

| Operação concluída | Allocated | Available |
|---|---:|---:|
| Inicial | 0 | 65533 |
| Attach A | 1 | 65532 |
| Attach B | 2 | 65531 |
| Handover B | 2 | 65531 |
| Detach A | 1 | 65532 |
| Re-attach B | 1 | 65532 |

Os dois primeiros IPs foram distintos e pertencem ao allocator observado. Handover preservou IP e quantidade. Re-attach terminou com uma única alocação; o IP anterior foi liberado e houve reutilização de endereço liberado. Recent Events continuou apresentando o stale-disconnect.

A implementação existente de re-attach pode manter uma reserva adicional durante a operação; o teste verifica o estado final. Não foram alterados Attach, Detach, transação PostgreSQL, compensações ou comportamento de pool cheio durante re-attach.

## 19–22. Exaustão, reutilização, warm-up e holes

Testes diretos no allocator cobrem:

- Capacidade completa de 65533 endereços únicos, inclusive após MarkAllocated de três endereços esparsos/altos.
- Nenhuma exaustão prematura; pool cheio retorna ErrIPPoolExhausted, available 0 e utilização 100%.
- Release torna o endereço elegível novamente; Allocate o reutiliza.
- Reservados, inválidos, fora da rede e MarkAllocated duplicado rejeitados.
- MarkAllocated remove endereço da pilha de reciclados para evitar duplicidade.
- Snapshot não altera cursor; consultas HTTP repetidas não alteram estado.
- Oito workers alocam 800 IPs distintos; snapshots mantêm invariantes durante alocação/liberação concorrente.

O warm-up existente consulta Sessions CONNECTED e chama MarkAllocated no mesmo pool antes de iniciar HTTP. A semântica de restauração foi testada diretamente com MarkAllocated, incluindo o cenário de holes. Restart PostgreSQL E2E não foi executado; ver item 34.

## 23. Frontend

Novos: `src/IPPoolCard.tsx`, `src/ip-pool-api.ts`, `src/ip-pool.css`, `tests/gate8.cjs`.
Alterados: `src/App.tsx` substitui o antigo card mock; `src/monitoring.tsx` somente exporta o hook de polling existente; `vite.config.ts` adiciona proxy da nova rota. README e este relatório documentam a entrega. Nenhuma dependência adicionada.

Composição do card preservada no Overview e na tela Telemetry. Valores vêm exclusivamente do endpoint, sem contagem de Sessions/Devices, Telemetry ou Recent Events. Percentual usa até quatro casas decimais; a barra não exagera visualmente uma ocupação pequena. O rodapé mostra estado da leitura e instante observado, sem afirmar warm-up fictício.

## 24–25. Polling e estados

Polling a cada cinco segundos após conclusão da leitura, timeout e cancelamento via transporte/hook existente. Respostas são validadas: inteiros não negativos, totais consistentes e percentual compatível.

- Loading: valores desconhecidos, representados por travessões.
- Success: snapshot autoritativo.
- Stale: último snapshot preservado após falha.
- Error inicial: desconhecido, nunca convertido em zero.
- Recovery automática validada.

Testadas falha de rede, resposta inconsistente, leitura lenta sem sobreposição e interrupção do polling no unmount. Não há atualização otimista.

## 26. Evidências visuais e navegador

11 verificações de navegador passaram; zero erros JavaScript não tratados. Resultados e snapshots: [results.json](evidence/gate8/results.json). Script reproduzível: [gate8.cjs](tests/gate8.cjs).

Nove screenshots reais do Overview; limites do card verificados nas três resoluções e amostras inspecionadas visualmente:

| Estado | 1920x1080 | 1440x900 | 1366x768 |
|---|---|---|---|
| Vazio | [PNG](evidence/gate8/empty-1920x1080.png) | [PNG](evidence/gate8/empty-1440x900.png) | [PNG](evidence/gate8/empty-1366x768.png) |
| Dois alocados | [PNG](evidence/gate8/allocated-1920x1080.png) | [PNG](evidence/gate8/allocated-1440x900.png) | [PNG](evidence/gate8/allocated-1366x768.png) |
| Após Detach | [PNG](evidence/gate8/detached-1920x1080.png) | [PNG](evidence/gate8/detached-1440x900.png) | [PNG](evidence/gate8/detached-1366x768.png) |

PNGs permanecem locais e ignorados pelo .gitignore já existente; JSON contém evidência versionável. Ambiente isolado: API 18108 e frontend localhost:5208, sem banco do usuário.

## 27–34. Validações

| Item | Comando/validação | Resultado |
|---|---|---|
| 27 | npm run build | PASS |
| 28 | go fmt ./... | PASS |
| 29 | go vet ./... | PASS |
| 30 | go build ./... | PASS |
| 31 | go test -count=1 ./... | PASS |
| 32 | go test -race -count=1 ./... no Docker | PASS |
| 33 | Data races | ZERO |
| 34 | PostgreSQL integration tests | NÃO EXECUTADOS |

Race detector executado em golang:latest, CGO_ENABLED=1, GOTOOLCHAIN=local e GOPROXY=off; código e cache de módulos montados somente para leitura, rede Docker desabilitada.

TEST_DATABASE_URL não estava disponível. **PostgreSQL integration tests not executed.** Restart/warm-up PostgreSQL E2E também não executado. Nenhum banco foi criado para substituir essa validação.

## 35. git diff --stat

Somente arquivos rastreados; arquivos novos aparecem no status abaixo:

```text
 cmd/api/main.go                          |  1 +
 internal/network/ippool.go               | 26 +++++++++++++-------------
 web/reference-version/README.md          |  6 ++++--
 web/reference-version/src/App.tsx        |  8 +++-----
 web/reference-version/src/monitoring.tsx |  2 +-
 web/reference-version/vite.config.ts     |  1 +
 6 files changed, 23 insertions(+), 21 deletions(-)
```

## 36. git status --short

```text
 M cmd/api/main.go
 M internal/network/ippool.go
 M web/reference-version/README.md
 M web/reference-version/src/App.tsx
 M web/reference-version/src/monitoring.tsx
 M web/reference-version/vite.config.ts
?? internal/network/ippool_status.go
?? internal/network/ippool_status_test.go
?? web/reference-version/GATE8_IP_POOL_STATUS.md
?? web/reference-version/evidence/gate8/
?? web/reference-version/src/IPPoolCard.tsx
?? web/reference-version/src/ip-pool-api.ts
?? web/reference-version/src/ip-pool.css
?? web/reference-version/tests/gate8.cjs
```

## 37–39. Escopo e parada

Nenhuma migration, banco novo ou persistência separada do pool. Sessions CONNECTED continuam sendo a fonte persistente para warm-up. Nenhuma alteração nas regras de Subscriber/Device, compatibilidade LTE/5G, Network Topology, Recent Events ou fontes de métricas de Telemetry. Interface original fora de reference-version preservada.

**Gate 9 não iniciado. Nenhum commit ou push realizado neste Gate. Parado para Human Review.**
