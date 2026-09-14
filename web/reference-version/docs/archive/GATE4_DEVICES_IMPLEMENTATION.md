# Gate 4 — Devices: implementação para revisão humana

Data: 13/09/2026. **Frontend implementado e validado; revisão humana pendente.**

Google Maps/API de mapas permanece pausado, conforme solicitado. O contrato global de Devices entregue no Gate 4A permitiu retomar este gate. Este relatório substitui o bloqueio anterior por ausência de listagem global.

**Ressalva de regressão:** o build e os testes de navegador passaram. `go test ./...` falhou no teste preexistente `TestGlobalDeviceHTTPContract`, devido à suposição de ordem quando dois registros têm o mesmo timestamp. Não considerar a suíte Go aprovada. Nenhum arquivo Go foi alterado neste gate.

## 1. Arquivos criados/modificados

Dentro de `web/reference-version/`:

- Novos: `src/device-api.ts`, `src/devices.tsx`, `src/DevicesPage.tsx`, `src/devices.css`, `tests/gate4.cjs`.
- Modificados: `src/App.tsx` (provider, rota Devices e KPI), `vite.config.ts` (proxy Devices), este relatório.
- Evidências: `evidence/gate4/results.json` e 12 screenshots.
- Build gerado em `dist/`.

Nenhuma dependência foi adicionada. Nenhuma mudança na versão original da interface.

## 2. Contrato Device

Campos exatos: `id`, `subscriber_id`, `imei` (string), `technology` (`LTE` ou `5G`), `status` (`REGISTERED` ou `INACTIVE`), `created_at`, `updated_at`. O cliente valida a forma do payload e datas. Não cria Cell, IP, Session, lifecycle ou delete de Device.

## 3. Funções HTTP

`getDevices()` usa GET `/api/v1/devices`; `getDevice(id)` usa GET `/api/v1/devices/{id}`; `getDevicesBySubscriber(id)` usa GET `/api/v1/devices?subscriber_id={id}`; `registerDevice(body)` usa POST `/api/v1/devices`, com `subscriber_id`, `imei`, `technology`.

Reutiliza `read`/`ApiError` e fetch nativo. Leituras canceláveis; timeout de 10 segundos. Apenas uma coleção `null` com HTTP bem-sucedido é normalizada para `[]`. Falhas e payload inválido não viram coleção vazia.

## 4. Coleção compartilhada

`DevicesProvider` mantém a coleção global consumida pela página e por Total Devices. Carga inicial, refresh manual e revalidação após POST confirmado. Controle de geração/AbortController impede que uma resposta antiga sobrescreva a coleção. Não há polling periódico de Devices.

## 5. Devices Page

Tabela real com IMEI, tecnologia, status, identidade do Subscriber, criação, atualização e botão de detalhes. A identidade visual reutiliza os padrões de Subscribers, com CSS adicional restrito a Devices. Estados vazios, indisponibilidade e snapshot antigo são distintos.

## 6. Subscriber selector

Reutiliza a coleção do `SubscribersProvider`; não faz GET individual por linha. Apenas ACTIVE pode ser selecionado. PENDING_ACTIVATION, SUSPENDED e DEACTIVATED aparecem desabilitados. Coleção desatualizada/indisponível bloqueia o cadastro até refresh. Sem ACTIVE, mostra `No active subscribers available` e link para criar/ativar Subscriber.

## 7. Register Device

Formulário com Subscriber, IMEI e LTE/5G. Validação básica de 15 dígitos. Aguarda resposta real e usa o Device devolvido; não cria registro otimista. Após confirmação, atualiza por UUID e revalida a coleção global. Durante a operação, bloqueia novo envio e fechamento. Se a situação do Subscriber mudou em outro cliente, o backend continua sendo a autoridade.

## 8. Details

Abertura por UUID consulta GET individual. Mostra todos os campos do contrato, timestamps completos e formatados, além de IMSI/MSISDN quando disponíveis na coleção compartilhada. Carregamento, erro e tentativa novamente explícitos.

## 9. Busca/filtros

Busca local por IMEI, technology, status, subscriber_id, IMSI e MSISDN. O seletor de Subscriber executa a consulta HTTP filtrada real. Essa resposta tem estado próprio e nunca substitui a coleção global. Ao trocar o filtro, cancela a leitura anterior. O total global continua igual ao KPI, enquanto a quantidade exibida reflete o filtro.

## 10. Erros

Preserva HTTP, código e mensagem do backend. Verificados: 400 por IMEI inválido, 404 por Subscriber inexistente, 409 por IMEI duplicado e 422 para Subscriber não ACTIVE. O navegador exibiu 409 e 422 reais. A validação UX também rejeitou IMEI curto. Falha de rede mantém erro explícito; não mostra sucesso fictício.

## 11. Subscriber 1:N Device

Um mesmo Subscriber recebeu LTE e 5G via interface. Outro recebeu um terceiro Device. O filtro do primeiro retornou dois registros, enquanto o total global permaneceu três na execução final.

## 12. Loading/error/stale

