# Gate 5 — Sessions por Device

Implementado em 13/09/2026, após aprovação do escopo por Device. Google Maps permanece pausado.

## Entrega

Nova página Sessions integrada ao backend existente: selecionar Device real, consultar conexão ativa, Attach, Handover e Detach. Dados exibidos: UUID de sessão, Device e Subscriber; célula; IP; status; attached_at, updated_at e, quando presentes, closed_at e disconnect_reason.

Arquivos novos: `src/session-api.ts`, `src/SessionsPage.tsx`, `src/sessions.css`, `tests/gate5.cjs`, `GATE5_SESSIONS_PLAN.md` e este relatório. Alterações em `src/App.tsx` (rota) e `vite.config.ts` (proxy Sessions). Evidências em `evidence/gate5/`.

## Contrato e limites

- GET `/api/v1/sessions?device_id={uuid}` retorna uma sessão ativa, não coleção.
- Somente HTTP 404 com código ACTIVE_SESSION_NOT_FOUND representa ausência de sessão. Outros 404 continuam sendo erros.
- POST `/api/v1/sessions/attach`: device_id/cell_id.
- POST `/api/v1/sessions/{uuid}/handover`: target_cell_id.
- POST `/api/v1/sessions/{uuid}/detach`: corpo vazio JSON.
- Fetch nativo e validação de payload, timeout 10 segundos, cancelamento de leituras ao trocar Device/sair da tela.
- Não existe GET global de Sessions. A página não agrega consultas por todos os Devices e não inventa histórico global.
- Devices/Subscribers reutilizados dos providers existentes; refresh manual. Sessões sem polling.
- O KPI Active Sessions já usa telemetria dos gates anteriores. A tabela de Sessions do Overview continua explicitamente mock; não foi apresentada como uma lista global real.
- Catálogo local de células espelha `internal/network/cell.go`: CELL-SP-001 Centro/LTE, CELL-SP-002 Barão Geraldo/5G, CELL-SP-003 Cambuí/5G. Não existe endpoint de catálogo. Este espelhamento exige atualização manual se o contrato mudar.

## Comportamento

Ações aguardam a resposta do servidor, sem sucesso otimista. Falha de escrita bloqueia novas ações até refresh, porque uma resposta perdida não prova que a operação falhou no servidor. Snapshot anterior permanece marcado stale. Trocar Device limpa o snapshot da seleção anterior.

Attach disponível sem conexão ativa confirmada; Handover/Detach disponíveis com CONNECTED. Handover para a célula atual fica desabilitado. Após Detach, exibe a resposta DISCONNECTED e motivo. Refresh consulta novamente a conexão ativa, podendo mostrar ausência. Não adicionada ação de re-attach para substituir uma conexão ativa.

## Evidência real

Teste `tests/gate5.cjs`, Chrome headless, API isolada Go em memória 18088 e Vite 5188. Preparação de Subscriber/Devices via API local de teste; operações de sessão pelo navegador.

Passaram:

- Ausência inicial confirmada por ACTIVE_SESSION_NOT_FOUND.
- Attach real: CONNECTED, IP alocado pelo servidor.
- Handover real: mesma sessão/mesmo IP, célula alterada.
- Detach real: DISCONNECTED, closed_at e VOLUNTARY_DETACH.
- Troca de Device sem herdar detalhes anteriores.
- Falha de transporte injetada: stale preserva snapshot; erro inicial não vira ausência; refresh recupera.
- HTTP 404 com outro código permanece erro.
- HTTP 500 injetado em escrita: sem CONNECTED fictício, exige refresh.
- Nenhum polling Sessions durante 11 segundos; telemetria permanece funcionando.
- Nenhuma consulta global Sessions, zero erros JavaScript não capturados.

Resultados/requisições: `evidence/gate5/results.json`. Capturas reais e inspecionadas: `connected-1920x1080.png`, `connected-1440x900.png`, `connected-1366x768.png`; adicional `disconnected-1366x768.png`. Controles de operação dentro da viewport nas três resoluções.

## Limitação de elegibilidade encontrada

Teste real mostrou que um Device REGISTERED consegue Attach mesmo quando seu Subscriber está SUSPENDED. Leitura de `cmd/api/main.go:53` confirma: `deviceCheckerAdapter.CheckDeviceAttachable` consulta apenas Device e seu status REGISTERED, sem revalidar status de Subscriber. A interface respeita o contrato atual; não foi criado um bloqueio de segurança apenas no frontend. Se a regra desejada for exigir Subscriber ACTIVE também para Attach, isso requer mudança de contrato/backend em passo separado.

O backend também não impõe compatibilidade de tecnologia Device/célula no fluxo examinado. O seletor apresenta as células do catálogo, sem inventar validação adicional.

## Build e preservação

`npm run build`: PASS (TypeScript + Vite, 1605 módulos). Primeiro build limitado pelo sandbox do esbuild; execução autorizada passou.

Comparação SHA-256 dos 57 arquivos de internal/, cmd/ e migrations/ com início deste gate: zero alterações. Nenhuma dependência, mudança Go, migration, PostgreSQL, compose ou integração Maps. Todos os dados de teste em instância isolada de memória.

A falha anterior de `TestGlobalDeviceHTTPContract` do Gate4A por empate de timestamps continua uma pendência documentada; não foi corrigida nem omitida. Não foi repetida a suíte completa Go neste gate, pois nenhuma alteração backend foi feita. Testes específicos Session/Network registrados abaixo.

Estado git permanece com modificações preexistentes em internal/device/{handler.go,handler_test.go,memory_repository.go,postgres_repository.go,postgres_repository_test.go,repository.go,service.go}, novo internal/device/list_test.go do Gate4A e web/ não rastreado. Nenhum commit/push.

## Testar no seu ambiente

Recarregue a interface atual e abra Sessions. Selecione o Device cadastrado, escolha a célula e clique Attach; depois escolha outra célula para Handover ou use Detach. Caso Vite não tenha recarregado a configuração do proxy, reinicie `npm run dev` na pasta reference-version. O runner continua iniciando frontend e API juntos; sem DATABASE_URL, armazenamento temporário em memória.

Instância de teste desta entrega: http://127.0.0.1:5188/#sessions, com dados temporários separados dos seus.

Validação Go específica: `go test ./internal/session ./internal/network` com TEST_DATABASE_URL vazio: PASS nos dois pacotes (cache). Não valida PostgreSQL nem elimina a pendência da suíte completa do Gate4A.
