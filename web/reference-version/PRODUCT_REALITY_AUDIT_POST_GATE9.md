# PRODUCT REALITY AUDIT — POST GATE 9

Data: 15/09/2026. Auditoria somente leitura de produção. **Este não é o Gate 10.**

## 1. Baseline e método

- Branch: `feat/demo-ui`.
- HEAD: `bc1b51f35596c998192d34a7bc3268df13d362bc`.
- Working tree inicial: CLEAN.
- Fonte primária: código Go e `web/reference-version/src`, execução real e navegador. Relatórios anteriores não foram tratados como prova suficiente.
- API MEMORY isolada em `127.0.0.1:18119`; Vite em `localhost:5219`, via `scripts/dev.mjs`, com DATABASE_URL vazio. Nenhum banco do usuário foi alterado.
- Chrome headless/Playwright: 12 rotas × 3 viewports = 36 visitas. Evidências em `evidence/product-reality/`; scripts em `evidence/product-reality-audit.cjs` e `evidence/product-reality-interactions.cjs`.
- Limite da evidência: visitas cobrem abertura e conteúdo; não constituem teste exaustivo de todo botão, todos os erros e todos os tamanhos de coleção. Onde indicado “código”, não houve reprodução isolada no navegador.

## 2. Validação

| Verificação | Resultado |
|---|---|
| go vet ./... | PASS |
| go build ./... | PASS |
| go test -count=1 ./... | PASS, todos os pacotes |
| npm run build | PASS, TypeScript e Vite |
| CLI simulator, 1 device por HTTP na API isolada | PASS, 1/1, 100%, zero falhas |
| Exceções JavaScript nas visitas/fluxos | 0 |
| Race detector | Não repetido: dispensado pelo escopo somente leitura; não alegamos nova medição de races |
| PostgreSQL | TEST_DATABASE_URL indisponível; PostgreSQL integration tests not executed |

Nenhum código de produção foi alterado. Não foi necessário restaurar alteração incidental. Artefatos de build são ignorados pelo Git.

## 3. Product Reality Matrix

Fontes frontend abaixo são relativas a `web/reference-version/src`; backend relativo à raiz do repositório. Rastreabilidade dos endpoints está na seção 4.

