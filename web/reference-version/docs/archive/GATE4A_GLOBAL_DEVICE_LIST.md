# Gate 4A — Global Device List Contract

**Concluído para HUMAN REVIEW.** Verificações finais conferidas em 12/09/2026. Gate 4 frontend não iniciado.

## 1. Causa original

O GET `/api/v1/devices` estava conectado exclusivamente à listagem por Subscriber. Sem `subscriber_id`, retornava HTTP 400 `MISSING_QUERY_PARAMETER`, impedindo a coleção global solicitada para o próximo gate.

## 2. Arquivos Go

Modificados, exclusivamente em `internal/device/`:

- `repository.go`
- `memory_repository.go`
- `postgres_repository.go`
- `service.go`
- `handler.go`
- `handler_test.go`
- `postgres_repository_test.go`

Criado: `internal/device/list_test.go`.

## 3. Repository antes/depois

Antes: `Save`, `FindByID`, `FindByIMEI`, `ListBySubscriber`.

Depois: todas as operações anteriores, mais:

```go
List(ctx context.Context) ([]*Device, error)
```

Nenhuma abstração adicional ou alteração no modelo Device.

## 4. MemoryRepository

`List` usa RLock e cria uma nova slice, com `copyDevice` para cada entidade. Não expõe o mapa nem seus ponteiros. Ordena por `created_at`, com desempate por `id`. Coleção global vazia é não nil e serializa como `[]`.

## 5. PostgresRepository

SQL explícito, reutilizando o scanner existente:

```sql
SELECT id, subscriber_id, imei, technology, status, created_at, updated_at
FROM devices
ORDER BY created_at ASC, id ASC
```

Fecha rows, propaga erros de consulta, scan e iteração. Coleção global vazia serializa como `[]`. `ListBySubscriber` permanece intacto, inclusive sua convenção anterior de vazio.

## 6. Service

Novo `List(ctx)` delega diretamente ao Repository. Registro, validações e consultas anteriores permanecem iguais.

## 7. Handler

O handler de coleção passa a se chamar `handleList`. Sem o parâmetro, chama `Service.List`. Com parâmetro não vazio, chama `ListBySubscriber`. Parâmetro explicitamente vazio continua retornando 400, evitando transformar um filtro inválido em exposição involuntária da coleção completa.

## 8. Contrato HTTP final

| Requisição | Resposta |
|---|---|
| GET `/api/v1/devices` | 200, coleção completa |
| GET `/api/v1/devices` sem registros | 200, `[]` |
| GET `/api/v1/devices?subscriber_id=ID` | 200, somente Devices daquele Subscriber |
| GET com Subscriber sem Devices | 200, coleção vazia conforme implementação existente |
| GET `/api/v1/devices?subscriber_id=` | 400, `MISSING_QUERY_PARAMETER` preservado |
| GET `/api/v1/devices/{id}` | Comportamento anterior preservado |
| POST `/api/v1/devices` | Comportamento anterior preservado, inclusive duplicidade 409 |
| Falha no Repository de listagem global | 500, `INTERNAL_SERVER_ERROR` |

Não foi criado endpoint separado ou endpoint de contagem.

## 9–12. Testes de contrato

`TestGlobalDeviceHTTPContract` exercita o mux HTTP real com Repository em memória: GET inicial vazio 200/[], POST A e B em assinantes diferentes, GET global A+B, filtros isolados, GET individual com igualdade de todos os campos, filtro vazio sem resultados e IMEI duplicado 409 sem aumentar a coleção.

`TestMemoryDeviceGlobalListCopiesAndOrder` verifica ordenação cronológica, desempate por ID e cópias defensivas de slice/entidades.

`TestMemoryDeviceGlobalListConcurrent` executa oito goroutines, cada uma gravando 30 Devices e consultando/mutando suas cópias retornadas, conferindo os 240 registros intactos ao final.

`TestGlobalDeviceListFailure` confirma HTTP 500 com código correto; erros não viram coleção vazia.

O teste PostgreSQL existente foi ampliado com terceiro Device de outro Subscriber, presença dos três no `List` global do Repository, ordenação e preservação do filtro com dois Devices. Fixture adicional possui identidade distinta dos testes de Session.

## 13–19. Verificações e ambientes

