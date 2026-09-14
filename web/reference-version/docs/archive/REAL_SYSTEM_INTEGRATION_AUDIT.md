# NEXUS CORE LAB — REAL SYSTEM INTEGRATION

## GATE 1 — BACKEND CONTRACT AUDIT

**Data:** 11/09/2026. **Estado:** concluído para HUMAN REVIEW; nenhuma integração autorizada ou implementada.

A interface em `web/reference-version/` está visualmente aprovada. Este documento audita o código existente e distingue contratos HTTP, estado interno do backend e conteúdo demonstrativo da interface. Não transforma propostas em contratos existentes.

**Escopo executado:** leitura de handlers, entidades, services, repositories em memória e PostgreSQL, composição da API, network, telemetry, simulator, migrações, testes e documentação. Somente este Markdown foi criado. Não foram executados simulator, migrações, operações HTTP de escrita, testes contra banco ou alterações de configuração. Não houve instalação, alteração de Go, UI, mapa ou mocks, commit ou push. As conclusões são estáticas; não constituem certificação de uma instância em execução nem medição de desempenho.

### Fontes e precedência

Os caminhos abaixo são relativos a este documento:

- Composição: [cmd/api/main.go](../../cmd/api/main.go).
- HTTP: [server.go](../../internal/platform/httpserver/server.go), [middleware.go](../../internal/platform/httpserver/middleware.go).
- Subscriber: [handler](../../internal/subscriber/handler.go), [entidade](../../internal/subscriber/subscriber.go), [service](../../internal/subscriber/service.go), [repository](../../internal/subscriber/repository.go), [memória](../../internal/subscriber/memory_repository.go), [PostgreSQL](../../internal/subscriber/postgres_repository.go).
- Device: [handler](../../internal/device/handler.go), [entidade](../../internal/device/device.go), [service](../../internal/device/service.go), [repository](../../internal/device/repository.go), [memória](../../internal/device/memory_repository.go), [PostgreSQL](../../internal/device/postgres_repository.go).
- Session: [handler](../../internal/session/handler.go), [entidade](../../internal/session/session.go), [service](../../internal/session/service.go), [repository](../../internal/session/repository.go), [memória](../../internal/session/memory_repository.go), [PostgreSQL](../../internal/session/postgres_repository.go), [eventos internos](../../internal/session/event.go).
- Network: [cell.go](../../internal/network/cell.go), [ippool.go](../../internal/network/ippool.go).
- Telemetry: [handler](../../internal/telemetry/handler.go), [metrics](../../internal/telemetry/metrics.go), [event](../../internal/telemetry/event.go), [worker](../../internal/telemetry/worker.go).
- CLI: [main](../../cmd/simulator/main.go), [runner](../../cmd/simulator/runner.go), [client](../../cmd/simulator/client.go).
- Persistência: [postgres.go](../../internal/platform/postgres/postgres.go); migrações [subscribers](../../migrations/000001_create_subscribers.up.sql), [devices](../../migrations/000002_create_devices.up.sql), [sessions](../../migrations/000003_create_sessions.up.sql).
- UI atual: [App.tsx](src/App.tsx), [package.json](package.json). Não há `vite.config.ts` nesta versão isolada.
- Dependências: [go.mod](../../go.mod), declarando Go `1.27.0` e `pgx/v5 v5.11.0`; não foi verificada a instalação local dessas versões.
- Contexto: [ADR-001](../../docs/adr/ADR-001-modular-monolith.md), [Human Engineering Review](../../docs/engineering/HUMAN_ENGINEERING_REVIEW.md), [Implementation Master Plan](../../docs/engineering/NEXUS_CORE_LAB_IMPLEMENTATION_MASTER_PLAN.md), [AI Engineering Guidelines](../../docs/engineering/AI_ENGINEERING_GUIDELINES.md).

Planos e comentários não substituem implementação: a documentação antiga exclui frontend do escopo, mas o pedido atual aprova a interface. A revisão chama re-attach de idempotente; o service efetivamente cria uma nova sessão em cada re-attach. Nesta auditoria prevalece o comportamento do código.

## 1. Inventário HTTP real

Há **16 registros de método/rota**: health, telemetry, 6 de subscriber, 3 de device e 5 de session. A consulta de subscriber com IMSI é uma variante da mesma rota, detalhada separadamente abaixo.

Convenções: `S` = Subscriber; `D` = Device; `E` = Session; `T` = TelemetryResponse, descritos na seção 2. Respostas de sucesso são JSON diretamente, sem envelope `data`. Erros dos handlers usam `{ "error": string, "message": string, "code": string }`. `500 INTERNAL_SERVER_ERROR` representa falhas internas não mapeadas nos três domínios.

| METHOD | ROUTE | REQUEST | RESPONSE | HTTP STATUS / codes | DOMAIN ERRORS | SOURCE FILE |
|---|---|---|---|---|---|---|
| GET | `/health` | Sem body/query | `{status:"ok",service:"nexus-core-lab"}` | 200 | Nenhum; não testa DB | `internal/platform/httpserver/server.go` |
| GET | `/telemetry` | Sem body/query | T | 200; 500 `TELEMETRY_STORAGE_ERROR` | Erro de `ActiveCount` | `internal/telemetry/handler.go` |
| POST | `/api/v1/subscribers` | `{imsi:string,msisdn:string}` | S provisionado | 201; 400 `INVALID_PAYLOAD` / `INVALID_TELECOM_IDENTITY`; 409 `SUBSCRIBER_ALREADY_EXISTS`; 500 | `ErrInvalidIMSI`, `ErrInvalidMSISDN`, `ErrDuplicateIMSI`, `ErrDuplicateMSISDN` | `internal/subscriber/handler.go` |
| GET | `/api/v1/subscribers/{id}` | UUID interno no path | S | 200; 404 `SUBSCRIBER_NOT_FOUND`; 400 `MISSING_IDENTIFIER` no handler; 500 | `ErrSubscriberNotFound` | `internal/subscriber/handler.go` |
| GET | `/api/v1/subscribers` | Opcional `status`; sem IMSI | S[] ou `null` vazio no PostgreSQL | 200; 500 | Falha de repository; status desconhecido não tem erro de validação dedicado | `internal/subscriber/handler.go` |
| GET | `/api/v1/subscribers?imsi=...` | IMSI não vazio; tem precedência sobre status | **S único**, não array | 200; 404 `SUBSCRIBER_NOT_FOUND`; 500 | `ErrSubscriberNotFound` | Mesmo handler/registro anterior |
| POST | `/api/v1/subscribers/{id}/activate` | ID; nenhum body necessário | S | 200; 404 `SUBSCRIBER_NOT_FOUND`; 409 `SUBSCRIBER_ALREADY_DEACTIVATED` / `INVALID_STATE_TRANSITION`; 500 | `ErrSubscriberNotFound`, `ErrAlreadyDeactivated`, `ErrInvalidTransition` | `internal/subscriber/handler.go` |
| POST | `/api/v1/subscribers/{id}/suspend` | ID; opcional `{reason:string}` | S | 200; 404 `SUBSCRIBER_NOT_FOUND`; 409 `SUBSCRIBER_ALREADY_DEACTIVATED` / `INVALID_STATE_TRANSITION`; 500 | Mesmos erros de transição | `internal/subscriber/handler.go` |
| POST | `/api/v1/subscribers/{id}/deactivate` | ID; opcional `{reason:string}` | S | 200; 404 `SUBSCRIBER_NOT_FOUND`; 409 `SUBSCRIBER_ALREADY_DEACTIVATED` / `INVALID_STATE_TRANSITION`; 500 | Mesmos erros de transição; estado terminal | `internal/subscriber/handler.go` |
| POST | `/api/v1/devices` | `{subscriber_id:string,imei:string,technology:"LTE" ou "5G"}` | D | 201; 400 `INVALID_PAYLOAD` / `INVALID_DEVICE_DATA`; 404 `SUBSCRIBER_NOT_FOUND`; 422 `SUBSCRIBER_NOT_ACTIVE`; 409 `DEVICE_ALREADY_EXISTS`; 500 | `ErrInvalidIMEI`, `ErrInvalidTechnology`, `ErrMissingSubscriber`, `ErrSubscriberNotFound`, `ErrSubscriberNotActive`, `ErrDuplicateIMEI` | `internal/device/handler.go` |
| GET | `/api/v1/devices/{id}` | UUID interno | D | 200; 400 `MISSING_IDENTIFIER` no handler; 404 `DEVICE_NOT_FOUND`; 500 | `ErrDeviceNotFound` | `internal/device/handler.go` |
| GET | `/api/v1/devices?subscriber_id=...` | **subscriber_id obrigatório** | D[] ou `null` vazio no PostgreSQL | 200; 400 `MISSING_QUERY_PARAMETER`; 500 | Ausência de query; falha de repository | `internal/device/handler.go` |
| POST | `/api/v1/sessions/attach` | `{device_id:string,cell_id:string}` | E nova | 201; 400 `INVALID_PAYLOAD` / `MISSING_ATTACH_FIELD` / `CELL_NOT_FOUND`; 404 `DEVICE_NOT_FOUND`; 422 `DEVICE_NOT_ELIGIBLE`; 500 | `ErrMissingDeviceID`, `ErrMissingCellID`, `network.ErrCellNotFound`, `ErrDeviceNotFound`, `ErrDeviceNotEligible`; pool esgotado cai em 500 | `internal/session/handler.go` |
| POST | `/api/v1/sessions/{id}/handover` | ID da sessão; `{target_cell_id:string}` | E atualizada | 200; 400 `MISSING_IDENTIFIER` / `INVALID_PAYLOAD` / `TARGET_CELL_NOT_FOUND`; 404 `SESSION_NOT_FOUND`; 422 `SESSION_NOT_CONNECTED`; 500 | `network.ErrCellNotFound`, `ErrSessionNotFound`, `ErrSessionNotConnected` | `internal/session/handler.go` |
| POST | `/api/v1/sessions/{id}/detach` | ID da sessão; nenhum body necessário | E desconectada | 200; 400 `MISSING_IDENTIFIER`; 404 `SESSION_NOT_FOUND`; 500 | `ErrSessionNotFound` | `internal/session/handler.go` |
| GET | `/api/v1/sessions/{id}` | UUID da sessão, inclusive encerrada | E | 200; 400 `MISSING_IDENTIFIER`; 404 `SESSION_NOT_FOUND`; 500 | `ErrSessionNotFound` | `internal/session/handler.go` |
| GET | `/api/v1/sessions?device_id=...` | **device_id obrigatório** | **E ativa única**, não array | 200; 400 `MISSING_QUERY_PARAMETER`; 404 `ACTIVE_SESSION_NOT_FOUND`; 500 | `ErrSessionNotFound` para nenhuma sessão ativa | `internal/session/handler.go` |

