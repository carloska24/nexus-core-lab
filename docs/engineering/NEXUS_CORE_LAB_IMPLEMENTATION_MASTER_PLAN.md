# NEXUS CORE LAB — IMPLEMENTATION MASTER PLAN

**Documento de Engenharia & Roadmap de Implementação**  
**Base:** [ADR-001](docs/adr/ADR-001-modular-monolith.md) | [AI_ENGINEERING_GUIDELINES.md](docs/engineering/AI_ENGINEERING_GUIDELINES.md) | [HUMAN_ENGINEERING_REVIEW.md](HUMAN_ENGINEERING_REVIEW.md)  
**Status:** Proposto para Aprovação  
**Arquitetura:** Monólito Modular em Go  

---

## 1. Visão Geral e Filosofia de Execução

O **NEXUS Core Lab** é um laboratório de engenharia backend em Go focado em conceitos fundamentais de redes 4G/5G, concorrência, sistemas distribuídos e observabilidade pragmática. 

Este plano estabelece a sequência de fatias verticais incrementais para levar o repositório do seu estado atual (apenas infraestrutura HTTP básica e `/health`) até o **Hero Flow** completo e a persistência relacional, respeitando rigorosamente a diretriz central: **a complexidade deve ser conquistada por requisitos concretos**.

### O Hero Flow Completo
```text
[Assinante Provisionado & Ativado]
               │
               ▼
   [Dispositivo Registrado (IMEI)]
               │
               ▼
[Attach: Sessão Aberta + IP Alocado (10.45.X.X)] ──► [Evento via Canal Go] ──► [Telemetria Atualizada]
               │
               ▼
  [Handover: Célula A ──► Célula B]              ──► [Evento via Canal Go] ──► [Telemetria Atualizada]
               │
               ▼
[Detach: Sessão Fechada + IP Liberado]           ──► [Evento via Canal Go] ──► [Telemetria Atualizada]
               │
               ▲
[Simulation Runner CLI: Múltiplas Goroutines Concorrentes executando o fluxo acima]
```

---

## 2. Matriz de Milestones

| Milestone | Nome do Marco | Entrega Principal | Persistência |
| :---: | :--- | :--- | :--- |
| **M1** | **Subscriber Registry** | Entidade, máquina de estados, regras de unicidade telecom e endpoints de ação. | In-Memory (`sync.RWMutex`) |
| **M2** | **Device Domain** | Associação 1:N com assinante ativo, validação de IMEI e integridade de hardware. | In-Memory (`sync.RWMutex`) |
| **M3** | **Network & Session Core** | Topologia estática de células, pool de IPs CIDR, Attach, Handover e Detach. | In-Memory (`sync.RWMutex`) |
| **M4** | **Telemetry & Async Events** | Propagação de eventos via canais Go, worker de métricas e `GET /telemetry`. | Em memória (`sync/atomic`) |
| **M5** | **Simulation Runner CLI** | Executável concorrente (`cmd/simulator`) simulando dispositivos e tráfego. | Conexão HTTP com a API |
| **M6** | **PostgreSQL & Migrations** | Repositório PostgreSQL, Docker Compose e migrações SQL sem tocar no domínio. | PostgreSQL 16 |
| **M7** | **Documentação & Portfólio** | README técnico, ADR-002, ADR-003 e seção de colaboração com IA. | — |

---

## 3. Detalhamento dos Milestones

---

### MILESTONE 1 — Subscriber Registry (Núcleo de Assinantes)

#### Objetivo
Estabelecer o primeiro domínio de negócio do laboratório, implementando a entidade de assinante, a máquina de estados telecom e seus endpoints de ação semântica, isolado de infraestrutura externa.

#### Comportamento Esperado do Domínio
- Identidade telecom única: `imsi` (15 dígitos numéricos) e `msisdn` (formato E.164 simulado).
- ID interno único via UUID v4 gerado pelo sistema.
- Máquina de estados:
  - `PENDING_ACTIVATION` (estado inicial padrão);
  - `ACTIVE` (habilitado para futuras sessões);
  - `SUSPENDED` (bloqueado; aceita motivo `reason` opcional; pode retornar para `ACTIVE`);
  - `DEACTIVATED` (estado terminal irreversível; rejeita qualquer transição posterior).
- Unicidade estrita: duplicatas de IMSI ou MSISDN retornam erro explícito de domínio mapeado para HTTP `409 Conflict`.
- Sem remoção física (`DELETE` inexistente).

