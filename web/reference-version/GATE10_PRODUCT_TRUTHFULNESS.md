# GATE 10 — PRODUCT TRUTHFULNESS & DEMO READINESS

Data: 15/09/2026. Implementado para Human Review. **Sem commit e sem push.**

## Baseline e escopo

- Branch: `feat/demo-ui`.
- HEAD inicial/final: `bc1b51f35596c998192d34a7bc3268df13d362bc`.
- Working tree inicial continha apenas os arquivos esperados da auditoria aprovada: `PRODUCT_REALITY_AUDIT_POST_GATE9.md`, `evidence/product-reality-audit.cjs`, `evidence/product-reality-interactions.cjs` e `evidence/product-reality/`.
- Esses arquivos foram preservados, sem exclusão ou sobrescrita. Não se exigiu working tree clean porque o próprio Gate autoriza sua presença.
- Alterações restritas à comunicação da UI, remoção de controles sem efeito e mock inacessível. Sem nova capacidade operacional ou mudança de arquitetura dos providers.
- Decisões seguiram o escopo aprovado: remover Save/toggles fictícios e legenda sem representação; manter sandbox local com aviso; manter layout e contratos. Não foi necessário inventar outro mecanismo de Hero Flow.

## Arquivos

Produção, todos em `web/reference-version/src/`:

| Arquivo | Alteração |
|---|---|
| App.tsx | Textos de Telemetry/Network, KPIs, sandbox e Configuration; remoção da legenda e do ramo mock inacessível |
| LivePanels.tsx | Origem e persistência dos gráficos explicitadas |
| monitoring.tsx | Helper de apresentação snapshotLabel; polling e arquitetura intactos |
| SubscribersPage.tsx | Coleção rotulada Snapshot, mantendo horário existente e estados de erro |
| DevicesPage.tsx | Coleção/filtro rotulados Snapshot; guards e requests intactos |

Adicionados: `tests/gate10.cjs`, este relatório e `evidence/gate10/{results.json,claims-search.txt}`. As capturas PNG são locais e ignoradas pelas regras existentes; `git check-ignore` confirmou isso. Nenhuma dependência nova.

## Antes/depois da comunicação

| Item | Antes | Depois |
|---|---|---|
| Telemetry | “Live telemetry · IP pool remains mock” | “Current API counters · browser observations · authoritative IP pool snapshot” |
| Session Activity | Live · observation since opening | “Local observation · this browser”; tooltip explica que reload limpa a janela e não há persistência |
| Events by Type | Live · process counters | “Current API process”; tooltip informa counters reais e reset ao reiniciar API |
| Total Events | Live | “Current API process”; mesmo cálculo dos quatro counters |
| Total Subscribers | Live | Snapshot; tooltip traz Last refreshed e ausência de polling contínuo |
| Total Devices | Live | Snapshot; mesmo tratamento, inclusive na tela/filtro da coleção |
| Network Cells | Configured cells · Campinas | Preservado: STATIC_CONFIG legítimo, três células |
| Network subtítulo | demonstration data | Real session associations · configured cells |
| Topology disclaimer | illustrative positions, not GPS | Preservado integralmente |
| Handover legend | Traço laranja sem trajetória renderizada | Removida; nenhuma animação criada |
| Simulator página | Network event simulator | Local Event Sandbox / Frontend-only simulation / Simulate locally |
| Configuration | Select, checkbox e Save sem efeito nos serviços | Informação somente leitura; controles e Save removidos |

Falhas/loading/stale continuam identificados, sem converter estado inválido em Snapshot ou counters atuais. Nenhum cálculo de KPI/gráfico mudou.

### Simulator

O texto informa que ações alteram somente a página, não fazem requests nem mudam Subscribers, Devices ou Sessions; orienta o Hero Flow real nas telas operacionais e distingue o `cmd/simulator` externo via HTTP. O menu e hash `#simulator` permanecem compatíveis; o título deixa explícito o sandbox. ATTACH/CELL_HANDOVER/DETACH/Clear continuam locais.

**Teste: zero requests backend observados durante os três cliques e Clear.** Os três registros locais apareceram e foram limpos. Isso se refere às ações do sandbox: os providers globais continuam seus polls regulares quando o usuário permanece na aplicação. Não foi criado browser→shell, endpoint, WebSocket ou SSE.

### Configuration e polling efetivo

Sem form editável, botão Save, checkbox de alerts ou persistência simulada. “Not configurable in this demo” e “Event notifications: not implemented in this demo” estão visíveis.

