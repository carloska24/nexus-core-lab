# Gate 2 — Frontend transport + Health + Telemetry

Data: 11/09/2026. Entregue para **HUMAN REVIEW**. Sem commit ou push.

## Escopo e decisões

O Gate 1 aprovado e o pedido do Gate 2 definiram o projeto: integrar somente as leituras reais de `/health` e `/telemetry` no frontend isolado. A implementação usa fetch nativo, proxy Vite e Context/estado React, sem bibliotecas novas. Não há integração de Subscribers, Devices, Sessions ou Recent Events na aplicação.

Premissas: laboratório local, uma API alvo por frontend, polling leve, disponibilidade HTTP observacional e estado em memória no cliente. Os testes de escrita usam exclusivamente uma instância Go de teste iniciada com `DATABASE_URL` vazio. Nenhuma credencial ou conexão de banco é enviada ao browser.

Decisões: proxy em vez de CORS no Go; fetch em vez de Axios; provider compartilhado em vez de polling por card; agendamento após conclusão em vez de setInterval sobreposto; snapshot anterior marcado stale em vez de zero em falha; janela local em vez de inventar histórico. O layout aprovado continua com seu canvas e escala existentes. Ajustes visuais limitados a estados, rótulos de mock e espaço para o nome real STALE_DISCONNECT.

## 1. Arquivos frontend criados

- [vite.config.ts](vite.config.ts): proxy de desenvolvimento.
- [src/api.ts](src/api.ts): contratos, validação de resposta e duas leituras HTTP.
- [src/monitoring.tsx](src/monitoring.tsx): provider, polling, estados e observações em memória.
- [src/LivePanels.tsx](src/LivePanels.tsx): gráficos reais e apresentação honesta de health/Database.
- [src/transport.css](src/transport.css): cores de estado, rótulos mock e ajuste da legenda do donut.
- [tests/gate2.cjs](tests/gate2.cjs): validação de navegador e operações na API isolada.
- Este relatório e arquivos em [evidence/gate2](evidence/gate2).

## 2. Arquivos modificados

- [src/App.tsx](src/App.tsx): ligação do provider; topbar/sidebar; KPIs; substituição dos gráficos artificiais; identificação do conteúdo que permanece mock; apresentação de System/API/Database baseada apenas no suporte atual.
- `dist/`: saída gerada por `npm run build`.

Não foram alterados `package.json`, locks, a versão original, arquivos de mapa, ReferenceArt, CSS base, Go ou configuração de banco. Não foram instalados pacotes. Os registros de eventos e sessões da UI continuam mock, sem chamadas aos endpoints de domínio.

## 3. Proxy

`vite.config.ts` encaminha `/health` e `/telemetry`, sem reescrever os paths. Target padrão: `http://127.0.0.1:8080`. Override central: variável de ambiente `NEXUS_API_TARGET` no processo Vite. Os componentes não contêm host da API.

O teste utilizou Vite `http://127.0.0.1:5176` → Go `http://127.0.0.1:18080`, preservando serviços eventualmente existentes em outras portas.

Para desenvolvimento normal, com a API já disponível em 8080, executar na pasta `web/reference-version`:

```powershell
npm run dev -- --host 127.0.0.1 --port 5175
```

Para reproduzir o ambiente de teste, em um terminal na raiz do repositório:

```powershell
$env:PORT='18080'
$env:DATABASE_URL=''
go run ./cmd/api
```

Em outro terminal, na pasta `web/reference-version`:

```powershell
$env:NEXUS_API_TARGET='http://127.0.0.1:18080'
npm run dev -- --host 127.0.0.1 --port 5176 --strictPort
```

O proxy é de **desenvolvimento**. O build estático precisa de roteamento mesma origem no ambiente de publicação; este Gate não implementa deploy ou reverse proxy de produção.

## 4. Contratos TypeScript

`Health`: `status: string`, `service: string`.

`Telemetry` preserva exatamente os campos JSON auditados:

```ts
interface Telemetry {
  service: string;
  timestamp: string;
  requests_total: number;
  metrics: {
    active_sessions: number;
    connected_devices: number;
    dropped_events_total: number;
  };
  events_total: {
    attach: number;
    cell_handover: number;
    detach: number;
    stale_disconnect: number;
  };
}
```

Não foram acrescentados campos de catálogo, histórico, pool ou readiness. As respostas também são validadas em runtime: campos obrigatórios, timestamp interpretável e contadores inteiros não negativos e seguros em JavaScript. HTML do proxy ou JSON inválido são falha, nunca sucesso com zeros.

## 5. getHealth