| Area | Element | Classification | Source | User-visible issue | Recommended next action |
|---|---|---|---|---|---|
| Overview | API/SYSTEM ONLINE | REAL | App + monitoring → api.ts → /health | Liveness não significa prontidão completa | Manter distinção explícita |
| Overview | Storage MEMORY/POSTGRES | REAL | monitoring → storage-api → /api/v1/system/storage | MEMORY é modo válido, não falha de DB | Preservar |
| Overview | Total Subscribers | DERIVED_REAL | subscribers provider → GET subscribers → length | Atualização não é polling contínuo; rótulo Live pode sugerir mais | Explicitar freshness |
| Overview | Total Devices | DERIVED_REAL | devices provider → GET devices → length | Mesmo limite; mudanças externas exigem refresh | Explicitar freshness |
| Overview | Active Sessions | REAL | monitoring → /telemetry → repository count | Snapshot periódico | Preservar |
| Overview | 3 Network Cells | STATIC_CONFIG | network-catalog.ts e internal/network/cell.go | Não é descoberta/contagem de antenas reais | Preservar “Configured” |
| Overview | Total Events | DERIVED_REAL | soma dos quatro counters /telemetry | Não é total histórico persistido | Preservar indicação de processo |
| Network | Associações Device/Session/Cell | REAL | topology.tsx → session-api → active session lookup | Poll individual por device conhecido | Documentar custo e freshness |
| Network | Agrupamento, aliases D-xx, linhas verdes | DERIVED_REAL | LiveTopology.tsx sobre snapshots confirmados | Aliases dependem da lista; não são identidade física | Manter IDs nos detalhes |
| Network | Coordenadas das torres e dos devices | DECORATIVE | network-catalog.ts / LiveTopology.tsx | Não GPS | Manter aviso ilustrativo |
| Network | Fundo Campinas, rosa dos ventos | DECORATIVE | /maps/campinas.jpg; SVG/CSS | Imagem geográfica estática, não Google API nem telemetria espacial | Manter atribuição |
| Network | Legenda Handover laranja | INCOMPLETE | App.tsx / LiveTopology.tsx | Não há trajetória laranja dinâmica correspondente | Remover promessa ou definir visual futuramente |
| Overview | Active Sessions preview | DERIVED_REAL | topologia, connected confirmadas, primeiras 5 | Manage não seleciona o device automaticamente | Deep-link futuro |
| Events | Recent Events | REAL | recent-events.tsx → /api/v1/events/recent | Últimos 100, processo atual, observação assíncrona | Preservar aviso |
| Telemetry | Session Activity | LOCAL_OBSERVATION | monitoring observations → LivePanels | Histórico começa ao abrir; perdido no reload | Não vender como histórico autoritativo |
| Telemetry | Events by Type | DERIVED_REAL | counters /telemetry → proporções | Reseta com API; não distribuição histórica persistida | Preservar |
| Pool | capacidade/alocados/disponíveis | REAL | ip-pool-api → endpoint → mesmo IPPool | Alocação pequena quase invisível na barra /16 | Valor textual já explica |
| Pool | porcentagem | DERIVED_REAL | allocated/capacity | Sem 17/256 fictício | Preservar |
| Telemetry | texto “IP pool remains mock” | BROKEN | App.tsx, subtítulo da página | Contradiz Gate 8 e dados reais | Corrigir texto em próximo gate |
| Subscribers | coleção, detalhe, lifecycle | REAL | SubscribersPage/subscribers/subscriber-api | Sem histórico completo de transições | Fluxo principal utilizável |
| Devices | coleção global/filtrada, registro/detalhe | REAL | DevicesPage/devices/device-api | Sem gerenciamento de inativação de device | Não sugerir operação inexistente |
| Sessions | attach/handover/detach | REAL | SessionsPage/session-api | Seleção manual e leitura individual, sem lista global | Melhorar continuidade do fluxo |
| Sessions | consulta histórica pelo ID | INCOMPLETE | backend GET sessions/{id} sem consumer UI | Histórico não recuperável pela tela após refresh | Evolução separada |
| System/API Status | liveness e métricas de requests | REAL | LivePanels/monitoring | Não equivale a painel de CPU, memória ou readiness | Manter nomes precisos |
| Database | modo/configuração/reachability | REAL | storage-api + storage handler | PG real não validado neste audit | Validar em ambiente próprio depois |
| Simulator UI | ATTACH/HANDOVER/DETACH local | MOCK | App.tsx Workspace, simEvents, UE-01 e células literais | Não opera rede; aviso explícito existente | Diferenciar claramente do CLI |
| Configuration | intervalo/alertas/Save | INCOMPLETE | App.tsx state local | Não muda polling nem notificações, não persiste | Remover falsa expectativa |
| App | mock sessions legado | MOCK | App.tsx const sessions e ramo antigo Workspace | Código morto: rotas atuais usam páginas especializadas | Limpeza futura, não confundir com runtime |
| App | logo, skyline/footer, frase, ícones | DECORATIVE | App.tsx e CSS | Nenhum estado operacional implícito | Preservar |
| App | data de observação/tempo de sessão mostrado | LOCAL_OBSERVATION | receivedAt e relógio do browser sobre timestamps | Tempo de snapshot, não relógio autoritativo contínuo | Identificar observação |

## 4. Backend surface e rastreabilidade

São **19 registros de rotas**. Query parameters não criam novas rotas.