| Recurso | Intervalo mostrado | Origem confirmada |
|---|---|---|
| Telemetry | ~5 s | monitoring.tsx, usePolling(getTelemetry, 5000) |
| Topology | ~5 s, Overview/Network | topology.tsx, TOPOLOGY_INTERVAL=5000 |
| IP Pool | ~5 s, painel aberto | IPPoolCard.tsx, usePolling(getIPPool,5000) |
| Recent Events | ~5 s | recent-events.tsx, setTimeout(poll,5000) |
| Storage | ~10 s | monitoring.tsx, usePolling(getStorage,10000) |
| Health | ~10 s | monitoring.tsx, usePolling(getHealth,10000) |

UI explica que os delays começam após completar request/rodada, não periodicidade exata de parede. Topology também possui retry curto interno enquanto a coleção carrega; a tabela descreve a cadência operacional, não cada transição de inicialização. Subscribers/Devices são snapshots renovados por carga, refresh manual ou operações relacionadas, sem adicionar polling.

O teste lê os call sites/constante acima e confere os valores do DOM. Se a configuração real mudar, exige atualizar conscientemente a apresentação. Não foi criada outra fonte funcional de configuração.

### Remoção de mock morto

Confirmado em `App` que `page === 'Sessions'` renderiza `SessionsPage` e `page === 'Events'` renderiza `RecentEvents full` antes de qualquer chamada a Workspace. Logo, o ramo Sessions/Events de Workspace era inacessível pelo router atual. Foram removidos apenas o array `sessions`, tabela/diálogo desse ramo e seus estados query/selected/headers/rows. Os estados saved/interval/alerts também saíram junto aos controles fictícios.

Nenhuma limpeza ampla de imports, helpers visuais ou arquitetura foi feita. Dados fictícios do sandbox permanecem intencionalmente locais e identificados. Build e navegação das telas reais passaram.

## Testes e evidência real

Ambiente de teste: API MEMORY isolada `127.0.0.1:18120`, frontend `localhost:5220`. DATABASE_URL vazio; nenhum banco do usuário manipulado. Chrome headless/Playwright, usando runtime já instalado, sem instalar dependências.

Comando: `node tests/gate10.cjs` no diretório `web/reference-version`.

Resultado final: **PASS**, nove grupos de verificação cobrindo os dez requisitos de verdade do produto:

1. Ausência do texto IP pool remains mock, mock legado e falso Save.
2. KPIs Snapshot, counters Current API process, Session Activity Local observation.
3. MEMORY legítimo e legenda sem promessa de handover visual.
4. Configuration sem inputs/selects/botões; seis intervalos coerentes com código; ausência de toggle de alerts.
5. Sandbox claramente frontend-only e zero requests disparados pelos cliques/Clear.
6. Provision → Activate → Register → Attach CELL-SP-001 via UI; sessão ativa observada, KPI 1, topologia com conexão, ATTACH no feed, IPPool 1.
7. Handover CELL-SP-002 via UI; confirmação, célula observada e evento atualizados; IPPool continua 1.
8. Detach via UI; DISCONNECTED, preview/topologia sem sessão ativa, KPI 0, DETACH no feed e IPPool 0.
9. Doze páginas nas três resoluções e zero exceções JavaScript não tratadas.

Mesmo subscriber/device foi usado no fluxo. Registro e attach retornaram 201; lifecycle/handover/detach retornaram 200. Respostas estão em `evidence/gate10/results.json`.

Durante desenvolvimento houve um erro TS de inferência de união genérica no rótulo; corrigido passando somente status aos helpers. A primeira rodada do teste UI assumia que Telemetry atualizaria junto com Topology e falhou ao ler o KPI antes do próximo poll. O teste foi corrigido para aguardar o estado real; a API MEMORY isolada foi reiniciada e a rodada final passou. Nenhum provider foi acelerado ou alterado para satisfazer o teste.

## Navegação e responsividade

Overview, Subscribers, Devices, Sessions, Network, Events, Telemetry, Database, System, API Status, Simulator e Configuration: 36 visitas, em 1920×1080, 1440×900 e 1366×768.

Capturadas as cinco páginas exigidas — Overview, Telemetry, Simulator, Configuration e Network — em **todas as três resoluções**, mais attach/handover: 17 PNGs locais. Nomes e medidas estão em results.json. Capturas selecionadas foram inspecionadas visualmente, incluindo Configuration em 1920/1366, Overview em 1366, Telemetry em 1920, Sandbox em 1440/1366 e Network em 1366.