`getHealth(signal?: AbortSignal): Promise<Health>` faz GET relativo `/health`, sem cache, aceita JSON e verifica HTTP, contrato, `status === 'ok'` e serviço `nexus-core-lab`. Falha de rede, HTTP ou payload é propagada ao estado de leitura.

## 6. getTelemetry

`getTelemetry(signal?: AbortSignal): Promise<Telemetry>` faz GET relativo `/telemetry`, sem cache. Erros HTTP preservam status, message e code quando disponíveis no JSON Go. Uma falha sem JSON usa o status HTTP. Não há retorno alternativo com 0/[] nem preenchimento de campos ausentes.

## 7. Polling

- Health: leitura imediata e nova tentativa 10 segundos após conclusão.
- Telemetry: leitura imediata e nova tentativa 5 segundos após conclusão.
- Timeout por leitura: 8 segundos, via AbortController.
- Uma subscription por recurso para a aplicação inteira; navegar entre páginas não multiplica timers.
- Sem sobreposição: a próxima tentativa só é agendada no finally da leitura anterior.
- Cleanup cancela timer, aborta leitura e impede atualização após unmount.
- Sem WebSocket, SSE, Redux, Zustand ou localStorage.

O polling inclui o tempo da requisição no período efetivo. O contador requests_total inclui essas próprias consultas, conforme o middleware Go existente.

## 8. Loading / success / error / stale

| Estado | Comportamento |
|---|---|
| loading | Sem snapshot: `—` nos KPIs e indicação Loading; health neutro |
| success | Snapshot validado, rótulo Live; health online |
| error | Sem sucesso anterior: Unavailable e `—`, sem zero artificial |
| stale | Falha após sucesso: mantém último snapshot, rótulo amarelo `Stale · last known` |

Health falho produz SYSTEM OFFLINE e indicador API offline mesmo com snapshot anterior de health preservado. Health e telemetry são independentes: telemetry 500 não inventa falha de health; health bem-sucedido também não torna telemetry bem-sucedida.

O status representa **disponibilidade HTTP observada**, não readiness completo. Database permanece neutro/UNKNOWN, sem inferência de PostgreSQL saudável.

## 9. Elementos que agora usam dados reais

- Status API da topbar e sidebar; SYSTEM ONLINE/OFFLINE.
- Data apresentada como horário da última leitura bem-sucedida de health.
- Active Sessions KPI: `metrics.active_sessions`.
- Total Events KPI: soma dos quatro campos de events_total.
- Events by Type em Overview e Telemetry: ATTACH, CELL_HANDOVER, DETACH, STALE_DISCONNECT. OTHER removido. Fatias/cores/percentuais são calculados das mesmas contagens; total zero mostra anel neutro e No events.
- Session Activity em Overview e Telemetry: observações reais de timestamp/active_sessions desde a abertura.
- System/API Status: health, última leitura e requests_total quando disponível; Database apresenta UNKNOWN.

Os percentuais mock +21% e +35% foram retirados dos KPIs reais. A janela de atividade é limitada aos últimos 30 minutos observados, no máximo 360 pontos; não carrega passado, não inclui série detached inventada. Até dois pontos há Collecting telemetry. Falhas ou intervalos longos quebram a linha, em vez de preencher a lacuna. A janela permanece durante navegação e reinicia ao recarregar a aplicação.

## 10. Elementos que continuam mock

- Total Subscribers, Total Devices e Network Cells, definidos como mock no código e identificados na apresentação.
- Tabelas de sessões e eventos, respectivas telas e detalhes; topologia e dispositivos demonstrativos.
- IPPool/warm-up, incluindo os números da referência, explicitamente marcados mock: não representam o pool real.
- Simulator da UI e preferências de Configuration continuam locais/demonstrativos.

Mapa/arte/controles e composição não foram substituídos. Nenhum gap P1/P2 foi implementado. As operações de domínio usadas **no teste externo** não significam integração dessas telas.

## 11. API offline

Sem sucesso prévio: SYSTEM OFFLINE, API em cor de falha, métricas Unavailable/`—`. Após sucesso: mesmos estados de falha no health, métricas anteriores preservadas como stale. Database permanece neutro nos dois casos. Quando as leituras voltam a funcionar, o polling recupera os estados automaticamente.

Os cenários de indisponibilidade/rede e telemetry 500 foram injetados pelo Playwright no navegador, sem desligar serviços existentes nem provocar falha no PostgreSQL. As leituras online e operações de domínio foram contra Go real via proxy.

## 12. Build

`npm run build`: **PASS**, TypeScript e Vite.