| Method | Path | Propósito / consumer frontend |
|---|---|---|
| GET | /health | Liveness; api.ts → monitoring → topbar/System/API Status |
| GET | /telemetry | Snapshot/counters; api.ts → monitoring → KPIs/gráficos/System |
| GET | /api/v1/system/storage | Diagnóstico; storage-api.ts → monitoring → Database/badges |
| GET | /api/v1/network/ip-pool | Snapshot allocator; ip-pool-api.ts → painel de pool |
| GET | /api/v1/events/recent | Feed limitado; recent-events.tsx → RecentEvents/Events |
| GET | /api/v1/subscribers | Coleção ou ?imsi exato; subscriber-api.ts → subscribers/SubscribersPage |
| GET | /api/v1/subscribers/{id} | Detalhe; mesmo módulo |
| POST | /api/v1/subscribers | Provision; mesmo módulo |
| POST | /api/v1/subscribers/{id}/activate | Activate; mesmo módulo |
| POST | /api/v1/subscribers/{id}/suspend | Suspend; mesmo módulo |
| POST | /api/v1/subscribers/{id}/deactivate | Deactivate; mesmo módulo |
| GET | /api/v1/devices | Global ou ?subscriber_id; device-api.ts → devices/DevicesPage |
| GET | /api/v1/devices/{id} | Detalhe; mesmo módulo |
| POST | /api/v1/devices | Register; mesmo módulo |
| GET | /api/v1/sessions?device_id={id} | Sessão ativa individual; session-api.ts → SessionsPage e topology |
| GET | /api/v1/sessions/{id} | Sessão por UUID; sem consumer frontend atual |
| POST | /api/v1/sessions/attach | Attach; session-api.ts → SessionsPage |
| POST | /api/v1/sessions/{id}/handover | Handover; mesmo módulo |
| POST | /api/v1/sessions/{id}/detach | Detach; mesmo módulo |

Backend: subscribers/devices/sessions seguem `internal/<domínio>/handler.go → service.go → repository` MEMORY ou PostgreSQL, compostos em `cmd/api/main.go`. Health vem de `internal/platform/httpserver/server.go`. Telemetry vem de `internal/telemetry/handler.go`, métricas e worker; recent vem de `internal/telemetry/recent.go`. Pool: `internal/network/ippool_status.go → IPPool.Snapshot`, mesma instância usada por Sessions. Storage: `internal/platform/storage/handler.go`, mesmo pool PostgreSQL da aplicação, sem pool adicional.

Transportes frontend identificados: `api.ts`, `subscriber-api.ts`, `device-api.ts`, `session-api.ts`, `storage-api.ts`, `ip-pool-api.ts`, `recent-events.tsx`. Providers não substituem falhas por fixtures. O 404 de active session com código esperado é ausência legítima, não erro convertido indevidamente em zero. Estados stale/error não são dados atuais confirmados.

Nenhuma chamada a endpoint inexistente identificada no código. Há sobreposição legítima de leitores: telemetry, topology e Sessions têm finalidades distintas. Topology faz O(N) consultas, concorrência limitada a 4; não há endpoint agregado de rede. Refresh da coleção e retry do filtro de Devices podem provocar releitura adicional pela dependência de receivedAt: oportunidade de revisão, não uma regressão de contrato comprovada por contagem de requests neste audit.

## 5. Navegação, estados e ações

Todas as 12 rotas abriram em todos os viewports. Evidência textual por visita: browser-audit.json.

| Página | Conteúdo/estado vazio observado | Ações e limites |
|---|---|---|
| Overview | KPIs 0, sem sessão/eventos, pool vazio válido | Manage/View all navegam; cards são displays |
| Subscribers | Coleção vazia adequada | Provision/Activate exercitados; busca local, exata, refresh, detalhes e lifecycle presentes no código |
| Devices | Coleção vazia adequada | Registro exercitado, filtro por subscriber e busca implementados |
| Sessions | Solicita seleção; ausência de sessão é válida | Attach/Handover/Detach exercitados com respostas reais |
| Network | Torres configuradas sem inventar devices conectados | Zoom/registro/nodes são interação local; mapa grande exige scroll |
| Events | “No events observed in this API execution” | Feed real depois do fluxo; detalhe de evento no código, sem filtro avançado |
| Telemetry | Linha em coleta/zero e donut sem eventos | Dados reais/local observation; subtítulo errado |
| Database | MEMORY legítimo, sem falso PG online | Read-only, refresh por provider |
| System | Status HTTP e dados observados | Não implementa gestão de serviços |
| API Status | Status HTTP real | Não é catálogo interativo de todas as APIs |
| Simulator | Sem eventos simulados inicialmente | Botões só alteram texto local; zero POST observado |
| Configuration | Preferências locais | Save confirma somente tela; retorno à rota volta para 5 segundos |

