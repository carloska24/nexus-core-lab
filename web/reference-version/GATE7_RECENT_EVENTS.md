# Gate 7 — Recent Events Feed

14/09/2026 — Implementado e validado. Aguardando Human Review.

Frontend: `C:\Users\joaob\OneDrive\Documentos\nexus-core-lab\web\reference-version`.
Backend: raiz do mesmo repositório, conforme autorização explícita do Gate 7.

## 1. Arquitetura escolhida e inspeção

Pipeline reutilizado: `session.Service` → `telemetrySessionAdapter` em `cmd/api/main.go` → canal do `telemetry.Worker` → `processEvent`. Session emite após a persistência bem-sucedida; o mesmo consumidor incrementa o contador correspondente e retém o evento. Não foi criado segundo mecanismo, goroutine, channel ou event bus.

O canal existente é não bloqueante e pode descartar eventos quando saturado, incrementando `dropped_events_total`. Esse comportamento foi preservado: a janela contém os eventos realmente consumidos, não é um histórico completo garantido de todas as operações. A leitura é eventualmente consistente com a emissão assíncrona.

## 2. Arquivos Go

Criados:

- `internal/telemetry/recent.go`
- `internal/telemetry/recent_test.go`

Modificados neste Gate:

- `internal/telemetry/event.go`: campos conhecidos do evento.
- `internal/telemetry/worker.go`: retenção no consumidor existente.
- `internal/session/event.go`: parâmetro de origem do Handover no contrato de emissão.
- `internal/session/service.go`: captura de origem antes da troca, sob lock do Device; emissão continua após persistência.
- `internal/session/session_test.go`: adaptação do emissor de teste e verificação de origem/destino.
- `cmd/api/main.go`: adaptação dos dados conhecidos e registro da rota.

As alterações em `internal/device/` já existiam antes do Gate 7 e foram preservadas. `go fmt ./...` normalizou terminações de linha; CRLF foi restaurado nos arquivos sem diferenças de conteúdo para evitar alterações incidentais fora do escopo.

## 3. Modelo Event final

Campos obrigatórios: `id`, `type`, `timestamp`, `device_id`, `subscriber_id`, `session_id`, `cell_id`.
Opcionais: `ip_address`, `from_cell_id`, `to_cell_id`, `disconnect_reason`.

Tipos exatos: ATTACH, CELL_HANDOVER, DETACH, STALE_DISCONNECT.

Device, Subscriber e Session mantêm seus UUIDs reais. O ID do evento é uma sequência decimal em string atribuída no consumo, única somente naquela execução do worker. Não é UUID nem identificador persistente; reinicia com a API. Timestamp é o instante UTC de emissão, como no pipeline anterior.

Cell, IP e motivo vêm da Session existente no ponto de emissão. Não houve consulta histórica nem inferência no frontend. Origem/destino são enviados apenas para CELL_HANDOVER.

## 4. Capacidade

Máximo de 100 eventos em memória por Worker/API. Sem crescimento ilimitado.

## 5. Eviction

Ao atingir 100 registros, remove o primeiro consumido ainda retido e acrescenta o novo. Teste de 104 eventos confirmou tamanho 100 e retenção dos eventos 5 a 104.

## 6. Thread-safety

Retenção, sequência e snapshots protegidos por `sync.RWMutex`. Snapshot copia os valores para um novo slice. Contadores atômicos e sincronização do canal/shutdown existentes foram preservados. Teste concorrente emite 1600 eventos, lê e modifica cópias defensivas e confirma ausência de perdas nesse cenário dimensionado.

## 7. Endpoint

`GET /api/v1/events/recent` → HTTP 200, `Content-Type: application/json`, `Cache-Control: no-store`.

Sem paginação, filtros, query language, endpoint por tipo ou contagem. Proxy de desenvolvimento do Vite foi atualizado especificamente para essa rota. Em produção, a rota deve chegar à mesma API Go, como as demais rotas integradas.

## 8. JSON

Resposta é um array; vazio retorna `[]`, não null. Exemplo da execução real de teste, reduzido a um registro completo:

