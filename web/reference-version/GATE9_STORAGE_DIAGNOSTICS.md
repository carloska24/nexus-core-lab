# Gate 9 — Storage & Database Diagnostics

15/09/2026. Gate 9 aprovado pelo Human Review, com checkpoint autorizado. Este relatório preserva o estado da entrega anterior ao checkpoint. Validações Go, race detector e build frontend são repetidas antes do commit.

## 1. Baseline Git

Branch `feat/demo-ui`, HEAD `d16b1ecfb79d3263baa9be718967e8382d2915c6`. Working tree estava CLEAN antes das alterações. Nenhum checkpoint novo criado.

## 2–3. Seleção e startup originais

Inspecionados cmd/api/main.go, cmd/migrate/main.go, inicialização PostgreSQL, repositories, shutdown, /health, /telemetry e provider/indicadores frontend.

`os.Getenv("DATABASE_URL") == ""` seleciona repositories MEMORY. Valor não vazio seleciona POSTGRESQL. Essa regra foi preservada, inclusive sem tratar uma string de espaços como configuração ausente.

`postgres.Open` usa sql.Open com driver pgx, configura o pool existente (máximo 25 conexões abertas e ociosas, lifetime cinco minutos) e executa PingContext com cinco segundos. Falha fecha o handle e impede startup. API também exige schema versão 3 e warm-up do IPPool antes de aceitar HTTP. Não houve fallback novo para memória nem relaxamento dessas condições.

Main possui o sql.DB, injeta o mesmo handle nos repositories e executa defer Close após shutdown normal. Repositories não possuem o lifecycle do pool. O processo migrate usa seu próprio handle e timeout de 30 segundos; não foi alterado nem executado neste Gate.

## 4. Arquivos

- `cmd/api/main.go`: import e registro do handler com o sqlDB existente.
- Novos `internal/platform/storage/handler.go` e `handler_test.go`.
- Novo `web/reference-version/src/storage-api.ts`.
- `src/monitoring.tsx`: uma leitura compartilhada de storage, sem alterar o hook ou polling das outras fontes.
- `src/App.tsx`: Topbar, indicador sidebar e descrição de diagnóstico.
- `src/LivePanels.tsx`: conteúdo mínimo da tela Database.
- `vite.config.ts`: proxy para a nova rota.
- `tests/gate9.cjs`, `evidence/gate9/results.json`, screenshots locais, README e este relatório.

## 5–8. Endpoint e JSON

`GET /api/v1/system/storage`, read-only, Content-Type application/json e Cache-Control no-store.

```json
{"mode":"MEMORY","database_configured":false,"database_status":"NOT_APPLICABLE"}
```

```json
{"mode":"POSTGRESQL","database_configured":true,"database_status":"AVAILABLE"}
```

```json
{"mode":"POSTGRESQL","database_configured":true,"database_status":"UNAVAILABLE"}
```

MEMORY é modo operacional válido, com dados temporários, não falha de banco. POSTGRESQL representa o modo selecionado no startup e a alcançabilidade do handle naquele instante. AVAILABLE não afirma schema, integridade de dados ou sucesso de todas as queries futuras.

## 9–12. Ping, timeout, HTTP e segurança

Construtor recebe *sql.DB. Quando não nil, armazena somente o method value db.PingContext; com nil, nenhum Ping é feito. Pequena função privada permite testes controlados sem introduzir manager/framework ou interface genérica. O diagnóstico nunca abre nem fecha um pool, não executa SQL de aplicação e não modifica repositories.

Cada request PostgreSQL utiliza context.WithTimeout(r.Context(), 2*time.Second). Timeout e cancelamento resultam em UNAVAILABLE. HTTP permanece 200 porque o diagnóstico foi realizado e descreve a indisponibilidade. POST/PUT/DELETE retornam 405. GET/HEAD seguem a semântica padrão do ServeMux Go.

Erro do driver nunca é serializado nem logado por esse handler. JSON contém exclusivamente os três campos documentados: sem URL, host, porta, usuário, senha, DSN ou token. Teste injeta mensagem de erro contendo credenciais fictícias e verifica ausência desses dados na resposta.

Não foram alterados logs ou política de falha já existentes no startup. A garantia de sanitização aqui diz respeito ao novo endpoint, não constitui uma auditoria completa de logs antigos.

## 13–16. Testes backend

- MEMORY: HTTP 200, configured false, NOT_APPLICABLE; construtor com DB nil não oferece função de Ping e não abre conexão.
- AVAILABLE: função de Ping retorna sucesso.
- UNAVAILABLE: função retorna erro contendo detalhes fictícios sensíveis; resposta sanitizada com exatamente três campos.
- Timeout: função aguarda ctx.Done, deadline de dois segundos confirmado, retorno bounded.
- Cancelamento: request previamente cancelada termina imediatamente em UNAVAILABLE.
- Liveness: /health continua HTTP 200 mesmo quando storage PostgreSQL falha; teste proíbe que /health invoque Ping.
- Métodos de mutação rejeitados.

A primeira compilação apontou import não utilizado no teste; removido antes das suítes completas. Nenhuma correção funcional decorrente disso.