| Verificação | Resultado |
|---|---|
| `go fmt ./...` | Executado com sucesso; preservados os bytes dos arquivos fora de Device para evitar alterações incidentais de formatação |
| `go vet ./...` | PASS no Windows e no Linux/Docker |
| `go build ./...` | PASS no Windows e no Linux/Docker |
| `go test ./...` | PASS no Windows; testes PostgreSQL condicionais não executados nesse ambiente, pois TEST_DATABASE_URL estava ausente |
| `go test -count=1 -v ./internal/device` | PASS no Docker, incluindo TestPostgresRepository_DevicePersistence (sem skip) |
| `go test -count=1 ./...` | PASS no Docker com TEST_DATABASE_URL configurada |
| `go test -race -count=1 ./...` | PASS no Docker com CGO_ENABLED=1 e TEST_DATABASE_URL configurada |
| Data races | **Zero detectadas** na suíte executada |
| `git diff --check` | PASS |

O race detector nativo não iniciou no Windows: `go: -race requires cgo; enable cgo by setting CGO_ENABLED=1`. A alternativa autorizada Docker foi utilizada com sucesso, em Go 1.27.1 linux/amd64.

Foi criado o container descartável `nexus-gate4a-test-db`, PostgreSQL 16-alpine, rede exclusiva `nexus-gate4a-test`, armazenamento tmpfs e sem exposição de porta no host. As três migrations **já existentes** prepararam apenas esse banco novo. O banco persistente `nexus-core-lab-postgres` não foi utilizado.

O container de validação recebeu o código e o cache de módulos Go do host em modo somente leitura. A primeira tentativa de download de dependência falhou por timeout TLS; a execução final usou o cache local com GOPROXY=off. Não foram alteradas dependências.

Sequência Linux final (exit code 0):

```sh
go run ./cmd/migrate -up
go vet ./...
go build ./...
go test -count=1 -v ./internal/device
go test -count=1 ./...
go test -race -count=1 ./...
```

`DATABASE_URL` e `TEST_DATABASE_URL` apontaram exclusivamente para o banco descartável. A validação PostgreSQL foi real, e não somente compilação dos testes.

Saída final do race detector:

```text
?   github.com/carloska24/nexus-core-lab/cmd/api [no test files]
?   github.com/carloska24/nexus-core-lab/cmd/migrate [no test files]
ok  github.com/carloska24/nexus-core-lab/cmd/simulator 2.986s
ok  github.com/carloska24/nexus-core-lab/internal/device 4.466s
ok  github.com/carloska24/nexus-core-lab/internal/network 2.019s
ok  github.com/carloska24/nexus-core-lab/internal/platform/httpserver 2.994s
?   github.com/carloska24/nexus-core-lab/internal/platform/postgres [no test files]
ok  github.com/carloska24/nexus-core-lab/internal/session 5.831s
ok  github.com/carloska24/nexus-core-lab/internal/subscriber 2.469s
ok  github.com/carloska24/nexus-core-lab/internal/telemetry 2.031s
```

Testes HTTP utilizaram httptest e o mux real. Não foi reiniciada a API de desenvolvimento anteriormente aberta; um processo antigo precisa ser reiniciado para servir o novo contrato.

Após a validação, o PostgreSQL temporário foi encerrado e sua rede exclusiva removida. Nenhum serviço de teste ficou em execução.

## 20. git diff --stat

```text
 internal/device/handler.go                  | 14 +++++++---
 internal/device/handler_test.go             |  4 +--
 internal/device/memory_repository.go        | 18 +++++++++++++
 internal/device/postgres_repository.go      | 25 ++++++++++++++++++
 internal/device/postgres_repository_test.go | 40 +++++++++++++++++++++++++++++
 internal/device/repository.go               |  1 +
 internal/device/service.go                  |  5 ++++
 7 files changed, 101 insertions(+), 6 deletions(-)
```

O stat do Git não inclui o arquivo novo não rastreado `list_test.go` (173 linhas), nem este relatório dentro de `web/`, diretório já não rastreado anteriormente.

## 21. git status --short

```text
 M internal/device/handler.go
 M internal/device/handler_test.go
 M internal/device/memory_repository.go
 M internal/device/postgres_repository.go
 M internal/device/postgres_repository_test.go
 M internal/device/repository.go
 M internal/device/service.go
?? internal/device/list_test.go
?? web/
```

## 22–23. Limites preservados

- Nenhuma migration, alteração de schema ou índice foi criada no projeto.
- Nenhuma alteração em Subscriber, Session, Telemetry, IPPool, compose ou dependências.
- Nenhuma implementação de frontend, agregação N+1, paginação ou avanço de Sessions.
- Nenhuma mudança no banco persistente do projeto.
- Sem commit/push.
- Gate 4 frontend aguarda aprovação humana do Gate 4A.