### Particularidades que o consumidor precisa respeitar

- `GET /api/v1/devices` sem query não lista todos; `GET /api/v1/sessions` sem query também não. Não existem parâmetros implementados de paginação, ordenação, pesquisa geral, filtro por célula ou listagem de sessões encerradas.
- Listas PostgreSQL vazias são slices nil serializadas como `null`; memória retorna `[]`. Normalizar apenas uma resposta bem-sucedida de lista para `payload ?? []`. Não converter erro HTTP em lista vazia.
- Memória não garante ordem de subscribers; PostgreSQL ordena por `created_at ASC`, sem desempate explícito. Não utilizar posição na lista como identidade.
- O filtro de status de subscriber é repassado ao repository; um valor desconhecido tende a produzir lista vazia, não 400.
- `suspend` e `deactivate` só tentam ler motivo com `ContentLength > 0` e ignoram erro do decoder. Não há contrato de rejeição de JSON inválido nesses dois bodies; enviar JSON válido e verificar o objeto retornado.
- No attach, o service valida célula antes de dispositivo. `cell_id` vazio resulta em `CELL_NOT_FOUND`; `device_id` vazio com célula válida chega ao checker/repository, podendo resultar em 404 na memória ou erro SQL/500 no PostgreSQL. O mapeamento `MISSING_ATTACH_FIELD` existe no handler, mas não assegura esse resultado para todo campo ausente.
- IDs de caminho/query são strings sem validação UUID uniforme no handler. IDs malformados podem ser 404 em memória e 500 no PostgreSQL. Usar IDs retornados pela API; IMEI/MSISDN não são esses IDs.
- Os ramos `MISSING_IDENTIFIER` dos handlers não implicam rota aceita com segmento vazio: o ServeMux resolve primeiro o caminho.
- `GET` registrado também admite HEAD pelo ServeMux; não são endpoints de negócio adicionais. 404/405 do roteador não têm garantia do envelope JSON de erro dos handlers.
- O middleware conta todas as requisições que chegam a ele, inclusive o próprio polling de `/telemetry`, e devolve `X-Request-ID`. Logs têm método, caminho, status e duração, mas não há API de consulta desses logs.
- Não há middleware de autenticação ou CORS na composição auditada. Uma chamada do navegador entre porta 5175 e 8080 não é mesma origem. Planejar proxy de desenvolvimento e mesma origem em produção; a API não serve a interface hoje. Nenhum proxy foi implementado nesta etapa.

## 2. Modelos reais expostos

Identificadores telecom são **strings**, nunca números JavaScript. Timestamps Go são serializados como strings RFC3339, com fração variável. O domínio cria horários UTC; o consumidor deve interpretar o offset, sem depender de quantidade fixa de casas decimais.

### Subscriber

| Field name | JSON name | Tipo Go / JSON | Valores | Nullable/optional | Significado |
|---|---|---|---|---|---|
| ID | id | string / string | UUID v4 gerado | Não | Identidade interna |
| IMSI | imsi | string / string | Exatamente 15 dígitos | Não | Identidade telecom única |
| MSISDN | msisdn | string / string | 10–15 dígitos, primeiro 1–9, `+` opcional | Não | Número único; regex `^\+?[1-9]\d{9,14}$` |
| Status | status | Status / string | `PENDING_ACTIVATION`, `ACTIVE`, `SUSPENDED`, `DEACTIVATED` | Não | Estado de contrato |
| SuspensionReason | suspension_reason | string / string | Texto livre | Omitido quando vazio | Motivo da suspensão |
| DeactivationReason | deactivation_reason | string / string | Texto livre | Omitido quando vazio | Motivo da desativação |
| CreatedAt | created_at | time.Time / string | Timestamp | Não | Provisionamento |
| UpdatedAt | updated_at | time.Time / string | Timestamp | Não | Última atualização |

Provisionamento começa em `PENDING_ACTIVATION`. Activate aceita pending/suspended; ACTIVE é no-op de entidade. Suspend só parte de ACTIVE ou atualiza motivo de SUSPENDED. Deactivate aceita qualquer estado não terminal. DEACTIVATED rejeita novas transições, inclusive nova desativação. Activate limpa motivo de suspensão; deactivate não necessariamente o limpa. Não existe DELETE ou edição de IMSI/MSISDN por HTTP.

**Diferença de validação/persistência:** o regex aceita `+` seguido de 15 dígitos (16 caracteres), mas a migração usa `VARCHAR(15)` para MSISDN. Essa borda pode falhar no PostgreSQL e ser aceita em memória. Não corrigida nesta auditoria.

### Device