Novos textos cabem nos cards e controles sem sobreposição nas capturas verificadas. Medições das cinco páginas: zero overflow horizontal; zero overflow vertical em Overview/Telemetry/Sandbox/Configuration. Network conserva 18 px internos de scroll vertical preexistente; não houve tentativa de redesign ou alteração do mapa. A escala fixa já existente produz texto pequeno em 1366×768; este Gate não promete resolver essa dívida visual geral.

Logo, skyline/footer, mapa, ícones, rosa dos ventos, posições ilustrativas e associações reais foram preservados. Storage Diagnostics Gate 9 não foi alterado; MEMORY foi testado no navegador. PostgreSQL AVAILABLE/UNAVAILABLE mantém o caminho anterior do endpoint, sem simular conexão real neste Gate.

## Validações finais

| Comando | Resultado |
|---|---|
| go fmt ./... | PASS; normalizou finais de linha em arquivos Go legados |
| go vet ./... | PASS |
| go build ./... | PASS |
| go test -count=1 ./... | PASS em todos os pacotes |
| npm run build | PASS, TypeScript + Vite |
| node tests/gate10.cjs | PASS |
| git diff --check | PASS |
| Race detector | Não repetido, conforme dispensa do Gate sem Go funcional |
| PostgreSQL integration | TEST_DATABASE_URL indisponível; não executado; nenhum banco criado |

A normalização incidental do go fmt não produziu diff de conteúdo Go. Como o baseline não tinha mudanças Go, foi restaurada somente a cópia de trabalho dos arquivos `*.go`; diff Go final vazio. Não houve alteração funcional Go inesperada. Não se afirma uma nova medição de data races.

## Busca final por false claims

Resultado bruto: `evidence/gate10/claims-search.txt`. Classificação das ocorrências relevantes restantes:

- `demo`: limitações verdadeiras de Configuration, sem promessa de capacidade inexistente.
- `simulator`: rota preservada e referência explícita ao CLI externo; sandbox local claramente identificado.
- `placeholder`: dicas de inputs, não fixtures retornadas como dado backend.
- `live`: classes CSS/nomes de componentes e readLabel para recursos realmente consultados periodicamente; coleções agora usam snapshotLabel. Não implica histórico persistente.
- `handover`: operações reais de Sessions, contrato/counters/eventos e exemplos locais do sandbox. Legenda laranja inexistente foi removida.
- `IP pool`: snapshot real, aria-labels e texto de estado; nenhuma afirmação de mock restante em produção.
- `sample`: quantidade de observações reais do browser no aria-label do gráfico, não série fictícia.
- `configuration`: navegação, catálogo estático e apresentação somente leitura.
- `save`, `fake`, `real-time`, `historical`: sem novo controle fictício ou promessa histórica introduzida. Termos em relatórios/testes são contexto de verificação, não UI operacional.

## Garantias de escopo

Nenhuma API, migration, schema, dependência, banco, persistência, autenticação, Google Maps, integração backend do Simulator, WebSocket/SSE ou event bus criado. Nenhuma mudança na semântica do IPPool, retenção de eventos ou Session.

Dívidas SUSPENDED/Attach, compatibilidade LTE/5G e MSISDN/VARCHAR continuam para decisão própria do Human Review. Nenhum Gate 11 iniciado.

## Git

`git diff --stat` (arquivos rastreados; novos relatório/teste/evidências não aparecem nesta estatística):

```text
 web/reference-version/src/App.tsx             | 33 +++++++--------------------
 web/reference-version/src/DevicesPage.tsx     |  8 +++----
 web/reference-version/src/LivePanels.tsx      |  4 ++--
 web/reference-version/src/SubscribersPage.tsx |  4 ++--
 web/reference-version/src/monitoring.tsx      |  4 ++++
 5 files changed, 20 insertions(+), 33 deletions(-)
```

Estado final abaixo inclui arquivos preexistentes da auditoria, preservados. Nada foi staged/commitado/publicado.
```text
 M reference-version/src/App.tsx
 M reference-version/src/DevicesPage.tsx
 M reference-version/src/LivePanels.tsx
 M reference-version/src/SubscribersPage.tsx
 M reference-version/src/monitoring.tsx
?? reference-version/GATE10_PRODUCT_TRUTHFULNESS.md
?? reference-version/PRODUCT_REALITY_AUDIT_POST_GATE9.md
?? reference-version/evidence/gate10/
?? reference-version/evidence/product-reality-audit.cjs
?? reference-version/evidence/product-reality-interactions.cjs
?? reference-version/evidence/product-reality/
?? reference-version/tests/gate10.cjs
```

**PARADO PARA HUMAN REVIEW.**