Sem snapshot: loading/erro e KPI `—`. Com snapshot e falha: dados anteriores marcados stale. Coleção vazia bem-sucedida: zero real. Refresh recupera o estado success. Falha offline foi injetada no transporte do navegador para permitir teste determinístico, mantendo a API isolada disponível aos demais checks.

## 13. Testes

`tests/gate4.cjs` executado com Chrome headless/Playwright, frontend 5187 e API Go 18087 em memória, `DATABASE_URL` vazio. O arquivo de resultados registra as verificações e todas as requisições do navegador.

Cobertura: vazio/KPI zero, ausência de ACTIVE, cadastro LTE/5G, relação 1:N, detalhes GET, buscas, filtro HTTP por Subscriber, total global, IMEI inválido/duplicado, Subscriber inexistente, três estados não ACTIVE, suspensão concorrente, stale, erro sem snapshot, recovery, ausência de polling de Devices e zero chamadas Sessions. Nenhum erro JavaScript não capturado.

Os dados de teste são sintéticos enviados à API Go real isolada, não fixtures inseridas na tela e não dados do banco do usuário. As validações HTTP 400/404/422 também incluem chamadas diretas de teste; não estão no log de requests do navegador.

## 14. Evidência Subscriber → Activate → Device

A execução final começou vazia. Pelo navegador: provisionou Subscriber, ativou, registrou LTE, registrou 5G para o mesmo UUID. Criou outro Subscriber e registrou outro Device. O teste também criou/ativou e suspendeu/desativou Subscribers para verificar elegibilidade. Uma suspensão externa após seleção gerou 422 real no cadastro, sem aumentar o total.

## 15. Coerência Total Devices/lista

Zero inicial confirmado na página e Overview. Três Devices reais ao final. Filtro por proprietário com dois registros preservou o total três. Busca local não altera total. `103` e `+8%` removidos do KPI Devices.

## 16. Screenshot Overview

`evidence/gate4/overview-1920x1080.png`, `overview-1440x900.png`, `overview-1366x768.png`. Composição preservada; apenas o KPI Devices passa a usar a coleção real. Topologia/mapa, Sessions mock, Recent Events mock e IPPool não foram alterados.

## 17. Screenshots Devices

`evidence/gate4/devices-{resolução}.png`, `register-{resolução}.png`, `details-{resolução}.png`, nas mesmas três resoluções. Capturas reais do navegador. Conferência visual de tabela, formulário, detalhes e Overview: sem controles cortados nas capturas examinadas; modais preservam rolagem para alturas menores.

## 18. Build

`npm run build`: PASS, TypeScript + Vite 6.4.3, 1602 módulos. A primeira tentativa esbarrou em acesso de diretório do esbuild no sandbox; a execução autorizada completou normalmente. Nenhuma alteração de código foi necessária para esse erro de ambiente.

## 19. Go regression

`go test ./...` com `TEST_DATABASE_URL` vazio: **FAIL** em `internal/device/list_test.go:51`, teste `TestGlobalDeviceHTTPContract` preexistente do Gate 4A. Os dois Devices receberam exatamente `2026-09-13T22:32:01.9582837Z`; a listagem desempata por UUID, enquanto o teste exige ordem de criação. As demais saídas de pacotes passaram; validação PostgreSQL não foi executada neste gate.

Diagnóstico confirmado por leitura: `memory_repository.go` ordena por CreatedAt e depois ID, mas o teste compara os IDs pela ordem do loop. A correção proposta para outro passo é comparar a coleção e sua ordenação pelo contrato `(created_at, id)`, sem exigir ordem de inserção em empate. **Não foi feita alteração Go para manter o escopo deste gate.**

## 20. Git status --short

Estado do repositório, incluindo mudanças já existentes do Gate 4A:

```text
 M internal/device/handler.go
 M internal/device/handler_test.go
 M internal/device/memory_repository.go
 M internal/device/postgres_repository.go
 M internal/device/postgres_repository_test.go
 M internal/device/repository.go
 M internal/device/service.go
?? internal/device/list_test.go
?? web/
```

O diretório web já era não rastreado. Não houve commit nem push.

## 21. Preservação backend/DB

Comparação SHA-256 com o início desta retomada: **57 arquivos verificados em internal/, cmd/ e migrations/, zero alterados**. As alterações Go listadas acima são anteriores, do Gate 4A. Não foram alterados PostgreSQL, migrations ou compose; não foram criados endpoints. Testes somente em memória.

## 22. Zero requests Sessions

Asserção do teste passou: **zero requests `/api/v1/sessions`**. Health continua sendo consultado; a lista Devices não fez polling no intervalo de 11 segundos observado. Não houve integração de Sessions, Network, Events, IPPool ou Simulator neste gate.

## Executar e revisar

Na pasta `web/reference-version`, `npm run dev` inicia frontend e backend pelo runner já existente. Abra a URL informada pelo Vite e entre em Devices. Sem DATABASE_URL, os dados são temporários. Para reproduzir o teste com as portas padrão do teste, use API 18086 e frontend 5186; ou configure GATE4_API/GATE4_ORIGIN conforme a instância isolada.

**Parada para HUMAN REVIEW. Frontend testado, ressalva Go explicitamente pendente. Não avançar automaticamente para Sessions nem Google Maps.**