```json
[
  {
    "id": "2",
    "type": "CELL_HANDOVER",
    "timestamp": "2026-09-14T18:01:37.7335304Z",
    "device_id": "05b832ad-8b8f-4715-aad9-e84861dcb1ce",
    "subscriber_id": "fbaa6c89-b6b2-447d-9c82-063acbe8f217",
    "session_id": "2c2d3a20-38f6-4930-9a80-e26be9b0acf3",
    "cell_id": "CELL-SP-002",
    "ip_address": "10.45.0.2",
    "from_cell_id": "CELL-SP-001",
    "to_cell_id": "CELL-SP-002"
  }
]
```

Timestamp UTC do backend; a interface apresenta a hora local. Payload completo em `evidence/gate7/results.json`.

## 9. Ordenação

Mais recentemente consumido primeiro, pela sequência de consumo do único worker. Não ordena por relógio: timestamps empatados ou regressivos não alteram essa ordem. Entre emissões concorrentes, prevalece a ordem efetivamente recebida pelo canal; não promete ordem de início das requisições.

## 10. Restart

API isolada foi encerrada e reiniciada. A primeira leitura retornou `[]`; novos eventos apareceram normalmente. Evidência: `evidence/gate7/restart.json`. Nova instância de Worker também é testada sem histórico. A interface comunica “Current API execution only”; a tela Events explicita o reset no restart.

## 11–15. Eventos observados e Handover

- ATTACH observado com Cell, IP e UUIDs reais.
- CELL_HANDOVER observado de CELL-SP-001 para CELL-SP-002.
- DETACH observado com `VOLUNTARY_DETACH` e última célula conhecida.
- STALE_DISCONNECT observado no re-attach, vinculado à sessão anterior e sua célula, com motivo `STALE_DISCONNECT`.
- `from_cell_id` é capturado antes de `sess.Handover` sob lock do Device. Só é emitido após `repo.Update` bem-sucedido; `to_cell_id` vem da sessão atualizada. No-op de Handover continua sem evento.

## 16. Counters Telemetry

Preservados `events_total.attach`, `cell_handover`, `detach`, `stale_disconnect`, além de `dropped_events_total`. Cada evento consumido é contado uma única vez. Teste real: attach=3, cell_handover=1, detach=1, stale_disconnect=1. Teste de retenção: 104 eventos totais com janela de apenas 100, demonstrando conceitos distintos.

Total Events continua calculado exclusivamente por `/telemetry`. Não usa tamanho do feed.

## 17. Arquivos frontend

Novos: `src/recent-events.tsx`, `src/RecentEvents.tsx`, `src/recent-events.css`, `tests/gate7.cjs`.
Modificados: `src/App.tsx`, `vite.config.ts`, `README.md`.
Documentação/evidências: este relatório e `evidence/gate7/`.

Fixtures do painel Recent Events foram removidos. Overview mostra cinco registros; Events mostra a janela completa. Device abre detalhes com todos os campos retornados, inclusive UUIDs reais, Subscriber, Session, Cell e IP. UUID abreviado visualmente mantém o valor completo nos detalhes. Não foi acrescentado tipo fictício.

## 18. Polling

Um RecentEventsProvider compartilhado para Overview e Events. Próxima consulta cinco segundos após concluir a anterior; timeout de oito segundos, sem sobreposição. Unmount cancela request e timer. Não há polling adicional por painel. Validação de JSON rejeita coleção maior que 100, IDs duplicados, campos obrigatórios inválidos e tipos desconhecidos.

## 19. Estados

Loading antes da primeira resposta; success para snapshot válido, inclusive vazio; stale preserva o último snapshot após falha; error representa falha sem snapshot anterior. Teste confirmou seis registros preservados durante HTTP 503, erro inicial sem falsa mensagem de lista vazia e recuperação automática.

## 20. Hero Flow real

Executado contra API isolada em memória: Subscriber → Activate → Device → Attach CELL-SP-001 → Handover CELL-SP-002 → Detach. Feed retornou DETACH, CELL_HANDOVER, ATTACH nessa ordem de leitura. Sessão e IP permaneceram iguais durante Handover. Operações realizadas por requisições HTTP reais; navegação e validação do painel realizadas em Chrome headless.

## 21. Re-attach real

Após o Detach, novo Attach criou uma sessão ativa. Outro Attach no mesmo Device gerou STALE_DISCONNECT da sessão anterior, seguido de ATTACH da substituta. A leitura newest-first mostrou ATTACH antes de STALE_DISCONNECT. Esse comportamento foi observado e comparado aos UUIDs retornados, sem simulação de eventos.