#### Arquivos e Pacotes
```text
internal/subscriber/
├── subscriber.go          # Entidade Subscriber, tipos de Status, invariantes e transições
├── repository.go          # Interface SubscriberRepository (Save, FindByID, FindByIMSI, ExistsByIMSIOrMSISDN, List)
├── memory_repository.go   # Implementação in-memory com sync.RWMutex e índices por ID e IMSI
├── service.go             # Casos de uso: Provision, Activate, Suspend, Deactivate, Get, List
├── handler.go             # Rotas HTTP e handlers REST mapeados para JSON padronizado
├── subscriber_test.go     # Testes unitários puros de domínio e transições de estado
└── handler_test.go        # Testes de integração HTTP (httptest)
cmd/api/main.go            # Composição do módulo Subscriber no roteador
```

#### Dependências e Tecnologias
- Nenhuma dependência externa. Uso estrito da biblioteca padrão do Go (`net/http`, `sync`, `encoding/json`, `time`, `crypto/rand` para UUIDs básicos ou `google/uuid` apenas se justificado).

#### Testes Necessários
- **Testes de Domínio:** Invariantes de transição (ex: tentar ativar um assinante desativado deve falhar com erro de domínio tipado; suspender com motivo; validação de 15 dígitos do IMSI).
- **Testes de Concorrência do Repositório:** Gravações e leituras simultâneas com `go test -race` garantindo ausência de data races.
- **Testes HTTP:** Provisionamento com sucesso (`201 Created`), duplicidade (`409 Conflict`), busca inexistente (`404 Not Found`), ativação (`200 OK`) e transição inválida (`409 Conflict`).

#### Critérios de Aceitação & Não-Objetivos
- [ ] Endpoints `POST /api/v1/subscribers`, `GET /api/v1/subscribers/{id}`, `GET /api/v1/subscribers`, `POST /api/v1/subscribers/{id}/activate`, `POST /api/v1/subscribers/{id}/suspend`, `POST /api/v1/subscribers/{id}/deactivate` funcionando.
- [ ] `go test -race ./...` e `go vet ./...` passando com 100% de sucesso.
- [ ] **Não-Objetivos:** Sem banco de dados PostgreSQL ainda; sem dados comerciais ou CRM (nomes, CPF, endereços, planos pré/pós).

---

### MILESTONE 2 — Device Domain (Gestão de Equipamentos)

#### Objetivo
Implementar a modelagem de equipamentos físicos virtuais (`Device`), vinculando-os com assinantes ativos em relação de cardinalidade 1:N com unicidade de hardware via IMEI.

#### Comportamento Esperado do Domínio
- Atributos do dispositivo: `id` (UUID), `subscriber_id` (UUID), `imei` (15 dígitos numéricos únicos globalmente), `technology` (`LTE` ou `5G`), `status` (`REGISTERED`, `INACTIVE`), `created_at`, `updated_at`.
- Invariantes de Associação:
  - Um dispositivo só pode ser registrado se o assinante correspondente existir e estiver no estado `ACTIVE` (rejeita se `PENDING_ACTIVATION`, `SUSPENDED` ou `DEACTIVATED`).
  - Cada IMEI deve ser único no sistema (tentativa de reuso de IMEI resulta em `409 Conflict`).
- Desacoplamento arquitetural: o pacote `device` consulta o assinante através de uma interface consumidora local pequena (`SubscriberChecker`), sem importar structs de repositório do pacote `subscriber`.

#### Arquivos e Pacotes
```text
internal/device/
├── device.go              # Entidade Device, enums de tecnologia (LTE, 5G) e validações
├── repository.go          # Interface DeviceRepository + implementação in-memory thread-safe
├── service.go             # Casos de uso: RegisterDevice, GetDevice, ListBySubscriber
├── handler.go             # Endpoints HTTP REST
├── device_test.go         # Testes de invariantes e validações
└── handler_test.go        # Testes de contrato HTTP
cmd/api/main.go            # Composição e injeção do SubscriberChecker no DeviceService
```

#### Dependências e Tecnologias
- Nenhuma dependência externa adicional. Padrão Go idiomático.

#### Testes Necessários
- Tentativa de registrar dispositivo para assinante inexistente (`404`) ou suspenso/pendente (`422` ou `409`).
- Validação de formato de IMEI (15 dígitos numéricos).
- Teste de unicidade de IMEI com concorrência (`-race`).

#### Critérios de Aceitação & Não-Objetivos
- [ ] `POST /api/v1/devices` e `GET /api/v1/devices/{id}` implementados e testados.
- [ ] Associação garantida apenas com assinantes ativos.
- [ ] **Não-Objetivos:** Sem sessões ou alocação de IP ainda; sem CRUD de modelos de fabricantes ou marcas comerciais.