Loading/error: providers têm estados explícitos e controles de escrita bloqueados enquanto busy/sem leitura confiável. Neste audit foi injetado HTTP 503 **somente no navegador** para IPPool: UI mostrou Unavailable, HTTP 503 e travessões, sem inventar snapshot zero. Não foi executada matriz completa de falhas/latência para todas as rotas; sua cobertura restante é inspeção de código, não aprovação runtime abrangente.

| Ação | Classificação | Evidência |
|---|---|---|
| Navegação sidebar, gear, Manage, View all | FUNCTIONAL | Sidebar exercitada; destinos restantes conferidos no código |
| Provision, Activate, Register, Attach, Handover, Detach | FUNCTIONAL | UI e respostas backend, JSON de interação |
| Suspend/Deactivate, detalhes, refresh, buscas/filtros | FUNCTIONAL | Implementação encaminha ação correta; nem todas exercitadas individualmente |
| Selecionar subscriber não ACTIVE / enviar enquanto busy / handover mesma célula | DISABLED_BY_DESIGN | Guards explícitos |
| Tooltip de device, registro da topologia, zoom +/−, detalhes de evento | FUNCTIONAL | Interações locais implementadas; não escrevem no backend |
| Simulator ATTACH/CELL_HANDOVER/DETACH/Clear | VISUAL_ONLY | Append/clear de estado local; zero POST nos três eventos testados |
| Configuration Save e preferências | VISUAL_ONLY | Estado local; valor 30 voltou a 5 após navegar |
| KPI cards, rosa dos ventos, branding e Build/Learn/Simulate/Explore | VISUAL_ONLY | Displays/decoração; não são operações implementadas |

Não foi encontrado bloqueador de escrita no caminho feliz testado. Isso não declara todos os controles integralmente homologados.

## 6. Hero Flow e Simulator

**Veredito: A — totalmente pela UI.** Provisionou-se IMSI 724059999999901, ativou-se subscriber, registrou-se device IMEI 860010009999902 no mesmo subscriber, fez-se Attach em CELL-SP-001, Handover para CELL-SP-002 e Detach. Respostas: registro/attach 201, handover/detach 200. UI apresentou CONNECTED e depois DISCONNECTED/VOLUNTARY_DETACH; pool retornou a zero e eventos/counters aumentaram.

A primeira rodada de navegador compartilhou o ambiente com uma execução CLI e selecionou o primeiro registro, que era do CLI. Por isso uma segunda rodada usou seleção explícita do subscriber/IMEI; `interaction-audit.json` é a prova do encadeamento no mesmo proprietário. Não ocultamos essa diferença entre rodadas.

Atrito: navegar entre páginas, selecionar owner/device manualmente, Manage não faz deep-link, não há lista/histórico global de Sessions; coleções não acompanham automaticamente todos os registros externos. Polling/event worker introduzem atraso de observação. Um attach/detach muito rápido pode gerar eventos sem um ponto não-zero no gráfico de sessões.

`cmd/simulator/main.go`, `runner.go`, `client.go`: compila; usa HTTP externo com contexto/timeouts, sem manipular repositórios diretamente. Execução real `go run ./cmd/simulator -api http://127.0.0.1:18119 -devices 1`: Provision → Activate → Register → Attach001 → Handover002 → Detach, 1 completado, zero falhas. A UI Simulator não chama esse CLI nem inicia binário. Não existe mecanismo browser→shell e não foi criado.

## 7. Gráficos, inventários e persistência

