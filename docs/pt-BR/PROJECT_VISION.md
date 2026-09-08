# NEXUS Core Lab — Project Vision

## 1. Visão

O NEXUS Core Lab é uma plataforma educacional de backend para simulação de conceitos presentes em redes móveis 4G/5G.

O projeto tem como objetivo explorar engenharia de software aplicada a telecomunicações, sistemas distribuídos, processamento concorrente, comunicação entre serviços, observabilidade e alta performance.

O sistema NÃO pretende implementar um Core 4G/5G comercial nem substituir componentes reais de uma operadora.

Ele funciona como um laboratório de engenharia no qual dispositivos, assinantes, sessões, células, eventos e serviços de rede são representados por entidades simuladas.

---

## 2. Objetivos técnicos

O projeto deverá demonstrar conhecimento prático em:

- Go
- desenvolvimento backend
- APIs REST
- gRPC
- PostgreSQL
- Redis
- mensageria assíncrona
- Docker
- arquitetura modular
- concorrência em Go
- sistemas distribuídos
- observabilidade
- métricas
- logging estruturado
- testes automatizados
- testes de integração
- CI/CD
- simulação de carga
- conceitos de telecomunicações 4G/5G

Em uma etapa posterior, será desenvolvido um pequeno módulo em C para experimentação com processamento de mensagens/protocolos.

---

# 3. Domínio

O domínio inicial será dividido em cinco áreas principais.

## Subscriber

Representa um assinante fictício da rede.

Exemplos de informações:

- ID interno
- IMSI simulado
- MSISDN simulado
- status
- data de criação

Nenhum identificador real de telecomunicações deverá ser utilizado.

---

## Device

Representa um equipamento virtual associado a um assinante.

Exemplos:

- Device ID
- Subscriber ID
- tipo do dispositivo
- tecnologia suportada
- status

Tecnologias simuladas inicialmente:

- LTE
- 5G

---

## Session

Representa a conexão lógica de um dispositivo à rede simulada.

Estados iniciais:

- DISCONNECTED
- ATTACHING
- CONNECTED
- DETACHING

Uma sessão poderá registrar:

- dispositivo
- subscriber
- tecnologia
- célula
- início da sessão
- último evento
- encerramento

---

## Location

Representa a localização lógica de um dispositivo dentro da rede.

A localização será baseada inicialmente em células simuladas.

Exemplo:

CELL-CPS-001

Campinas / SP

Cada movimentação poderá gerar um evento de localização.

---

## Network

Representa os elementos simulados da infraestrutura.

Inicialmente:

- Cells
- Network Nodes
- Network Status

Posteriormente poderão ser introduzidos conceitos inspirados em componentes de redes 4G/5G.

---

# 4. Fluxo básico

Um fluxo inicial poderá ocorrer da seguinte maneira:

Device Simulator
|
v
Subscriber Validation
|
v
Session Creation
|
v
Network Assignment
|
v
Location Registration
|
v
Event Publication
|
v
Telemetry / Persistence

Exemplo:

DEVICE-00042 solicita conexão.

O sistema:

1. identifica o dispositivo;
2. valida o subscriber;
3. cria uma sessão;
4. associa uma célula;
5. registra a localização;
6. publica eventos;
7. atualiza métricas.

---

# 5. Eventos

O projeto deverá evoluir para uma arquitetura orientada a eventos.

Exemplos:

subscriber.created

device.registered

device.attached

device.detached

session.created

session.closed

location.updated

network.cell.changed

network.error

Os eventos poderão ser processados assincronamente.

---

# 6. Arquitetura inicial

A primeira versão será construída como um modular monolith.

Motivo:

Evitar complexidade prematura de microservices enquanto os limites do domínio ainda estão sendo definidos.

Estrutura prevista:

cmd/
api/

internal/
subscriber/
device/
session/
location/
network/
telemetry/

pkg/

configs/

migrations/

docs/

tests/

deployments/

Posteriormente alguns módulos poderão ser extraídos para serviços independentes.

---

# 7. Persistência

Banco principal:

PostgreSQL

Responsável inicialmente por:

- subscribers
- devices
- sessions
- cells
- location history
- event metadata

Redis poderá ser introduzido posteriormente para:

- cache
- estado temporário
- sessões de alta frequência
- rate limiting

---

# 8. Comunicação

Primeira fase:

REST

Segunda fase:

gRPC

Terceira fase:

mensageria assíncrona.

A introdução dessas tecnologias deverá acontecer somente quando houver uma necessidade arquitetural clara.

---

# 9. Observabilidade

A aplicação deverá possuir observabilidade desde as primeiras versões.

Inicialmente:

- structured logging
- request ID
- health check

Posteriormente:

- Prometheus
- Grafana
- distributed tracing

Métricas previstas:

- active_sessions
- connected_devices
- disconnected_devices
- requests_total
- request_duration
- errors_total
- events_processed
- events_failed

---

# 10. Simulador

Será desenvolvido um Device Simulator separado.

Ele deverá ser capaz de gerar dispositivos virtuais e simular:

- attach
- detach
- heartbeat
- mudança de célula
- atualização de localização
- perda de conexão
- reconexão

Posteriormente serão realizados testes com diferentes volumes de dispositivos.

Exemplos:

10

100

1.000

10.000+

O objetivo será estudar comportamento, concorrência, throughput e latência.

---

# 11. Protocol Lab

Uma área experimental será adicionada posteriormente.

Objetivo:

Estudar conceitos relacionados a protocolos utilizados em telecomunicações.

Poderão ser estudados:

- SIP
- Diameter
- SCTP

Não será implementada uma infraestrutura comercial completa desses protocolos.

O laboratório deverá utilizar mensagens fictícias e ambientes controlados.

---

# 12. Módulo C

Uma etapa futura deverá introduzir C.

Possíveis responsabilidades:

- parsing de mensagens
- serialização binária
- processamento de buffers
- benchmark de processamento

A integração com Go deverá possuir justificativa técnica e documentação.

---

# 13. Segurança

Mesmo sendo um laboratório, boas práticas serão aplicadas.

Incluindo:

- validação de entrada
- configuração via environment variables
- secrets fora do repositório
- tratamento de erros
- limites de requisição
- logs sem informações sensíveis

Todos os dados utilizados serão fictícios.

---

# 14. Testes

O projeto deverá possuir:

Unit Tests

Integration Tests

API Tests

Load Tests

Cada módulo importante deverá possuir cobertura de testes adequada.

O objetivo não será perseguir artificialmente 100% de cobertura, mas testar comportamentos relevantes.

---

# 15. CI/CD

GitHub Actions será utilizado para:

- build
- tests
- lint
- static analysis

Nenhum commit deverá depender apenas de execução manual para verificar a integridade básica do projeto.

---

# 16. Documentação arquitetural

Decisões importantes serão registradas utilizando ADRs.

Exemplos:

ADR-001 — Modular Monolith

ADR-002 — PostgreSQL

ADR-003 — REST API

ADR-004 — Event Broker

ADR-005 — Redis

ADR-006 — gRPC

Isso permitirá acompanhar a evolução das decisões arquiteturais do projeto.

---

# 17. Fora do escopo

A primeira versão NÃO deverá:

- implementar um Core 5G real;
- conectar-se a redes de operadoras;
- utilizar IMSIs reais;
- utilizar números telefônicos reais;
- implementar billing real;
- processar tráfego real;
- reproduzir infraestrutura proprietária;
- implementar protocolos telecom completos.

O projeto é exclusivamente educacional e experimental.

---

# 18. Roadmap

## Phase 0 — Foundation

- repository
- Go module
- project structure
- configuration
- logging
- health endpoint
- Docker
- PostgreSQL

## Phase 1 — Subscriber Registry

- subscriber entity
- repository
- service
- API
- validation
- tests

## Phase 2 — Device Registry

- devices
- subscriber association
- network capability
- status

## Phase 3 — Session Manager

- attach
- detach
- session lifecycle
- state machine

## Phase 4 — Location Service

- cells
- device location
- movement
- location history

## Phase 5 — Event Architecture

- event model
- broker
- producers
- consumers
- retries
- idempotency

## Phase 6 — Observability

- Prometheus
- Grafana
- metrics
- tracing

## Phase 7 — Device Simulator

- virtual devices
- concurrent operations
- load generation

## Phase 8 — Protocol Lab

- SIP concepts
- Diameter concepts
- SCTP concepts

## Phase 9 — C Integration

- protocol processing experiment
- Go/C integration
- benchmarks

## Phase 10 — NOC Dashboard

- network overview
- active devices
- active sessions
- latency
- cells
- event stream
- system health

---

# 19. Resultado esperado

Ao final, o NEXUS Core Lab deverá funcionar simultaneamente como:

1. laboratório de aprendizado;
2. projeto de portfólio;
3. demonstração de engenharia backend;
4. estudo de sistemas distribuídos;
5. introdução prática ao domínio de telecomunicações;
6. ambiente para experimentos de performance;
7. demonstração pública de evolução técnica.

O valor do projeto não será medido apenas pela quantidade de funcionalidades.

As principais características deverão ser:

clareza arquitetural;

qualidade do código;

testabilidade;

observabilidade;

documentação;

decisões técnicas justificadas;

evolução incremental;

performance mensurável.