## 22–28. Verificações

| Verificação | Resultado |
| --- | --- |
| npm run build | PASS final, 1612 módulos |
| go fmt ./... | PASS |
| go vet ./... | PASS |
| go build ./... | PASS |
| go test -count=1 ./... | PASS |
| go test -race -count=1 ./... | PASS no Docker |
| Data races detectadas | **0** |
| tests/gate7.cjs | PASS, 10 verificações; zero erros JavaScript não tratados |
| Reinício real | PASS, 4 verificações adicionais |
| Rodapé nas três resoluções | PASS, sem corte |

Windows estava com CGO_ENABLED=0, impossibilitando race nativo. Foi usada a imagem oficial `golang:latest` já instalada, sem rede, código e cache de módulos montados somente leitura, CGO_ENABLED=1, GOTOOLCHAIN=local e GOPROXY=off. Nenhuma instalação ou serviço de terceiros criado.

Comando executado no container: `go test -race -count=1 ./...`.

O teste de resposta lenta confirmou máximo de uma consulta de feed simultânea e intervalo de pelo menos 4,8 segundos após conclusão, tolerância do teste para o agendamento nominal de cinco segundos. Sair da aplicação interrompeu consultas.

## 29. PostgreSQL

TEST_DATABASE_URL indisponível. Testes PostgreSQL reais **não executados**. Não foi criado banco, migration ou persistência de eventos. PASS geral não é apresentado como validação PostgreSQL.

## 30. Evidência visual

Screenshots reais finais, preservando composição e topologia do Gate 6:

- [1920x1080](evidence/gate7/overview-1920x1080.png)
- [1440x900](evidence/gate7/overview-1440x900.png)
- [1366x768](evidence/gate7/overview-1366x768.png)

Inspeção visual identificou e corrigiu corte do rodapé e largura da tabela. Alterações CSS são restritas ao painel de eventos. Não foram alterados mapa, coordenadas, antenas ou lógica de TopologyProvider. Ambiente de evidências: frontend `http://localhost:5198`, API `http://127.0.0.1:18098`, somente memória; dados da instância habitual do usuário não foram utilizados.

## 31. git diff --stat

Inclui alterações preexistentes de Devices; arquivos novos e `web/` untracked não aparecem no diff convencional:

```text
 cmd/api/main.go                             | 23 ++++++++++++-----
 internal/device/handler.go                  | 14 +++++++---
 internal/device/handler_test.go             |  4 +--
 internal/device/memory_repository.go        | 18 +++++++++++++
 internal/device/postgres_repository.go      | 25 ++++++++++++++++++
 internal/device/postgres_repository_test.go | 40 +++++++++++++++++++++++++++++
 internal/device/repository.go               |  1 +
 internal/device/service.go                  |  5 ++++
 internal/session/event.go                   |  1 +
 internal/session/service.go                 |  9 ++++---
 internal/session/session_test.go            | 13 +++++++---
 internal/telemetry/event.go                 | 17 +++++++-----
 internal/telemetry/worker.go                |  6 +++++
 13 files changed, 149 insertions(+), 27 deletions(-)
```

## 32. git status --short

```text
 M cmd/api/main.go
 M internal/device/handler.go
 M internal/device/handler_test.go
 M internal/device/memory_repository.go
 M internal/device/postgres_repository.go
 M internal/device/postgres_repository_test.go
 M internal/device/repository.go
 M internal/device/service.go
 M internal/session/event.go
 M internal/session/service.go
 M internal/session/session_test.go
 M internal/telemetry/event.go
 M internal/telemetry/worker.go
?? internal/device/list_test.go
?? internal/telemetry/recent.go
?? internal/telemetry/recent_test.go
?? web/
```

## 33–35. Limites e revisão

- Nenhuma persistência de eventos, PostgreSQL para eventos, migration ou histórico infinito.
- IPPool, Simulator, Database Diagnostics, Google Maps e autenticação não integrados.
- Regras de Subscriber suspenso e compatibilidade LTE/5G não alteradas.
- Nenhum Gate posterior iniciado. Sem commit e sem push.

**PARADO PARA HUMAN REVIEW.**