- Session Activity: /telemetry a cada ~5s; timestamp real do snapshot e active_sessions. Frontend guarda no máximo 360 pontos/30 minutos; gaps por falha ou intervalo >15s. Estado do provider atravessa navegação, mas reload o elimina. Não é consulta histórica ao servidor. Se timestamp não avança, reinicia buffer; isso não detecta universalmente restart de API. Após restart com relógio avançando podem coexistir observações pré/pós-restart na janela até expirar. Comportamento de restart confirmado pelo código, não reinício provocado neste audit.
- Events by Type: quatro counters do processo, ATTACH/CELL_HANDOVER/DETACH/STALE_DISCONNECT, proporções calculadas. Reload refaz snapshot; restart do processo zera counters. Recent Events é buffer separado de até 100 eventos, não deve ser comparado ao total como se fosse histórico completo.
- IPPool: capacidade 65.533 e alocados/disponíveis autoritativos. Percentual muito pequeno pode parecer barra vazia, embora texto correto. Não há valor fictício de warm-up.
- MOCK: Simulator local com UE-01 e eventos literais; array legado de sessions em App.tsx, ramo inacessível pelas rotas especializadas atuais. Não classificar telas atuais de Sessions/Events como mock por causa desse código morto.
- STATIC_CONFIG: três células em network-catalog.ts/backend cell.go, opções LTE/5G, catálogo de rotas, limites de polling/buffer. Literais de validação, cores e limites não são por si mocks.
- DECORATIVE: skyline/footer, logo, mapa estático, compass, posições ilustrativas. A atribuição do mapa é visível; não há Google Maps ativo.
- LOCAL_OBSERVATION: histórico Session Activity, receivedAt/última leitura, duração calculada no snapshot. Não persistidos pelo browser nem convertidos em histórico backend.

## 8. Responsividade

Base visual fixa 1536×1024 com escala uniforme limitada a 1, centralizada. Não é reflow fluido.

- 1920×1080: Overview cabe; margens laterais grandes esperadas pela escala limitada. Screenshot overview-live.png inspecionado.
- 1440×900: Overview cabe, cards preservados; letras de tabelas/legendas pequenas pela escala. overview-1440.png inspecionado.
- 1366×768: escala 0,75; navegação cabe, mas densidade/textos miúdos prejudicam leitura. Network é mais alto que a área inicial: controles inferiores ficam parcialmente fora da captura antes de scroll; não confundir ajuste do Overview com ajuste da página Network. network-1366.png inspecionado.
- Visitas DOM em todas as 36 combinações registraram geometria do documento, mas `overflow:hidden` no root não prova ausência de clipping interno. Screenshots foram gerados para Overview/Network/Configuration nas três resoluções; não foi feita inspeção visual individual de screenshot de cada uma das 36 combinações.
- Sem quebra estrutural evidente no Overview inspecionado. Não há evidência suficiente para certificar tabelas longas, tooltips abertos e todos os modais nas três resoluções. Isso fica como limite explícito, não como “responsividade 100% aprovada”.

## 9. Dívidas de domínio e PostgreSQL

| Dívida | Estado confirmado |
|---|---|
| A. SUSPENDED + device REGISTERED pode Attach | Sim pelo caminho de código: deviceCheckerAdapter em cmd/api/main.go verifica registro do device, não estado atual do subscriber. CheckSubscriberActive atua no registro do device. Não reproduzido separadamente neste audit. Decisão de domínio pendente |
| B. LTE/5G versus Cell | Não há enforcement de compatibilidade no contrato DeviceInfo usado por Session. UI permitiu device LTE fazer handover para CELL-SP-002 5G; CLI registra 5G e faz attach001 LTE. Confirmado runtime e código |
| C. MSISDN/VARCHAR | Inconsistente: subscriber.go aceita + opcional e até 15 dígitos, mas migrations/000001 usa VARCHAR(15); +15 dígitos soma 16 caracteres. PostgreSQL não executado para reproduzir |
| D. IPPool holes/restart | Correção preservada: MarkAllocated não avança nextHost; Allocate ignora reservados. Testes Go passaram, warm-up em cmd/api reaproveita IPs. Restart real PostgreSQL não validado |
| E. Recent Events | Buffer em memória limitado a 100, reinicia com processo; worker assíncrono. Não é event store durável; não promete isso |