---

### MILESTONE 3 — Network & Session Core (Plano de Conexão e Mobilidade)

#### Objetivo
Implementar a camada central de conectividade da rede simulada: topologia de células estática, alocador de endereçamento IP dinâmico e o ciclo de vida completo de sessões (`ATTACH`, `HANDOVER`, `DETACH`).

#### Comportamento Esperado do Domínio
- **Topologia de Rede Estática:** Células pré-definidas em memória sem CRUD desnecessário:
  - `CELL-SP-001` (Campinas Centro)
  - `CELL-SP-002` (Campinas Barão Geraldo)
  - `CELL-SP-003` (Campinas Cambuí)
- **Pool de IP Virtual Privado:** Alocador thread-safe gerenciando um bloco CIDR fictício (ex: `10.45.0.0/16`). Aloca IP sequencial no `ATTACH` e devolve ao pool no `DETACH`.
- **Máquina de Estados da Sessão:**
  - `ATTACH`: Dispositivo registrado solicita conexão na célula X. Valida assinante ativo, valida dispositivo registrado, aloca IP virtual, define status `CONNECTED`.
  - Concorrência/Idempotência: Se o dispositivo já possuir uma sessão ativa, a anterior é encerrada forçadamente com log `STALE_DISCONNECT` e a nova assume o controle.
  - `HANDOVER`: Dispositivo conectado altera sua célula de fixação (de Célula A para Célula B). Atualiza registro de localização e timestamp de último evento.
  - `DETACH`: Encerra a sessão (`DISCONNECTED`), registra timestamp de encerramento e libera o IP de volta ao pool.

#### Arquivos e Pacotes
```text
internal/network/
├── cell.go                # Topologia e catálogo estático de células simuladas
├── ippool.go              # Gerenciador concorrente de alocação de IPs CIDR
└── ippool_test.go         # Testes de exaustão e reciclagem de IPs
internal/session/
├── session.go             # Entidade Session, status (CONNECTED, DISCONNECTED), validações
├── repository.go          # Interface e repositório in-memory indexado por SessionID e DeviceID
├── service.go             # Casos de uso: Attach, Handover, Detach, GetActiveSession
├── handler.go             # Endpoints HTTP: POST /sessions/attach, POST /sessions/{id}/handover, POST /sessions/{id}/detach
├── session_test.go        # Testes de ciclo de vida e concorrência de sessão
└── handler_test.go        # Testes de integração HTTP
```

#### Dependências e Tecnologias
- Biblioteca padrão (`net`, `sync`, `net/http`).

#### Testes Necessários
- Fluxo completo de Attach -> Handover -> Detach validando a devolução do IP.
- Teste de concorrência com 50 goroutines executando Attach/Detach simultâneos para validar integridade do `IPPool` e ausência de deadlocks.
- Re-attach idempotente fechando sessão obsoleta.

#### Critérios de Aceitação & Não-Objetivos
- [ ] Endpoints de Attach, Handover e Detach operando com respostas semânticas estruturadas.
- [ ] Pool de IP alocando e desalocando sem vazamento de endereços.
- [ ] **Não-Objetivos:** Sem protocolos binários de telecomunicações (GTP-U/NAS); sem roteamento de pacotes físicos no sistema operacional.

---

### MILESTONE 4 — Telemetry & Asynchronous Events (Observabilidade e Concorrência)

#### Objetivo
Construir a camada de observabilidade operacional leve e desacoplada, utilizando canais nativos do Go para tráfego assíncrono de eventos de rede e contadores atômicos em memória.

#### Comportamento Esperado do Domínio
- **Pipeline de Eventos em Canais Go:** O serviço de sessões publica eventos (`session.attached`, `session.handover`, `session.detached`, `subscriber.created`) em um canal interno bufferizado (`chan Event`).
- **Worker Concorrente de Telemetria:** Uma goroutine em background consome eventos do canal sem bloquear os handlers HTTP de escrita, atualizando contadores com segurança thread-safe (`sync/atomic`).
- **Endpoint Operacional `GET /telemetry`:** Retorna JSON com métricas consolidadas em tempo real:
  - `active_subscribers`
  - `registered_devices`
  - `active_sessions`
  - `total_handowers`
  - `total_attaches`
  - `total_detaches`
  - `events_processed_total`
