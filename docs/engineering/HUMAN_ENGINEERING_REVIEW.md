# NEXUS CORE LAB — HUMAN ENGINEERING REVIEW

Esta síntese consolida as conclusões das três rodadas de _Grill Me_, estabelecendo as diretrizes que garantem que o NEXUS Core Lab seja percebido como o trabalho técnico de uma equipe de engenharia de software sênior — autêntico, pragmático e tecnicamente convincente —, e não uma aplicação massivamente gerada por IA.

---

## 1. Decisões Confirmadas

### Arquitetura & Domínio

- **Monólito Modular Estrito (ADR-001):** Um único executável backend que abriga domínios bem delimitados (`subscriber`, `device`, `session`, `network`, `telemetry`).
- **Comunicação entre Módulos:** Desacoplada via _consumer-defined interfaces_ (interfaces pequenas declaradas no pacote que consome a dependência) e composição no `cmd/api/main.go`. Não há structs internas vazando entre domínios nem importações circulares.
- **Topologia de Rede Semântica:** Células de rede operam com uma topologia estática em memória (ex: 3 células simulando regiões de Campinas/SP). Elimina-se o CRUD dinâmico artificial de infraestrutura de antenas.
- **Hero Flow Principal:**
  1. Provisionar Assinante (`POST /api/v1/subscribers`);
  2. Ativar Assinante (`POST /api/v1/subscribers/{id}/activate`);
  3. Registrar Dispositivo vinculado (`POST /api/v1/devices`);
  4. Dispositivo executa Attach de Rede (`POST /api/v1/sessions/attach` -> aloca IP virtual privado);
  5. Dispositivo realiza Handover entre células (`POST /api/v1/sessions/{id}/handover`);
  6. Dispositivo executa Detach (`POST /api/v1/sessions/{id}/detach` -> libera IP);
  7. Inspecionar telemetria em tempo real (`GET /telemetry`).

### Ciclo de Vida e Regras de Negócio

- **Subscriber:** Máquina de estados explícita: `PENDING_ACTIVATION` -> `ACTIVE` <-> `SUSPENDED` -> `DEACTIVATED` (estado terminal irreversível). Sem endpoint `DELETE` físico.
- **Unicidade Estrita:** IMSI (15 dígitos) e MSISDN (E.164 simulado) únicos globalmente. Tentativas de duplicidade retornam HTTP `409 Conflict`.
- **Sessões & Concorrência:** Cada equipamento possui no máximo 1 sessão ativa. Re-attach substitui a sessão anterior de forma idempotente (`STALE_DISCONNECT`).

### Persistência & Observabilidade

- **Persistência Progressiva em 2 Etapas:**
  - _Milestone 1:_ Repositório in-memory thread-safe (`sync.RWMutex`), garantindo foco absoluto nas regras de domínio e testes rápidos.
  - _Milestone 2:_ Introdução justificada do PostgreSQL via Docker Compose com migrações SQL nativas, sem alterar o contrato do domínio.
- **Observabilidade Progressiva Leve:** `log/slog` da biblioteca padrão, middleware de `Request ID` em todas as requisições HTTP e contadores operacionais atômicos (`sync/atomic`) expostos em `GET /telemetry`.

---

## 2. Riscos de Overengineering Mapeados e Mitigados

| Risco de Overengineering                                 | Como foi mitigado                                                                                                                                      |
| :------------------------------------------------------- | :----------------------------------------------------------------------------------------------------------------------------------------------------- |
| **Microserviços prematuros**                             | Mantido o Monólito Modular conforme ADR-001 até que haja necessidade operacional mensurável.                                                           |
| **Message Broker externo (Kafka / RabbitMQ / NATS)**     | Uso de **Go channels nativos** no processo para eventos assíncronos. Brokers externos descartados enquanto não houver múltiplos serviços distribuídos. |
| **Armazenamento em Cache (Redis)**                       | Descartado. O estado efêmero de sessões é resolvido com concorrência segura em memória e persistência relacional.                                      |
| **Comunicação gRPC prematura**                           | Descartada. A API REST padrão atende perfeitamente ao monólito e ao Hero Flow.                                                                         |
| **Stack pesada de monitoramento (Prometheus + Grafana)** | Adiados. O endpoint nativo `GET /telemetry` expõe os dados sem obrigar o avaliador a levantar múltiplos containers auxiliares.                         |
| **CRUDs anêmicos para infraestrutura física**            | Células de rede nascem com dados fixos bem configurados no código/memória, sem rotas HTTP desnecessárias para gerenciar antenas ou frequências.        |

---

## 3. Sinais de Artificialidade ("AI-Generated Code") a Evitar Rigidamente

Para garantir que o código pareça escrito por engenheiros experientes, as seguintes regras foram estabelecidas:

1. **Proibição de Interfaces Espelho 1:1:** Não criar interfaces cujo único propósito seja espelhar uma única struct de implementação (ex: banido criar `type SubscriberService interface` se existe apenas `type subscriberService struct`). Em Go, interfaces pertencem a quem consome e só existem quando há múltiplas implementações reais (ex: repositório in-memory vs repositório postgres).
2. **Proibição de Sufixos Genéricos:** Nomes como `Manager`, `Helper`, `Utils`, `Common`, `HandlerImpl` ou `BaseRepository` são terminantemente vetados. Tipos devem ser substantivos do domínio (`Subscriber`, `Device`, `Session`, `IPPool`).
3. **Proibição de Camadas Falsas (Clean Architecture Dogmática):** Não criar 5 subpastas para 30 linhas de código (`domain/ports`, `domain/entities`, `infrastructure/adapters/in/http`). O pacote coeso em `internal/subscriber` manterá arquivos proporcionais e coesos.
4. **Sem Proliferação de DTOs Idênticos:** Não duplicar structs em 4 níveis (`SubscriberEntity`, `SubscriberModel`, `SubscriberDTO`, `SubscriberInput`) a menos que haja diferença substancial de dados e validação.
5. **Comentários de Código Qualificados:** Banidos comentários redundantes como `// Activate activates the subscriber`. Comentários devem explicar o **porquê** de regras de negócio telecom não óbvias.
6. **Testes Baseados em Comportamento:** Evitar testes superficiais cujo objetivo seja apenas bater percentual de cobertura. Os testes devem cobrir invariantes de estado, concorrência (com detecção de _data races_ via `go test -race`) e casos de borda reais.

---

## 4. Pontos que Dão Identidade Própria ao Projeto

- **Foco Educacional em Telecomunicações Reais:** Uso correto da semântica 3GPP (IMSI, MSISDN, IMEI, Attach, Detach, Handover, IP Pool CIDR) sem pretender ser uma pilha binária comercial.
- **Simulador Concorrente Nativo (`cmd/simulator`):** O projeto não é apenas uma API passiva; ele possui um executável que atua como dispositivos virtuais com goroutines autônomas gerando carga e eventos reais de rede.
- **Concorrência com Propósito:** Goroutines usadas onde realmente modelam concorrência física (dispositivos móveis independentes e workers de telemetria), e não para operações triviais síncronas de CRUD.

---

## 5. Elementos que Tornam o Projeto Tecnicamente Convincente

1. **Experiência do Avaliador sem Fricção (_Zero-Setup Friction_):**

   ```bash
   # Terminal 1: API REST
   go run ./cmd/api

   # Terminal 2: Simulação de Tráfego Concorrente
   go run ./cmd/simulator

   # Terminal 3: Inspeção de Telemetria ao Vivo
   curl http://localhost:8080/telemetry

   # Execução de Testes com Race Detector
   go test -race ./...
   ```

2. **Robustez de Domínio em Go:** Validação estrita de tipos, mapeamento de erros HTTP semânticos (`409 Conflict`, `422 Unprocessable Entity`), e encerramento gracioso via contexto.
3. **Evolução Arquitetural Transparente:** O histórico de commits e os ADRs documentarão o crescimento progressivo (In-Memory -> PostgreSQL).

---

## 6. Elementos que Devem Permanecer Fora de Escopo (`OUT OF SCOPE`)

- Criptografia SIM / Algoritmos Milenage / Chaves Ki e OPc.
- Protocolos binários de baixo nível (ASN.1, GTP-U, NAS, SCTP).
- Interface web/frontend (o foco do laboratório é 100% backend/sistemas).
- Kubernetes / Helm Charts / Service Mesh.
- Módulos experimentais em C (postergados até que o núcleo em Go esteja 100% consolidado e haja necessidade mensurável de benchmark).

---

## 7. Como a IA é Utilizada como Parte da Equipe de Engenharia

O projeto documentará no `README.md` (sob a seção **Engineering Process & AI Collaboration**) o modelo de trabalho adotado:

> **Transparência Técnica:**
> O NEXUS Core Lab foi desenvolvido utilizando Inteligência Artificial em modelo de _Pair Programming_ e assistência de implementação, sob rigorosa governança de engenharia humana.
>
> - **Papel do Engenheiro Humano:** Direção de produto, definição de escopo, especificação dos modelos de telecomunicações, desafio de arquitetura (/grill-me), revisão técnica de código e critérios de aceitação.
> - **Papel do Agente de IA:** Assistente de implementação, validação estática de requisitos, geração de testes unitários orientados a casos de borda e documentação de decisões.
> - **Governança:** Todas as decisões respeitam o [AI_ENGINEERING_GUIDELINES.md](docs/engineering/AI_ENGINEERING_GUIDELINES.md) e os ADRs do repositório, garantindo código idiomático em Go, livre de complexidade acidental ou abstrações artificiais.