| Field name | JSON name | Tipo Go / JSON | Valores | Nullable/optional | Significado |
|---|---|---|---|---|---|
| ID | id | string / string | UUID v4 | Não | ID interno; não IMEI nem UE-01 |
| SubscriberID | subscriber_id | string / string | ID de Subscriber | Não | Vínculo com assinante |
| IMEI | imei | string / string | 15 dígitos | Não | Identidade de equipamento única |
| Technology | technology | Technology / string | `LTE`, `5G` | Não | Tecnologia cadastrada do dispositivo |
| Status | status | Status / string | `REGISTERED`, `INACTIVE` | Não | Cadastro, não conectividade |
| CreatedAt | created_at | time.Time / string | Timestamp | Não | Registro |
| UpdatedAt | updated_at | time.Time / string | Timestamp | Não | Atualização |

Registro exige subscriber ACTIVE. A operação disponível cria REGISTERED; não existe HTTP para transicionar a INACTIVE, editar ou remover. Um assinante pode ter vários dispositivos. Device não contém célula, IP ou apelido UE. Esses campos vêm de sessão ou são apresentação.

### Session

| Field name | JSON name | Tipo Go / JSON | Valores | Nullable/optional | Significado |
|---|---|---|---|---|---|
| ID | id | string / string | UUID v4 | Não | Identidade da sessão |
| DeviceID | device_id | string / string | ID de Device | Não | Equipamento |
| SubscriberID | subscriber_id | string / string | ID de Subscriber | Não | Assinante |
| CellID | cell_id | string / string | `CELL-SP-001`, `CELL-SP-002`, `CELL-SP-003` nas operações atuais | Não | Célula atual/última |
| IPAddress | ip_address | string / string | IPv4 virtual alocado | Não | IP da sessão, preservado no histórico após detach |
| Status | status | Status / string | `CONNECTED`, `DISCONNECTED` | Não | Conectividade |
| DisconnectReason | disconnect_reason | string / string | `VOLUNTARY_DETACH`, `STALE_DISCONNECT` | Omitido quando vazio | Razão de encerramento |
| AttachedAt | attached_at | time.Time / string | Timestamp | Não | Início |
| UpdatedAt | updated_at | time.Time / string | Timestamp | Não | Última atualização; não é timestamp de evento específico |
| ClosedAt | closed_at | *time.Time / string | Timestamp de encerramento | Ponteiro nil omitido, não `null` no JSON normal | Fim da sessão |

Re-attach aloca novo IP e cria novo ID, encerrando a sessão anterior por STALE_DISCONNECT. Não é retry idempotente: não repetir automaticamente attach após timeout sem reconciliar o estado. Handover mantém ID/IP; mesma célula é no-op sem evento. Detach repetido retorna sucesso sem nova liberação ou evento.

O checker de attach verifica existência e status REGISTERED do device. Não revalida o estado atual do subscriber nem compatibilidade LTE/5G com a célula. Suspender/desativar subscriber não desconecta automaticamente sessões existentes neste código.

### TelemetryResponse e objetos aninhados

Todos os campos abaixo são obrigatórios e sem `omitempty`; nenhum é nullable na resposta bem-sucedida.

| Field name | JSON name/path | Tipo Go / JSON | Valores | Significado |
|---|---|---|---|---|
| Service | service | string / string | `nexus-core-lab` | Serviço |
| Timestamp | timestamp | time.Time / string | UTC atual | Hora da construção do snapshot |
| RequestsTotal | requests_total | uint64 / number | >= 0 | Requests desde início do processo |
| Metrics | metrics | MetricsSnapshot / object | Objeto | Gauges e perdas |
| ActiveSessions | metrics.active_sessions | int64 / number | >= 0 em operação normal | `ActiveCount` do repository |
| ConnectedDevices | metrics.connected_devices | int64 / number | Mesmo valor de active_sessions | Derivado da invariante uma sessão ativa por device; não total cadastrado |
| DroppedEventsTotal | metrics.dropped_events_total | uint64 / number | >= 0 | Eventos descartados por saturação do canal |
| EventsTotal | events_total | EventsTotalSnapshot / object | Objeto | Contadores consumidos |
| Attach | events_total.attach | uint64 / number | >= 0 | ATTACH processados |
| CellHandover | events_total.cell_handover | uint64 / number | >= 0 | CELL_HANDOVER processados |
| Detach | events_total.detach | uint64 / number | >= 0 | DETACH processados |
| StaleDisconnect | events_total.stale_disconnect | uint64 / number | >= 0 | STALE_DISCONNECT processados |

Não existem campos `total_subscribers`, `total_devices`, `cells`, `ip_pool`, `recent_events`, `history`, `uptime`, `database_status` ou `storage_mode` nessa resposta. `OTHER` não é tipo de evento real; a quarta categoria correta é STALE_DISCONNECT. uint64 pode ultrapassar a precisão segura de number em JavaScript em execuções muito longas; não há codificação string para contadores hoje.

## 3. Matriz completa UI → Backend

**A — suportado diretamente:** endpoint entrega o dado/operação. **B — derivável:** cálculo/junção de dados HTTP reais. **C — parcialmente suportado:** informação relacionada insuficiente para a promessa atual. **D — não suportado:** não exposto por HTTP. Elementos puramente locais estão marcados D/local, sem implicar necessidade de ampliar backend.

Abreviações de endpoints: **SUB** = GET `/api/v1/subscribers`; **DEV(s)** = GET `/api/v1/devices?subscriber_id={s}`; **SES(d)** = GET `/api/v1/sessions?device_id={d}`; **TEL** = GET `/telemetry`; **HEALTH** = GET `/health`. As operações POST referem-se às rotas completas da seção 1.

### Overview e estrutura compartilhada

| UI element | Required data | Existing endpoint | Classe | Frontend transformation | Backend gap |
|---|---|---|---|---|---|
| Total Subscribers | Todos os assinantes | SUB sem filtro | B | Normalizar null; contar a lista completa | Sem count dedicado, mas não necessário no lab |
| Total Devices | Todos os devices | SUB + DEV(s) de cada assinante | B | União por ID e contagem | Sem listagem global; custo N+1 |
| Active Sessions KPI | Contagem CONNECTED | TEL | A | Ler metrics.active_sessions | Nenhum |
| Network Cells KPI | Catálogo completo | Nenhum; catálogo interno | D | Só manter como configuração demonstrativa conhecida | Catálogo não exposto |
| Total Events KPI | Contadores dos quatro tipos | TEL | B | Soma de events_total | Sem total histórico persistido |
| +12/+8/+21/+35% | Base histórica comparável | Nenhum | D | Remover na integração real | Histórico ausente |
| Topologia: devices conectados e conexões verdes | Sessões ativas, célula, device | SUB + DEV(s) + SES(d) | B | Junção por UUID; agrupar por cell_id | Sem leitura global eficiente |
| Topologia: antenas, nome/tecnologia, Online | Catálogo e saúde de antenas | Nenhum | C | Pode usar layout estático aprovado, identificado como demonstrativo | Sem catálogo HTTP; nenhuma saúde de rádio real |
| Topologia: IP | ip_address | SES(d) ou GET sessão por ID | A | Exibir apenas para sessão adequada | Nenhum para sessão conhecida |
| Topologia: handover curvo | Origem/destino/tempo do evento | POST handover fornece novo estado | C | Animar operação confirmada desta UI com estado anterior conhecido | Sem feed global ou caminho histórico |
| Active Sessions table | Device, MSISDN, célula, IP, duração | SUB + DEV(s) + SES(d) | B | IMEI de Device, MSISDN de Subscriber, demais de Session; tempo decorrido | Sem lista global; não é snapshot atômico |
| View all Sessions | Navegação para lista | Mesma composição | B | Navegação local e conjunto completo | Nunca considerar 5 exemplos como total |
| Recent Events: hora/tipo/device/details | Registros ordenados de eventos | Nenhum | D | Não reconstruir identidades por diferenças de contadores | Sem histórico/feed |
| View all Events | Lista histórica | Nenhum | D | Pode navegar, mas conteúdo real indisponível | Mesmo gap |
| Session Activity: active | Série temporal de contagem | TEL | B | Janela de observações por polling | Sem pontos anteriores à abertura |
| Session Activity: detached | Série de encerramentos | TEL events_total | C | Deltas por intervalo, rotulados como eventos; não estoque de sessões desconectadas | Sem série persistida/estoque global |
| Events by Type | Contagens e percentuais | TEL | B | Soma, porcentagens, cores consistentes; tratar total zero | Sem histórico de reinícios |
| IP Pool Usage: alocados/capacidade/disponíveis | Snapshot do IPPool | TEL só dá sessões | C | Sessões como aproximação explícita, nunca pool exato | AllocatedCount/capacidade não expostos |
| IPPool warm-up completed | Resultado/data/quantidade do warm-up | Nenhum | D | Não afirmar sucesso com texto fixo | Só log de startup |
| API bolinha e System Online | Disponibilidade observada da API | HEALTH | A para API; C para sistema inteiro | Atualizar por sucesso/falha/última leitura | Health não testa dependências |
| Database bolinha | Modo e disponibilidade de storage | Nenhum | D | Mostrar desconhecido/não disponível futuramente | Nenhum status DB HTTP |
| Data e hora da topbar | Relógio | TEL timestamp ou relógio local | B | Identificar origem e formatar | Sem necessidade de endpoint novo |
| Sidebar, ícones, footer, links, engrenagem | Apresentação e navegação | Nenhum necessário | D/local | Manter composição e rotas locais | Não é gap de backend |
| Mapa, rosa dos ventos, +/−, legenda | Apresentação/cartografia | Nenhum necessário | D/local | Preservar mapa atual e controles locais | Não exige Leaflet/API de mapas |