```text
vite v6.4.3 building for production...
✓ 1594 modules transformed.
dist/index.html                   0.70 kB │ gzip:  0.40 kB
dist/assets/index-B8pIwa9c.css     24.59 kB │ gzip:  6.49 kB
dist/assets/index-vtCBlZOB.js     179.57 kB │ gzip: 57.33 kB
✓ built in 14.04s
```

Houve uma falha inicial de fechamento JSX corrigida antes do build final. O sandbox também impediu a leitura de diretórios ancestrais pelo esbuild; build e Vite passaram ao executar com permissão local adequada, sem workaround no Go.

## 13. Testes

Não existia script de testes frontend no package.json. Foi criado e executado `node tests/gate2.cjs`, com o Playwright já disponível no runtime desktop, sem instalação no projeto. O caminho do módulo pode ser substituído pela variável `PLAYWRIGHT_MODULE` para outro ambiente.

**12 verificações passaram**, registradas em [test-results.json](evidence/gate2/test-results.json): erro inicial sem zeros, leituras reais, attach observado, handover/detach observados, telemetry 500/stale, rede offline, recuperação, viewports menores, categorias reais na tela Telemetry, timeout sem overlap, encerramento do polling ao fechar a página e ausência de erros JS não tratados.

Uma primeira execução parou porque o seletor de teste leu o SVG do ícone em vez do gráfico. Corrigido o seletor, a execução completa passou. Essa tentativa deixou uma sessão ativa na instância isolada em memória, explicando o baseline 1 da evidência final; nenhum banco persistente foi afetado.

`go test ./...`: **PASS**, resultados dos pacotes de teste em cache. `cmd/api`, `cmd/migrate` e `internal/platform/postgres` não possuem testes. `TEST_DATABASE_URL` não estava definido; testes de integração PostgreSQL condicionais não foram executados contra banco. Não se afirma validação PostgreSQL neste Gate.

## 14. Evidência de alteração real observada pela UI

Foram executados requests reais existentes, externamente à interface, na API isolada: provisionar, ativar, registrar device, attach, handover e detach. Nenhuma mudança em cmd/simulator foi necessária.

| Momento | active_sessions | attach | cell_handover | detach | stale_disconnect | Total Events |
|---|---:|---:|---:|---:|---:|---:|
| Antes | 1 | 1 | 0 | 0 | 0 | 1 |
| Após attach | 2 | 2 | 0 | 0 | 0 | 2 |
| Após handover/detach | 1 | 2 | 1 | 1 | 0 | 4 |

O teste esperou esses valores no DOM dos KPIs e capturou o dashboard. Snapshots completos: [real-telemetry.json](evidence/gate2/real-telemetry.json). Os números são evidência de teste, não fixtures incorporadas à aplicação.

## 15. Screenshots e revisão visual

- [1920×1080 — online real](evidence/gate2/1920x1080-online.png).
- [1920×1080 — telemetry 500/stale](evidence/gate2/1920x1080-telemetry-stale.png).
- [1920×1080 — offline/stale](evidence/gate2/1920x1080-offline.png).
- [1440×900](evidence/gate2/1440x900.png).
- [1366×768](evidence/gate2/1366x768.png).

Capturas feitas por Chrome real via Playwright; online, offline e 1366×768 foram inspecionadas visualmente. As verificações de dimensões confirmaram que o canvas cabe nos dois viewports menores. A composição, diagonal sidebar/topbar, mapa, containers e footer foram preservados. As diferenças são as exigidas pelos dados reais e seus estados.

## 16. git diff --stat

Executado na raiz do repositório, retornou **sem saída**. Isso não significa ausência de trabalho frontend: `web/` já estava inteiro untracked antes deste Gate. Não foi feito git add para forçar um diff.

## 17. git status --short

Executado na raiz:

```text
?? web/
```

Esse estado agregado já existia no início. A lista de arquivos deste Gate está nos itens 1–2; não houve staging, commit ou push.

## 18. Confirmação de isolamento

**ZERO alterações de código Go/backend, PostgreSQL, migrations, compose.yaml, go.mod ou go.sum.** `git diff --name-only -- cmd internal migrations compose.yaml go.mod go.sum` não retornou arquivos. Nenhuma operação SQL foi executada; as operações de domínio de validação afetaram somente a memória da API de teste em :18080.

Todos os arquivos escritos pertencem a `web/reference-version/`. A versão anterior e o mapa foram preservados. Nenhum endpoint foi criado. Nenhuma integração adicional está autorizada por esta entrega.

**PARAR PARA HUMAN REVIEW — NÃO COMMIT.**