- **Middleware de Request ID & Structured Logging:** Injeção de `X-Request-ID` em todas as requisições HTTP e uso do `log/slog` padronizado.

#### Arquivos e Pacotes
```text
internal/telemetry/
├── event.go               # Tipos de eventos de domínio (Payload, Tipo, Timestamp)
├── collector.go           # Coletor de métricas em memória com sync/atomic e worker de consumo
├── handler.go             # Handler HTTP para GET /telemetry
└── collector_test.go      # Testes de concorrência e processamento de eventos
internal/platform/httpserver/
├── middleware.go          # Middleware de Request ID, logging estruturado e contagem de requisições
└── server.go              # Registro do endpoint /telemetry e injeção do collector
```

#### Dependências e Tecnologias
- Biblioteca padrão do Go (`log/slog`, `sync/atomic`, `context`).

#### Testes Necessários
- Teste do worker de canal: envio de rajada de eventos em paralelo garantindo que nenhum evento seja perdido e os contadores reflitam os números exatos.
- Encerramento gracioso (*graceful shutdown*): ao fechar a API, o canal de telemetria drena os eventos pendentes com timeout.

#### Critérios de Aceitação & Não-Objetivos
- [ ] `GET /telemetry` retornando contadores precisos refletindo operações reais.
- [ ] Logs estruturados exibindo `request_id`, método, rota e latência.
- [ ] **Não-Objetivos:** Sem Prometheus, Grafana ou OpenTelemetry neste estágio; sem Kafka/RabbitMQ externo.

---

### MILESTONE 5 — Simulation Runner CLI (Motor de Concorrência do Laboratório)

#### Objetivo
Criar o executável autônomo `cmd/simulator` para demonstrar concorrência real em Go e fornecer a experiência de demonstração definitiva (*Hero Flow*) para avaliadores técnicos.

#### Comportamento Esperado do Domínio
- Executável autônomo acionado via `go run ./cmd/simulator`.
- Configuração via flags e variáveis de ambiente (número de dispositivos, duração, intervalo).
- Funcionamento:
  1. Provisiona $N$ assinantes e ativa-os na API;
  2. Registra $N$ dispositivos vinculados;
  3. Lança $N$ goroutines autônomas concorrentes (cada uma simulando um equipamento móvel independente com seu próprio ciclo de vida);
  4. Cada goroutine executa: Attach $\rightarrow$ permanência $\rightarrow$ 1 a 3 Handovers aleatórios entre as células $\rightarrow$ Detach;
  5. Tratamento de cancelamento gracioso via `Ctrl+C` (`context.WithCancel`).
- Exibição de resumo no terminal ao finalizar (taxa de sucesso, erros encontrados, duração).

#### Arquivos e Pacotes
```text
cmd/simulator/
├── main.go                # Ponto de entrada do simulador CLI
├── client.go              # Cliente HTTP leve para interagir com a API do monólito
├── scenario.go            # Lógica de orquestração de cenários concorrentes com WaitGroup
└── scenario_test.go       # Teste de execução de cenário em ambiente mock/httptest
```

#### Dependências e Tecnologias
- Biblioteca padrão do Go (`sync.WaitGroup`, `time`, `net/http`, `os/signal`).

#### Testes Necessários
- Execução de teste do simulador contra um `httptest.Server`, validando que todas as goroutines encerram de forma limpa e sem vazamento de recursos.

#### Critérios de Aceitação & Não-Objetivos
- [ ] Avaliador executa `go run ./cmd/api` em um terminal e `go run ./cmd/simulator` em outro, observando a simulação fluir e inspecionando `GET /telemetry`.
- [ ] **Não-Objetivos:** Sem interface gráfica; sem necessidade de instalar Node/React/Electron.

---

### MILESTONE 6 — PostgreSQL & Migrations (Persistência Conquistada por Requisitos)

#### Objetivo
Introduzir persistência em banco relacional PostgreSQL com migrações de esquema versionadas, implementando as interfaces de repositório existentes sem alterar uma única linha da lógica de domínio ou dos handlers.

#### Comportamento Esperado do Domínio
- Manter o comportamento das entidades e casos de uso idêntico ao dos marcos anteriores.
- Persistência permanente para assinantes, dispositivos e sessões.
- Configuração via variáveis de ambiente (`DATABASE_URL`, `DB_MAX_OPEN_CONNS`, etc.).
- Migrações SQL executadas automaticamente no startup ou via comando explícito.