### Telas de domínio

| UI element | Required data | Existing endpoint | Classe | Frontend transformation | Backend gap |
|---|---|---|---|---|---|
| Subscribers: Subscriber/Status | MSISDN e status real | SUB | A | Não substituir estados por Active genérico | Nenhum |
| Subscribers: Device | Equipamentos vinculados | DEV(s) | B | Mostrar zero/um/vários IMEIs | UI atual presume vínculo único |
| Subscribers: Search | Texto/IMSI/status | SUB; variante imsi | B | Busca textual local; IMSI exato por API | Sem pesquisa geral/paginação server-side |
| Subscribers: Open details | Entidade e motivos/datas | GET `/api/v1/subscribers/{id}` | A | Modal com campos reais | Nenhum |
| Subscribers: provisionar/ativar/suspender/desativar, se aprovados depois | Requests e transições | POSTs de subscriber | A | Formulários, tratar conflitos e estado terminal | UI atual não implementa estes formulários |
| Devices: listagem global | Devices de todos assinantes | SUB + DEV(s) | B | Unir por UUID | Sem GET global |
| Devices: Device/IMEI/Status | UUID, IMEI, REGISTERED/INACTIVE | DEV(s) ou GET `/api/v1/devices/{id}` | A para atributos | UE-01 é alias local, preservar UUID | Alias persistido não existe |
| Devices: Cell/conectividade | Sessão ativa por device | SES(d) | B | 404 ACTIVE_SESSION_NOT_FOUND significa sem sessão | REGISTERED não significa conectado |
| Devices: Search/Open | Dados de cadastro | DEV(s), GET device por ID | B / A | Filtrar lista / consultar ID | Sem pesquisa IMEI HTTP dedicada |
| Devices: cadastro futuro | Subscriber ativo, IMEI, tecnologia | POST `/api/v1/devices` | A | Selecionar subscriber pelo UUID | Nenhum para operação existente |
| Sessions: tabela ativa e search | Sessões ativas, MSISDN/IMEI | SUB + DEV(s) + SES(d) | B | Junções e filtro local | N+1; sem snapshot conjunto |
| Sessions: duração | attached_at/closed_at | SES(d) ou GET sessão por ID | B | Ativa: agora−attached_at; encerrada: closed_at−attached_at | Não existe duration pronto |
| Sessions: Open | Sessão conhecida | GET `/api/v1/sessions/{id}` | A | Consultar UUID da sessão, não do device | Nenhum |
| Sessions: histórico de todas encerradas | Lista de sessões encerradas | Só GET por ID conhecido | C | IDs lembrados pelo cliente não são histórico completo | Sem consulta histórica de coleção |
| Network: mapa/legenda/controles | Cartografia e composição | Nenhum necessário | D/local | Manter versão aprovada | Nenhuma alteração de mapa necessária |
| Network: células, tecnologias, conexões, IP e handover | Catálogo + estado + eventos | SES(d), DEV(s), POST handover | C no conjunto | Usar matriz detalhada do Overview | Catálogo/feed ausentes; geografia demonstrativa |
| Events: Time/Event type/Device/Details | Evento com timestamp e entidades | Nenhum | D | Não preencher com estimativas | Sem registros de eventos por HTTP |
| Events: Search/Open | Coleção e identificador de evento | Nenhum | D | Só possíveis após fonte real ou como demo explícita | Sem API/histórico/ID de evento |
| Telemetry: Session Activity | Observações no tempo | TEL | B para active; C para detached atual | Janela local/deltas com legenda correta | Sem histórico persistido |
| Telemetry: Events by Type | Quatro contadores | TEL | B | STALE_DISCONNECT em vez de OTHER; total zero | Nenhum para contadores do processo |
| Telemetry: IP Pool Usage/warm-up | Estado do alocador | Nenhum específico | C / D | Não atribuir valores exatos a gauges de sessão | Pool não exposto |

### Simulator, operação e preferências

| UI element | Required data | Existing endpoint | Classe | Frontend transformation | Backend gap |
|---|---|---|---|---|---|
| Simulator: ATTACH | device_id/cell_id e sessão resultante | POST `/api/v1/sessions/attach` | A para ação individual | Exige seleção real e reconciliação, não simples texto | Não dispara CLI |
| Simulator: CELL_HANDOVER | session_id/target_cell_id | POST `/api/v1/sessions/{id}/handover` | A para ação individual | Usar sessão atual; confirmar resultado | Não dispara CLI |
| Simulator: DETACH | session_id | POST `/api/v1/sessions/{id}/detach` | A para ação individual | Atualizar estado após sucesso | Não dispara CLI |
| Simulator: iniciar runner concorrente/Hero Flow completo | Execução, progresso, resultado do CLI | Nenhum | D | Manter execução pelo operador no terminal | Sem controle HTTP de jobs |
| Simulator: log/Clear | Lista de operações desta UI | Respostas individuais | B/local | Registrar somente resultados observados; Clear limpa apenas visualização | Não é Recent Events global |
| System: Online | Saúde geral | HEALTH/TEL | C | Nomear estado observado e última atualização | Sem readiness geral/uptime/version |
| System: contagens Subscribers/Devices/Active Sessions | Dados reais | SUB + DEV(s), TEL | B / B / A | Reusar a mesma fonte do Overview | Células continuam sem HTTP |
| API Status: disponibilidade/service | Resposta health | HEALTH | A | Exibir status observado | Nenhum |
| API Status: requests/latência | requests_total; tempo de ida e volta | TEL / HEALTH | A / B | RTT medido no browser é percepção do cliente | Não há métricas de latência interna |
| API Status: mesmas contagens de domínio da tela atual | Entidades/gauge | SUB + DEV(s), TEL | B | São reais, porém não descrevem saúde da API | Evitar dashboard repetido sem significado |
| Database: status/motor/métricas/conexões | Diagnóstico storage | Nenhum | D | Não inferir PostgreSQL pelo health | Falta diagnóstico HTTP; browser nunca acessa DB |
| Database: contagens repetidas | Entidades/gauge | SUB + DEV(s), TEL | C como informação de DB | Não chamar de estatística PostgreSQL: pode ser memória | Modo de storage não exposto |
| Configuration: refresh interval | Preferência do browser | Nenhum | D/local | Futuramente aplicar ao polling e persistir localmente se aprovado | Não é configuração operacional |
| Configuration: notifications | Preferência local + eventos | Nenhum para feed | C/local | Pode reger avisos de ações locais | Notificações globais dependem de eventos |
| Configuration: Save preferences | Persistência da preferência | Nenhum | D/local | Hoje só estado desta tela; não afirmar gravação no servidor | Nenhuma API de configuração |