## 17–20. Frontend e separação das fontes

Topbar preserva a composição aprovada: API continua baseada exclusivamente em /health; Storage apresenta MEMORY ou POSTGRESQL · ONLINE/OFFLINE. Sidebar usa o mesmo snapshot; a tela Database apresenta modo, configuração, último status reportado e instante de observação.

Uma assinatura no MonitoringProvider consulta storage dez segundos após a conclusão da leitura anterior. Infraestrutura existente mantém deadline de oito segundos, AbortController, cancelamento no unmount e ausência de sobreposição. Não foram alterados os intervalos de health/Telemetry/IPPool.

Validador aceita somente combinações coerentes de modo/configuração/status. Loading mostra Checking; success mostra o diagnóstico; erro inicial mostra Unknown; stale preserva o último estado explicitamente rotulado e usa indicador neutro. Falha de transporte nunca é convertida em novo PostgreSQL OFFLINE. O último status conhecido pode ser OFFLINE, mas fica acompanhado de Stale.

Decisão: usar o provider existente evita três consultas separadas para Topbar, sidebar e tela Database. Não houve cache, worker, channel ou dependência nova. Nenhum cálculo de readiness global foi introduzido.

## 21–23. Execução e evidência visual

API isolada em 127.0.0.1:18109, frontend localhost:5209, DATABASE_URL vazio. /health retornou ok e o storage retornou MEMORY/false/NOT_APPLICABLE. Nenhum banco do usuário foi manipulado.

**PostgreSQL integration tests not executed.** TEST_DATABASE_URL indisponível; nenhum banco criado. AVAILABLE/UNAVAILABLE são testes controlados, não evidência de PostgreSQL real. Não foi possível demonstrar startup real com PostgreSQL indisponível sem contrariar a política existente; ela permanece intacta. Runtime UNAVAILABLE é coberto por teste de Ping controlado.

Screenshots reais MEMORY:

- [1920x1080](evidence/gate9/memory-1920x1080.png)
- [1440x900](evidence/gate9/memory-1440x900.png)
- [1366x768](evidence/gate9/memory-1366x768.png)

Limites do indicador verificados nas três resoluções; composição inspecionada visualmente. Nenhuma screenshot simulada foi apresentada como PostgreSQL real. PNGs locais são ignorados pelo Git conforme regra existente. [Resultados de navegador](evidence/gate9/results.json).

Testes de navegador: **11 verificações PASS, zero erros JavaScript não tratados**. Incluem loading, MEMORY real, PostgreSQL AVAILABLE/UNAVAILABLE controlados, stale, erro inicial, recovery, JSON inconsistente, intervalo sem sobreposição, timeout de transporte e unmount.

O cenário de timeout inicialmente expirou na sequência de interceptações do Playwright. Uma reprodução isolada confirmou AbortController e estado stale. O teste final inicia esse cenário após recarga limpa e usa um servidor HTTP local com resposta atrasada; passou sem alterar o hook de produção. O servidor auxiliar é encerrado pelo teste.

## 24–30. Regressão

| Validação | Resultado |
|---|---|
| npm run build | PASS |
| go fmt ./... | PASS; finais de linha incidentais fora do escopo preservados |
| go vet ./... | PASS |
| go build ./... | PASS |
| go test -count=1 ./... | PASS |
| go test -race -count=1 ./... | PASS via Docker |
| Quantidade de data races | ZERO |

Docker golang:latest, CGO_ENABLED=1, GOTOOLCHAIN=local, GOPROXY=off, rede desabilitada e mounts de código/módulos somente para leitura.

## 31. git diff --stat

Arquivos novos não aparecem nesta estatística dos rastreados:

```text
 cmd/api/main.go                          |  2 ++
 web/reference-version/README.md          |  7 +++++--
 web/reference-version/src/App.tsx        | 13 ++++++++-----
 web/reference-version/src/LivePanels.tsx |  8 +++++---
 web/reference-version/src/monitoring.tsx |  7 +++++--
 web/reference-version/vite.config.ts     |  1 +
 6 files changed, 26 insertions(+), 12 deletions(-)
```

## 32. git status --short

```text
 M cmd/api/main.go
 M web/reference-version/README.md
 M web/reference-version/src/App.tsx
 M web/reference-version/src/LivePanels.tsx
 M web/reference-version/src/monitoring.tsx
 M web/reference-version/vite.config.ts
?? internal/platform/storage/
?? web/reference-version/GATE9_STORAGE_DIAGNOSTICS.md
?? web/reference-version/evidence/gate9/
?? web/reference-version/src/storage-api.ts
?? web/reference-version/tests/gate9.cjs
```

## 33–35. Escopo e Human Review

Nenhuma migration, dependência nova, pool adicional ou persistência nova. IPPool/Gate 8 intacto; Subscriber, Device, Session, transações, topologia, eventos, counters e Simulator preservados. /health continua liveness simples. Nenhum serviço PostgreSQL do usuário foi derrubado ou reconfigurado.

Sem commit, sem push, sem Gate 10. Validação final concluída. Parado para Human Review.