PostgreSQL é suportado por código nos repositórios, startup/schema check e warm-up; isso é distinto de validação neste audit. TEST_DATABASE_URL ausente: **PostgreSQL integration tests not executed**. Nenhum banco, schema ou migration criado. Falhas de startup PostgreSQL não fazem fallback silencioso para MEMORY. Storage Diagnostics usa o pool existente e ping com timeout; /health continua somente liveness.

## 10. Backlog priorizado

**P0 — bloqueador da demonstração feliz:** nenhum encontrado no Hero Flow MEMORY executado. Não significa ausência de riscos em outros cenários.

**P1 — alto valor:**

1. Verdade de produto: corrigir subtítulo mock obsoleto; distinguir Simulator local do CLI; tornar Configuration honesta sobre ausência de efeito; adequar legenda Handover.
2. Freshness/continuidade: expor snapshots de coleções, simplificar seleção e deep-link Device→Session; medir polling por device antes de escalar.
3. Human Review de regras SUSPENDED/Attach e LTE/5G; testes de domínio após decisão, sem inventar política nesta auditoria.
4. Resolver MSISDN versus VARCHAR e validar PostgreSQL em ambiente de integração separado, antes de declarar paridade MEMORY/PG.
5. Completar homologação de estados de erro/loading, ações secundárias e coleções maiores no navegador.

**P2 — evolução/polish:**

- Legibilidade a 1366×768 e scroll de Network; acessibilidade/nomes dos controles de zoom.
- Limpar código mock inacessível e melhorar ocupação visual de páginas esparsas.
- Explorar consulta de sessão por UUID já existente sem confundir com lista global inexistente.
- História persistente de eventos/gráficos e descoberta geográfica são novas capacidades, não pequenas correções.

## 11. Única recomendação

**RECOMMENDED GATE 10:** Product Truthfulness & Demo Readiness.

**WHY:** o Hero Flow real já funciona; os maiores desencontros imediatos são promessas visuais/textuais e controles locais que parecem operacionais. Corrigi-los reduz ambiguidade sem expandir backend.

**EXACT SCOPE:** ajustar somente comunicação da Telemetry/IPPool, rotular inequivocamente Simulator local versus CLI, remover/desabilitar com explicação preferências sem efeito, adequar legenda de handover e identificar snapshots de coleções. Validar novamente o fluxo feliz e essas interações no browser. Não construir integração de Simulator neste gate.

**OUT OF SCOPE:** novas APIs, histórico persistente, Google Maps, autenticação, schema/migrations, novas regras de Subscriber ou LTE/5G, execução de shell no browser, redesign amplo, novos módulos.

**Não iniciado.** Proposta para Human Review, não autorização presumida.

## 12. Resposta ao recrutador

Hoje é possível provisionar e ativar um subscriber, registrar seu device, abrir sessão com IP, trocar célula e encerrar sessão totalmente pela UI, observando estado real via HTTP, topologia ilustrativa vinculada às sessões, eventos e IPPool. Também é possível executar o Hero Flow pelo CLI externo. Na auditoria isso foi demonstrado em MEMORY.

Não deve ser apresentado como rede física/GPS, histórico persistido de telemetria, Simulator UI integrado ou console completo de configuração. PostgreSQL possui implementação, mas não foi validado aqui. O projeto já demonstra integração frontend/backend e lifecycle de telecom simulado; ainda precisa das correções de transparência e validações delimitadas acima para uma apresentação sem ressalvas.

## 13. Encerramento

Somente este relatório e evidências/scripts de auditoria foram adicionados. Nenhuma alteração funcional, commit, push ou Gate 10. Working tree final deve listar exclusivamente este relatório, scripts product-reality e pasta evidence/product-reality; screenshots podem permanecer ignorados pelas regras existentes. **PARADO PARA HUMAN REVIEW.**