## 4. Auditoria especial do Overview

**Pode receber dados reais hoje, sem mudar backend:** total de subscribers; total de devices por agregação; contagem de sessões ativas; soma de eventos processados; distribuição por tipo; lista de sessões ativas por enumeração; identificação/IP/célula atual dos dispositivos; saúde observada da API; série de sessões ativas formada a partir da abertura da tela.

**Só parcialmente:** desenho completo da topologia (estado real sobre geografia demonstrativa); animação global de handover; série de encerramentos; ocupação do pool inferida por sessões. Essas diferenças precisam estar explícitas na futura integração.

**Não suportado por HTTP:** catálogo de células completo, saúde de cada antena, Recent Events global, histórico anterior de atividade, snapshot exato do pool, confirmação do warm-up, saúde/modo PostgreSQL e percentuais históricos de crescimento.

As cinco linhas visíveis podem ser uma prévia de uma lista maior, desde que o total venha da fonte completa e “View all” use essa mesma coleção. Não esconder falha parcial atrás de total menor. A aprovação visual não valida os números e estados mock como contratos.

## 5. Estratégia dos KPIs

Considere S assinantes, D dispositivos e A sessões ativas. Os custos abaixo descrevem o código/volume de consultas, não benchmarks.

| KPI | Fonte e cálculo corretos | Custo atual | Novo endpoint? |
|---|---|---|---|
| Total Subscribers | `length(SUB sem filtro)` | 1 HTTP, O(S) dados; PostgreSQL lê/ordena lista | Não para pequeno lab; count é otimização futura |
| Total Devices | União dos IDs de DEV(s) para **todos** os subscribers, inclusive inativos | 1+S HTTP, O(S+D) payload; consultas por FK indexada em PG | Não estritamente; leitura global é melhoria importante |
| Active Sessions | TEL.metrics.active_sessions | 1 HTTP; memória O(1), PG `COUNT(*) WHERE status='CONNECTED'`, custo dependente do plano/volume | Não |
| Network Cells | Internamente `len(network.ListCells()) = 3` | O(3) interno, inacessível ao browser | Sim para valor autoritativo HTTP; manter estático como configuração não é integração real |
| Total Events | attach + cell_handover + detach + stale_disconnect de um único TEL | Mesmo request TEL; 4 leituras atômicas além da consulta do gauge | Não; rotular desde início do processo/eventos processados |

Não substituir Total Devices por connected_devices. Não usar número de células atualmente ocupadas como total de células. Não somar dropped_events_total ao donut: o tipo dos descartados não é fornecido. Se o requisito for “todos os eventos realmente ocorridos”, os contadores atuais são parciais por perdas e reinícios.

Os percentuais `+12%`, `+8%`, `+21%`, `+35%` são mock. Recomenda-se removê-los no futuro modo real; não usar variação desde a abertura da tela como crescimento histórico sem mudar o significado explicitamente.

Reusar cache/fonte de dados entre KPI e lista. As leituras separadas não são uma transação: durante alterações pode haver diferença temporária entre gauge e lista. Exibir última atualização/falha parcial e reconciliar; não prometer igualdade instantânea impossível com os endpoints atuais.

## 6. Dados reais para a topologia

O catálogo interno é estático e sem latitude/longitude, cobertura, sinal ou estado online:

| ID real | Name | Region | Tech |
|---|---|---|---|
| CELL-SP-001 | Campinas Centro | Campinas - SP | LTE |
| CELL-SP-002 | Campinas Barão Geraldo | Campinas - SP | 5G |
| CELL-SP-003 | Campinas Cambuí | Campinas - SP | 5G |

`FindCell` valida attach/handover. `ListCells` existe como função Go, **não como endpoint**. `Cell` tem JSON tags, mas isso não o torna exposto por HTTP.

Sessões fornecem cell_id e ip_address; devices fornecem tecnologia cadastrada. UE-01 a UE-05 não existem como identidade persistida: se mantidos como aliases visuais, mapear de maneira estável a UUIDs, sem limitar artificialmente o total a cinco. A UI atual mostra IMEI sob Device ID; na integração, guardar UUID para operações e distinguir a apresentação.

O mapa atual de Campinas pode continuar intacto. Antenas e dispositivos posicionados nele representam **geografia visual demonstrativa**, não localização medida. Um handover é uma transição entre IDs válidos, não prova de deslocamento físico. O traço laranja permanente não deve sugerir um evento atual sem observação confirmada.

Para um conjunto global real hoje: listar subscribers → devices de cada um → sessão ativa de cada device. Associar por UUID, agrupar por célula e colocar marcadores no layout já aprovado. Não criar coordenadas “reais” a partir de alias ou inferir tecnologia da sessão a partir de cor.

## 7. Recent Events e pipeline de telemetry

O worker mantém um **canal bufferizado de 256 eventos**, não um histórico de 256 entradas. `Emit` é não bloqueante; saturação incrementa dropped_events_total. O worker consome os eventos e incrementa quatro contadores atômicos. Não guarda uma lista após consumo, não persiste e não expõe o canal ao navegador. No shutdown drena o buffer; emissões depois de fechado são descartadas sem incrementar esse contador.

O Event interno tem type, session_id, device_id, subscriber_id, cell_id e timestamp. Não contém event_id, célula de origem/destino separadas ou mensagem de detalhes. O adapter registra o horário de emissão; para handover, a sessão já contém a célula de destino. Os campos do Event **não são retornados por `/telemetry`**.

Logo, não é possível reconstruir corretamente “23:41:09 · CELL_HANDOVER · UE-01 · SP-001 → SP-003” a partir de contadores. Polling pode perder várias transições entre duas leituras. `updated_at` não identifica o tipo de transição, e o estado final não revela as células intermediárias.

As tabelas SQL guardam estados de sessões encerradas, não eventos de handover. Mesmo essas sessões não têm listagem histórica HTTP. Registrar respostas de ações desta UI pode formar um log local, mas não inclui CLI/outros clientes nem substitui Recent Events global.

**Gap comprovado:** feed de eventos recentes. Se aprovado para demonstração, a menor solução é retenção limitada no processo e leitura HTTP com identificação/ordem e política explícita de reinício/perdas. Para detalhes origem→destino, o evento precisa ser enriquecido na transição; somente guardar o Event atual não basta. Persistência histórica durável é requisito separado, não recomendação automática de banco/migração.

## 8. Session Activity

Não existe série temporal exposta ou persistida de telemetria. É possível montar uma **client-side observation window**: armazenar cada TEL.timestamp e metrics.active_sessions recebido enquanto a interface está aberta; limitar a janela ao período escolhido e indicar quando começou a coleta.

Não preencher os 30 minutos anteriores com dados inventados. Falhas e suspensão da aba são intervalos sem observação, não zero. Ao recarregar, a janela recomeça se não houver persistência local explicitamente implementada; persistência local também não seria histórico autoritativo do servidor.