#### Arquivos e Pacotes
```text
deployments/
└── docker-compose.yml     # Definição do container PostgreSQL 16
migrations/
├── 000001_create_subscribers.up.sql
├── 000001_create_subscribers.down.sql
├── 000002_create_devices.up.sql
└── 000003_create_sessions.up.sql
internal/platform/postgres/
├── connection.go          # Gerenciamento de pool de conexões com pgxpool / database/sql
└── migrator.go            # Execução de migrações nativas
internal/subscriber/
└── postgres_repository.go # Implementação da interface SubscriberRepository via SQL
internal/device/
└── postgres_repository.go # Implementação da interface DeviceRepository via SQL
internal/session/
└── postgres_repository.go # Implementação da interface SessionRepository via SQL
```

#### Dependências e Tecnologias
- Driver PostgreSQL moderno para Go (`github.com/jackc/pgx/v5` ou `database/sql` com `lib/pq`).
- Docker Compose.

#### Testes Necessários
- Testes de integração de repositório contra PostgreSQL real (via container de teste ou banco local).
- Validação de idempotência e rollback das migrações SQL.

#### Critérios de Aceitação & Não-Objetivos
- [ ] A aplicação sobe conectando ao Postgres quando configurado, ou mantém fallback In-Memory se explicitado.
- [ ] Migrações aplicam índices únicos em `imsi`, `msisdn` e `imei`.
- [ ] **Não-Objetivos:** Sem ORMs pesados (GORM); escrita de queries SQL limpas, diretas e indexadas.

---

### MILESTONE 7 — Documentação Técnica & Transparência de Engenharia

#### Objetivo
Finalizar a apresentação do laboratório, consolidando a documentação para que qualquer engenheiro ou recrutador compreenda a arquitetura, execute a demonstração sem atritos e veja a governança técnica adotada.

#### Entregáveis
- **`README.md` Principal:**
  - Descrição clara do laboratório e diagramas de arquitetura.
  - Guia de execução rápida em 3 comandos (*Zero-Setup Friction*).
  - Tabela de endpoints e comandos do simulador.
  - Seção formal **Engineering Process & AI Collaboration** (documentando a metodologia de trabalho humano + IA com base no `HUMAN_ENGINEERING_REVIEW.md`).
- **ADRs Complementares:**
  - `docs/adr/ADR-002-progressive-persistence-inmemory-to-postgres.md`
  - `docs/adr/ADR-003-inprocess-concurrency-over-message-brokers.md`

---

## 4. Confronto com as Diretrizes de Engenharia Humana

Antes de iniciar a implementação, cada etapa foi confrontada com os princípios definidos no [HUMAN_ENGINEERING_REVIEW.md](HUMAN_ENGINEERING_REVIEW.md):

| Regra Anti-Padrão de IA | Como este Master Plan assegura conformidade |
| :--- | :--- |
| **Proibição de Interfaces Espelho 1:1** | Interfaces só existem onde há motivo real: nos repositórios (porque existem implementações In-Memory e Postgres) e onde o consumidor precisa de desacoplamento (`SubscriberChecker`). |
| **Vetado o uso de Managers/Helpers/Utils** | Nenhum pacote ou struct recebeu sufixos genéricos. Todos os tipos são substantivos do domínio (`Subscriber`, `Device`, `Session`, `IPPool`). |
| **Proibição de Subpastas Prematuras (Clean Arch Excessiva)** | Cada módulo vive em seu pacote único coeso (`internal/subscriber`, `internal/device`, `internal/session`), mantendo arquivos proporcionais e testes lado a lado. |
| **Concorrência Justificada** | Concorrência usada com propósito estrito: simulação de múltiplos celulares (`cmd/simulator`), worker de canal de telemetria e encerramento gracioso HTTP. |
| **Tecnologias Posteragadas com Rigor** | Redis, Kafka/RabbitMQ, gRPC, Prometheus/Grafana e C permanecem estritamente fora de escopo. |

---

## 5. Protocolo de Execução por Milestone

A implementação será executada **milestone por milestone**, seguindo este ciclo inviolável:

1. **Apresentação:** O agente detalha os arquivos que serão criados/modificados no milestone.
2. **Implementação:** Construção exclusiva do escopo do marco vigente.
3. **Formatação & Qualidade:** Execução de `go fmt ./...`, `go vet ./...` e compilação `go build ./...`.
4. **Verificação de Testes:** Execução de testes de unidade e concorrência (`go test -race ./...`).
5. **Inspeção de Diff:** Apresentação exata das alterações realizadas.
6. **Parada Obrigatória de Revisão:** Nenhuma etapa seguinte será iniciada sem a revisão e aprovação explícita do usuário.
