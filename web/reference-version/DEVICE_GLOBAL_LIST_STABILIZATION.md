# Device Global List — Stabilization

Data: 14/09/2026. Micro-gate restrito à regressão de ordenação do teste HTTP global de Devices.

## 1. Causa da falha

TestGlobalDeviceHTTPContract comparava a resposta global com a ordem em que dois Devices foram registrados. CreatedAt pode empatar; nesse caso a ordem de inserção não corresponde necessariamente ao contrato. A falha foi reproduzida antes da edição: 7 falhas em 30 repetições, sempre com timestamps iguais e IDs retornados em ordem lexicográfica.

## 2. Contrato confirmado no código

- internal/device/memory_repository.go, List: CreatedAt.Before; se CreatedAt.Equal, compara ID com `<`.
- internal/device/postgres_repository.go, List: `ORDER BY created_at ASC, id ASC`.

Portanto, a ordem é created_at ASC e id ASC em empate. Nenhum repositório foi modificado neste micro-gate.

## 3–5. Mudança mínima

Único arquivo Go modificado: internal/device/list_test.go.

Adicionado import de sort e ordenação da slice esperada `created`, antes de compará-la à resposta HTTP. Total: 8 linhas adicionadas, zero removidas. Não se ordena a resposta recebida: o teste continua verificando a ordem real devolvida pelo endpoint.

```go
// The global contract orders by CreatedAt, then ID when timestamps tie.
sort.Slice(created, func(i, j int) bool {
    if created[i].CreatedAt.Equal(created[j].CreatedAt) {
        return created[i].ID < created[j].ID
    }
    return created[i].CreatedAt.Before(created[j].CreatedAt)
})
```

As asserções de quantidade, IDs, filtro, detalhes e duplicidade permanecem. O teste existente TestMemoryDeviceGlobalListCopiesAndOrder já usa timestamps empatados de forma determinística e valida o desempate. Nenhuma mudança funcional, novo endpoint, schema ou migration.

## 6–9. Verificações Go

| Comando | Resultado |
|---|---|
| go fmt ./... | Executado com sucesso. Restaurados os bytes originais de 43 arquivos com normalizações incidentais fora do escopo. |
| go vet ./... | PASS |
| go build ./... | PASS |
| go test -count=1 ./... | PASS, suíte completa sem cache de resultados |
| go test -count=100 -run '^TestGlobalDeviceHTTPContract$' ./internal/device | PASS, 100 execuções consecutivas |

Resultados de vet/build/test obtidos antes da pausa solicitada pelo usuário. Na retomada, a comparação com o snapshot confirmou que o único arquivo Go diferente do início do micro-gate continua sendo list_test.go.

## 10–11. Race detector

**PASS — go test -race -count=1 ./... concluiu com exit code 0. ZERO data races detectadas na suíte executada.** Todos os pacotes com testes passaram: cmd/simulator, internal/device, internal/network, internal/platform/httpserver, internal/session, internal/subscriber e internal/telemetry.

O race detector nativo Windows não iniciou porque CGO_ENABLED=0. Utilizada a alternativa Docker autorizada: imagem oficial golang:latest já instalada, CGO_ENABLED=1, GOTOOLCHAIN=local, GOPROXY=off, sem rede. Código e cache de módulos montados somente para leitura. Comando:

```text
go test -race -count=1 ./...
```

A execução anterior à pausa não tinha resultado recuperável; apenas essa verificação foi repetida na retomada.

## 12. PostgreSQL

Testes PostgreSQL reais NÃO executados: TEST_DATABASE_URL ausente no ambiente de integração usado. Não foi criado banco nem executada migration. Não foram utilizados os bancos de aplicações existentes no Docker. Os testes condicionais de PostgreSQL foram pulados pelas próprias suítes; o PASS geral não é apresentado como validação PostgreSQL.

## 13. Contrato HTTP preservado

O teste usa o ServeMux real com httptest, cobrindo:

- GET /api/v1/devices: coleção global, vazio inicial e ordenação dos registros.
- GET /api/v1/devices?subscriber_id={id}: apenas Devices daquele Subscriber; filtro sem resultados preservado.
- GET /api/v1/devices/{id}: Device individual completo.
- POST /api/v1/devices: registro HTTP 201; IMEI duplicado HTTP 409 sem aumentar a coleção.

Nenhum handler ou contrato funcional foi alterado neste micro-gate.

## 14. git diff --stat

O repositório já tinha alterações do Gate4A. Saída do diff rastreado:

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

Essas mudanças são preexistentes. list_test.go já era não rastreado e não aparece no diff normal. Comparação separada com sua cópia do início do micro-gate: `1 file changed, 8 insertions(+)`. Este relatório é o único novo artefato da entrega.

## 15. git status --short

```text
 M internal/device/handler.go
 M internal/device/handler_test.go
 M internal/device/memory_repository.go
 M internal/device/postgres_repository.go
 M internal/device/postgres_repository_test.go
 M internal/device/repository.go
 M internal/device/service.go
?? DEVICE_GLOBAL_LIST_STABILIZATION.md
?? internal/device/list_test.go
?? web/
```

Nenhum commit, push ou alteração do índice.

## 16–17. Escopo preservado

Comparação dos 78 arquivos de web/reference-version registrados no snapshot (excluindo node_modules): ZERO arquivos alterados. Gates 4 e 5 preservados.

Nenhuma mudança em Subscriber, Session, Telemetry, Network, IPPool, Simulator, schema ou migrations. Não alteradas as regras de Subscriber suspenso/Attach nem compatibilidade LTE/5G com Cell. Gate 6 NÃO iniciado.

Parada para HUMAN REVIEW.