Para encerramentos, calcular deltas de events_total.detach por intervalo; se incluir re-attach, somar o delta de stale_disconnect e rotular essa escolha. São **eventos observados por intervalo**, não quantidade atual de “detached sessions”. Não calcular sessões ativas como attach−detach: existem reinícios, perdas, stale_disconnect e estado persistido.

Contadores que diminuem indicam descontinuidade possível, exigindo reinício de baseline; não produzir valores negativos. A API não fornece process_id/start_time, então nem todo reinício pode ser detectado só pela comparação. Histórico persistido requer outra decisão e não é necessário para uma primeira janela observacional honesta.

## 9. Situação real do IPPool

| Aspecto | Implementação real |
|---|---|
| CIDR lógico | `10.45.0.0/16` |
| Total bruto | 65.536 endereços |
| Reservas | `10.45.0.0` rede; `10.45.0.1` gateway simulado; `10.45.255.255` broadcast simulado |
| Faixa nominal utilizável | `10.45.0.2` a `10.45.255.254`, inclusivos |
| Capacidade nominal | `65534 - 2 + 1 = 65.533` |
| AllocatedCount | `len(allocated)` protegido por mutex; método Go, não HTTP |
| Allocate | Prioriza reciclados em LIFO; depois avança nextHost |
| Release | Remove de allocated e adiciona a recycled; rejeita liberação inexistente/duplicada |
| Warm-up | Startup PostgreSQL consulta IPs de sessões CONNECTED e chama MarkAllocated; falha interrompe startup |
| HTTP | Nenhum snapshot específico; TEL fornece somente active_sessions/connected_devices |

Não reservar `.0` e `.255` de cada /24: o código opera um /16, com as três reservas globais acima. O número 256 pertence ao tamanho padrão do canal de eventos, não à capacidade do IPPool.

**Limitação importante do alocador:** MarkAllocated avança nextHost até depois do maior host marcado e não reconstrói os buracos livres anteriores. Após um warm-up com apenas `10.45.0.10` alocado, os hosts `.2` a `.9` não entram na fila de reciclagem. Por isso, capacidade nominal menos AllocatedCount não garante quantidade efetivamente alocável nesta implementação após restart. Esta observação vem da leitura de MarkAllocated/Allocate; não foi executado um teste novo nem aplicada correção.

Em operação estável de uma instância, há um IP por sessão ativa. Porém, attach reserva IP antes da persistência e detach libera depois dela; snapshots concorrentes podem diferir. Warm-up, falhas, reinícios e múltiplos processos também impedem tratar o gauge de sessões como leitura exata do alocador.

Se houver 17 alocados, a ocupação **nominal** seria aproximadamente `17 / 65533 × 100 = 0,02594%`, não 6,6%. Esse exemplo é cálculo explicativo, não estado real observado. Não manter a confirmação mock de “1 active session IP pre-allocated”: o resultado do warm-up só aparece em log e não é exposto.

## 10. Auditoria do simulator

Execução existente, a partir da raiz do repositório, pelo operador em terminal:

```powershell
go run ./cmd/simulator -api http://localhost:8080 -devices 5
```

Comando documentado, **não executado nesta auditoria**. Pressupõe API disponível; produz alterações reais de domínio.

| Flag/comportamento | Valor real |
|---|---|
| `-api` | Default `http://localhost:8080` |
| `-devices` | Default 5; permitido 1 a 100; inválido encerra com código 1 |
| Timeout HTTP | 10 segundos configurados no main; não há flag de timeout |
| Concorrência | Uma goroutine por virtual device |
| Cancelamento | Contexto cancelado por interrupt/SIGTERM |
| Identidades | Geradas com seed de tempo e índice; únicas dentro da execução, não garantia global entre execuções simultâneas |
| Relatório | Solicitados, concluídos, falhas, duração, taxa de sucesso e snapshot final TEL |
| Falha por device | Para naquele passo; demais continuam; não há rollback global/limpeza automática |

Hero Flow por device: provisionar Subscriber → ativar → registrar Device **5G** → attach em **CELL-SP-001** → handover para **CELL-SP-002** → detach. O fluxo comprova que o sistema aceita um device 5G em célula LTE; não impor compatibilidade inexistente no frontend.

O client usa HTTP/JSON e DTOs próprios, sem importar domínios. São seis requests de escrita por device bem-sucedido mais a consulta final de telemetry; timeout de contexto dessa consulta é 3 segundos. Não espera garantia de drenagem do worker antes do snapshot final, portanto os contadores podem estar ligeiramente atrasados.

Não existe servidor/rota para iniciar, parar ou acompanhar o runner pela API. O browser não deve executar o binário nem comandos locais. As três ações individuais de sessão têm endpoints e podem futuramente compor controles com seleção de IDs reais, mas isso é diferente de disparar o CLI concorrente. Recomenda-se manter o CLI como ferramenta do operador e não criar um gerenciador de jobs apenas para preencher essa tela.

Ao final de um fluxo completo bem-sucedido, as sessões criadas estão desconectadas; não esperar cinco devices conectados permanentemente no mapa. Os subscribers e devices permanecem cadastrados. O polling pode não observar as etapas intermediárias rápidas.

## 11. System / API Status / Database

`/health` devolve status ok sem consulta de dependências. É sinal de resposta do serviço HTTP, não de integridade do banco ou das antenas. `/telemetry` consulta ActiveCount no repository e falha com 500 TELEMETRY_STORAGE_ERROR se essa consulta falhar, mas uma resposta 200 também ocorre em modo memória.

`DATABASE_URL` vazio seleciona repositories em memória; preenchido seleciona PostgreSQL. Startup PostgreSQL faz ping com timeout, valida `schema_migrations` (versão máxima pelo menos 3) e executa warm-up. Isso não equivale a monitoramento contínuo, nem verifica toda a estrutura de cada tabela. O pool SQL usa defaults de 25 conexões abertas/ociosas e lifetime 5 minutos; esses valores não são métricas HTTP.

System pode apresentar dados de domínio reais já disponíveis, sem fingir uptime, CPU, RAM, versão, prontidão global ou saúde de rádio. API Status pode apresentar resposta de health, última observação, requests_total e RTT medido pelo browser, deixando claro que RTT não é tempo interno de processamento.

Database hoje não tem dados suficientes para um painel real. Não concluir “PostgreSQL online” por bolinha verde, sucesso de health ou existência local de migrações. Uma futura leitura mínima de modo de storage e readiness do backend só vale se essa informação continuar necessária na UI. Credenciais, DSN e conexão SQL jamais devem ir para o navegador.

## 12. Configuration

Não há API de configuração operacional. PORT e DATABASE_URL são configuração de startup; não são editáveis pelo frontend. O catálogo e o CIDR também não possuem configuração HTTP.

A tela atual contém apenas estado React de intervalo, checkbox e confirmação de salvamento. Esse estado é perdido ao desmontar a Workspace; não configura polling ou notificações reais nem persiste no servidor.

Recomendação futura: manter somente preferências de exibição realmente aplicadas no cliente, ou retirar a tela se não oferecer utilidade. Não criar CRUD de configuração para justificar a existência do menu. Qualquer decisão visual fica para a revisão; a tela atual permanece intacta.

## 13. Gaps comprovados e menor mudança possível

As mudanças abaixo são **opções para revisão**, não implementação aprovada. P0 significa necessário para o recorte principal escolhido, não justificativa para implementar todos os itens.

| Gap | UI affected | Por que a API atual não supre | Menor mudança possível, se aprovada | Vale implementar? / prioridade |
|---|---|---|---|---|
| Acesso browser entre origens sem CORS/proxy | Todas as integrações | API não configura CORS e frontend isolado não tem proxy configurado | Proxy de desenvolvimento e mesma origem no deploy; sem mudança de domínio | Sim, decisão de transporte P0; backend pode ficar intacto |
| Lista global de devices e sessões | Devices/Sessions/Overview/Network | Endpoints obrigam subscriber_id/device_id | Primeiro usar agregação limitada; depois ampliar leitura de coleção com contrato explícito e limites | P1 no lab; torna-se P0 se volume/latência inviabilizar agregação |
| Catálogo não exposto | Network Cells/topologia | ListCells só existe em Go | Leitura HTTP do catálogo estático, sem CRUD | P1; útil para eliminar duplicação autoritativa |
| Recent Events inexistente | Events/Overview | Worker descarta conteúdo após contabilizar | Retenção limitada em memória e consulta ordenada; enriquecer origem/destino para handover | P1 se tabela global real for exigida; sem DB inicialmente |
| Histórico de sessões sem listagem | Sessions históricas | GET por ID não descobre sessões encerradas | Consulta de coleção limitada/filtrada no repository e handler | P2; não necessário para lista ativa |
| Snapshot exato do pool ausente | IP Pool Usage | AllocatedCount/CIDR/reservas não expostos | Snapshot consistente do alocador, publicado no canal HTTP escolhido | P1 se card exato for mantido |
| Buracos do pool após warm-up | Disponíveis e esgotamento | nextHost avança sem recuperar hosts livres anteriores | Revisão localizada da reconstrução/alocação e teste de restart com lacunas | P1; não prometer disponibilidade exata antes de resolver |
| Warm-up sem estado HTTP | Aviso warm-up | Existe somente log de startup | Preferir remover aviso; opcionalmente expor resultado armazenado | DROP para aviso decorativo; P2 se diagnóstico necessário |
| Modo/saúde de storage ausentes | Database e bolinha | Health não consulta DB; telemetry não identifica modo | Diagnóstico mínimo de modo e prontidão, sem credenciais | P1 se indicador DB permanecer operacional |
| Série histórica persistida ausente | Session Activity | Só snapshot e contadores voláteis | Janela local primeiro; persistência somente com requisito histórico explícito | P2; não necessária para começo |
| Growth percentages sem base | KPIs | Sem período anterior comparável | Remover percentuais mock | DROP |
| Controle HTTP do CLI ausente | Simulator | CLI é cliente, não serviço de jobs | Manter CLI externo; ações individuais usam endpoints existentes | DROP para botão de executar binário |
| Configuração operacional ausente | Configuration | Sem contrato de leitura/escrita | Preferência local ou retirada | DROP para novo subsistema de configuração |
| Coordenadas/saúde física ausentes | Mapa/Online nas células | Catálogo sem lat/lon/estado | Manter cartografia demonstrativa aprovada | DROP para inventar localização/monitoramento físico |
| MSISDN aceita tamanho maior que coluna | Provisionamento na futura integração | Regex aceita até 16 caracteres com `+`; coluna tem 15 | Alinhar contrato/persistência após decisão; cobrir caso de borda | P1 de consistência; não executar migração agora |
| Erros de entrada variam por repository | Formulários/ações | UUID não validado uniformemente; motivo ignora decode; ordem de validação attach | Definir e testar comportamento de entrada nos handlers/services | P1; não bloqueia consumo de IDs válidos |

## 14. Priorização recomendada

- **P0:** aprovar recorte de integração; definir acesso mesma origem/proxy; tipar contratos exatos; distinguir carregamento, erro, vazio e dado desatualizado; usar IDs reais e fonte compartilhada de KPI/lista. São principalmente decisões e trabalho de frontend futuro.
- **P1:** leituras globais se necessário, catálogo, eventos recentes se exigidos, pool exato, diagnóstico mínimo DB e consistência dos casos de entrada/persistência. Fazer apenas itens aceitos na revisão.
- **P2:** histórico persistido, descoberta de sessões encerradas, diagnóstico de reinício/warm-up e otimizações comprovadas por volume.
- **DROP:** crescimentos inventados, OTHER fictício, saúde física de antenas, execução de binário pelo browser, configuração operacional artificial e aviso fixo de warm-up.

## 15. Menor conjunto recomendado de alterações backend

**Para iniciar a integração principal de cadastro, consulta e ciclo de sessão: zero mudanças obrigatórias de backend**, desde que o lab aceite agregação de consultas, mesma origem/proxy e exibição honesta dos componentes ainda sem dados. A API já implementa as operações de domínio necessárias ao Hero Flow.

Para tornar todo o Overview operacional conforme sua intenção, o conjunto mínimo condicionado à revisão é:

1. Expor leitura do catálogo estático.
2. Expor snapshot real do pool, após tratar a semântica de disponibilidade pós-warm-up.
3. Reter e expor eventos recentes limitados, com origem/destino quando necessários aos detalhes.
4. Expor diagnóstico mínimo de storage se o indicador de Database continuar obrigatório.
5. Ampliar consultas globais de devices/sessões apenas se a agregação atual for inadequada para o volume ou para a experiência exigida.

Não é necessário criar um endpoint para cada card. A escolha de estender uma resposta operacional ou acrescentar leituras pequenas precisa de contrato revisado; este relatório não atribui URLs fictícias a essas propostas. Nenhuma nova tabela é necessária para catálogo, diagnóstico ou janela recente em memória. Histórico durável é outro escopo.

## 16. Ordem incremental recomendada de integração

1. HUMAN REVIEW deste documento: aprovar recorte, significado dos indicadores e gaps que realmente precisam ser atendidos.
2. Definir ambiente de teste, transporte mesma origem e modo de armazenamento. Não usar um banco com dados que não possam ser alterados para ensaios do Hero Flow.
3. Integrar leituras health/telemetry e seus estados de falha, preservando aparência.
4. Integrar Subscribers e seus detalhes; depois operações de ciclo de vida, se aprovadas.
5. Integrar Devices, vínculos e contagem completa. Cadastro usa subscriber UUID ativo.
6. Integrar sessões por device, detalhes e ações individuais. Reconciliar re-attach e retries; a tabela e o mapa usam a mesma fonte.
7. Alimentar Overview e Network com estado real sobre o mapa atual. Células e geografia permanecem claramente identificadas conforme suporte disponível.
8. Substituir gráficos mock por contadores e janela observacional, sem preencher passado inexistente.
9. Implementar somente gaps backend aprovados e então conectar Recent Events/pool/diagnóstico.
10. Validar Hero Flow com operador usando CLI e observar também o estado final desconectado; testar falhas/reinício/coleções vazias.
11. Comparar visualmente em 1920×1080, 1440×900 e 1366×768, considerando a área útil do navegador. Preservar sidebar, diagonal, topbar, mapa, densidade e espaçamentos aprovados.

Esses passos são planejamento; nenhum começou nesta auditoria.

## 17. Arquivos frontend que futuramente precisariam mudar

| Arquivo | Motivo futuro |
|---|---|
| `web/reference-version/src/App.tsx` | Trocar fontes mock por contratos reais; IDs, listas, estado de carregamento/erro, comandos, gráficos, pool e status conforme escopo aprovado |
| `web/reference-version/src/main.tsx` | Somente se necessária composição de estado compartilhado; não obrigatória para simples integração |
| Novo módulo pequeno de contratos/HTTP, nome a definir | Tipos e tratamento central de erros/listas; não existe nem foi criado nesta etapa |
| Configuração Vite/proxy futura, se escolhida | Desenvolvimento mesma origem; não existe vite.config.ts hoje |
| `web/reference-version/src/styles.css` e `comparison.css` | Apenas se estados reais/textos exigirem adaptação pontual; não redesenhar layout |
| `web/reference-version/src/ReferenceArt.tsx` | Somente se pontos de dados precisarem ligação com estado; arte aprovada não exige alteração por si |

`public/maps/campinas.jpg`, atribuições e mapa atual devem permanecer. Nenhuma nova biblioteca de mapas é necessária. Não é necessário modificar a versão original em `web/src/` para esse trabalho isolado.

## 18. Arquivos backend que somente se necessário precisariam mudar

| Necessidade aprovada | Arquivos candidatos |
|---|---|
| Catálogo HTTP | `internal/network/cell.go` como fonte existente; pequeno handler no módulo; registro em `cmd/api/main.go`; testes de contrato |
| Coleções globais | `internal/device/handler.go`, `service.go`, `repository.go`, implementações memory/postgres; equivalentes em `internal/session/`; testes correspondentes |
| Eventos recentes | `internal/telemetry/worker.go`, `event.go`, `handler.go`; `internal/session/event.go`/`service.go` e adapter de `cmd/api/main.go` se enriquecer handover; testes |
| Pool exato e warm-up | `internal/network/ippool.go`, `ippool_test.go`; composição/warmUpIPPool de `cmd/api/main.go`; handler/DTO escolhido para expor snapshot |
| Storage readiness | `cmd/api/main.go`, `internal/platform/httpserver/` e uso controlado de `internal/platform/postgres/`; testes |
| Consistência de validação | Handlers/services dos domínios afetados e seus testes; eventual migração de MSISDN apenas após decisão explícita |

Não há motivo demonstrado para alterar `cmd/simulator` para a primeira integração. Não editar migrações já aplicadas para introduzir mudanças silenciosas. Não adicionar dependência Go, WebSocket, SSE, broker ou CRUD de antenas como consequência automática desta auditoria.

## 19. Riscos, limites e evidências de testes

### Riscos de integração

1. **Agregação não atômica:** 1+S+D consultas para descobrir todas as sessões ativas; limitar concorrência, evitar polling sobreposto, indicar falha parcial e reconciliar. Isso é viável para lab pequeno, não promessa de escala.
2. **Identidades diferentes:** UUID, IMEI, IMSI e MSISDN têm papéis distintos. Não enviar UE-01, SP-001 abreviado ou IMEI em campos que esperam UUID/CELL-SP-001.
3. **Estado de cadastro ≠ conectividade:** REGISTERED pode não ter sessão; subscriber ACTIVE pode ter vários devices. Não deduzir cardinalidade um-para-um entre subscriber e device.
4. **Memória versus PostgreSQL:** coleções vazias e IDs malformados podem responder diferentemente. Memória perde entidades no restart; PG mantém entidades, mas não contadores de telemetry.
5. **Contadores assíncronos:** não são um log completo; podem atrasar, perder eventos e reiniciar. Gauges vêm do repository, não desses contadores.
6. **Re-attach não idempotente:** retry cego pode gerar outra sessão/stale_disconnect. Handover para mesma célula e detach repetido são no-op; não inflar contadores na UI manualmente.
7. **Pool por processo:** alocador e locks de device vivem no processo. Não pressupor segurança operacional de várias instâncias compartilhando o mesmo DB; índices únicos podem rejeitar conflitos, mas não constituem coordenação distribuída do pool.
8. **Disponibilidade do pool:** capacidade nominal não representa todos os endereços imediatamente alcançáveis após warm-up com lacunas; não expor “Available” como garantia sem tratamento.
9. **Ausência de autenticação:** a API expõe ações de escrita sem autenticação nesta composição. Não publicar como serviço operacional protegido sem decisão separada. Esta auditoria não implementa autenticação.
10. **Scope creep:** API de mapa não resolve falta de estado telecom; não substituir mapa aprovado nem criar subsistemas para preencher menus.
11. **Fluxo rápido do CLI:** pode terminar antes de um polling capturar conexão. Ausência de device conectado após o CLI não significa falha se detach concluiu.
12. **Entrada/persistência:** tamanho de MSISDN, UUID malformado e decode de reason exigem validação futura de contrato; não esconder 500 como “nenhum registro”.

### Testes existentes consultados

- [subscriber/handler_test.go](../../internal/subscriber/handler_test.go): provisionamento 201, duplicidade 409, consultas, transições e estado terminal. [subscriber_test.go](../../internal/subscriber/subscriber_test.go): validação e transições.
- [device/handler_test.go](../../internal/device/handler_test.go): registro, duplicidade, subscriber inexistente/inativo, IMEI inválido e query obrigatória.
- [session/handler_test.go](../../internal/session/handler_test.go): attach, elegibilidade, célula inválida, handover, detach repetido, busca por ID/device e query ausente.
- [session/session_test.go](../../internal/session/session_test.go): ciclo, re-attach, concorrência e `TestSession_EventEmitter`, que verifica no-op sem emissão, stale_disconnect e detach sem duplicação.
- [telemetry/handler_test.go](../../internal/telemetry/handler_test.go): campos do snapshot e TELEMETRY_STORAGE_ERROR; [worker_test.go](../../internal/telemetry/worker_test.go): tipos, saturação, drain e shutdown.
- [network/ippool_test.go](../../internal/network/ippool_test.go): catálogo, alocação/reciclagem, dupla liberação, concorrência e MarkAllocated. Não comprova reconstrução de todos os buracos livres pós-restart.
- [simulator/runner_test.go](../../cmd/simulator/runner_test.go): Hero Flow em servidor fake, execução concorrente, isolamento de falhas, cancelamento e identidades. Isso não é validação de uma instância PostgreSQL real.
- Há testes PostgreSQL nos três módulos, incluindo ciclo, ActiveCount, índices únicos parciais e concorrência em session. As migrações impõem unicidade de IMSI/MSISDN/IMEI, FKs, estados válidos e uma sessão CONNECTED por device/IP.

Os testes foram usados como evidência de intenção/contrato e lidos em conjunto com o código; **não foram executados nesta etapa**. Não há alegação de “todos os testes passaram”, benchmark, captura de API real ou validação visual nova.

## 20. Critérios de aceite para a próxima etapa

- [ ] Revisão humana aprova este mapa de contratos e o recorte de integração antes de implementar.
- [ ] UI aprovada, versão original e mapa preservados; alterações restritas ao escopo autorizado.
- [ ] Transporte browser/API funciona sem expor credenciais de DB ou exigir acesso SQL pelo navegador.
- [ ] Subscriber/Device/Session preservam enums, IDs, JSON names e campos opcionais reais.
- [ ] Coleções `null` bem-sucedidas são tratadas; erros não viram zero nem lista vazia.
- [ ] KPIs usam conjuntos completos e as mesmas fontes das listas; prévia de cinco linhas é explicitamente prévia.
- [ ] Growth percentages mock não aparecem como métricas reais.
- [ ] Active Sessions usa gauge do repository; Total Devices não usa connected_devices.
- [ ] Recent Events só é global/real se houver fonte de eventos adequada; log local é identificado como local.
- [ ] Série temporal não inventa passado; deltas, falhas e reinícios têm semântica explícita.
- [ ] Pool não usa 256; valores exatos dependem de snapshot real e semântica de disponibilidade validada.
- [ ] Células usam IDs completos; localização visual não é apresentada como GPS real.
- [ ] Health não acende Database como confirmação de PostgreSQL saudável.
- [ ] Ações de sessão usam IDs reais, não geram sucesso antes da resposta e reconciliam timeouts/re-attach.
- [ ] CLI continua ferramenta externa; browser não executa binários ou comandos locais.
- [ ] Configuração local é aplicada de fato e não é apresentada como configuração de servidor.
- [ ] Contratos são testados com vazio, erro, transição inválida, detach repetido, re-attach e reinício, em ambiente apropriado aprovado.
- [ ] Validação visual posterior cobre 1920×1080, 1440×900 e 1366×768 sem redesenhar a composição aprovada.

**HUMAN REVIEW — PARAR AQUI.** Este Gate entrega somente a auditoria. Nenhuma proposta deste documento autoriza implementação, alteração de backend, banco, UI ou mapa.
