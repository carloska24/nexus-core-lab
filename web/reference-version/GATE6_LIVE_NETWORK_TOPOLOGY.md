# Gate 6 — Live Network Topology

Data: 14/09/2026. Status: implementado e validado, aguardando Human Review.

Pasta: `C:\Users\joaob\OneDrive\Documentos\nexus-core-lab\web\reference-version`.

## 1. Arquivos desta entrega

- Novos: `src/network-catalog.ts`, `src/topology.tsx`, `src/LiveTopology.tsx`, `src/topology.css`, `tests/gate6.cjs`.
- Modificados: `src/App.tsx`, `src/session-api.ts`, `src/SessionsPage.tsx` e `README.md`.
- Relatório: este documento. Evidências: `evidence/gate6/`.
- A alteração em SessionsPage apenas amplia o tipo do estado de seleção para string; não muda o comportamento do Gate 5. session-api reexporta o catálogo centralizado.

## 2. Catálogo confirmado

Espelho estático de `internal/network/cell.go`, centralizado em `src/network-catalog.ts`:

| ID | Nome | Tecnologia |
| --- | --- | --- |
| CELL-SP-001 | Campinas Centro | LTE |
| CELL-SP-002 | Campinas Barão Geraldo | 5G |
| CELL-SP-003 | Campinas Cambuí | 5G |

Não é descoberta por API nem indica torres físicas ou disponibilidade. Coordenadas SVG são ilustrativas, explicitamente identificadas como não GPS. Fundo de Campinas, antenas, legenda, controles de zoom e composição existentes foram preservados. Google Maps permanece pausado.

## 3. Agregação

TopologyProvider usa a coleção real do DevicesProvider e consulta a sessão ativa de cada UUID. Subscribers fornece enriquecimento de identificação. Cada rodada captura a coleção, ordena por UUID e publica o resultado após concluir as consultas. Não foi criado endpoint global de sessões.

## 4. Concorrência

Máximo de quatro consultas de sessão simultâneas, sem dependência adicional. Teste com respostas atrasadas confirmou o limite e uma consulta por Device em cada rodada.

## 5. Polling

Próxima rodada agendada cinco segundos após finalizar a anterior, sem sobreposição. Ativo somente em Overview/Network; sair dessas telas cancela requisições e timers. Ao retornar, dados anteriores são marcados stale até revalidação. A coleção de Devices é a fornecida pelo provider existente.

## 6. Tratamento de 404

Somente HTTP 404 com código `ACTIVE_SESSION_NOT_FOUND` significa ausência de sessão. Outros 404 e falhas são erros. O teste injetou `ROUTE_NOT_FOUND` e confirmou que não vira desconexão. Respostas com identidade incompatível ou estado diferente de CONNECTED também não são aceitas como sessão ativa.

## 7. Estados e falhas

Loading aguarda a primeira observação; success representa rodada completa; partial combina sucesso e falhas; stale preserva observações anteriores identificadas como antigas; error representa falha sem dados confirmados. Links antigos ficam cinza e pontilhados. Falhas não fabricam estado disconnected. A recuperação ocorre na próxima rodada bem-sucedida.

## 8. Devices sem sessão

Permanecem no registro expansível da topologia, com identidade real e estado sem sessão. Não recebem conexão verde. Devices desconhecidos por falha são identificados separadamente.

## 9. Devices conectados

Links verdes representam sessões confirmadas para a célula informada pelo backend. Marcadores usam UUID como identidade e aliases visuais determinísticos. Clique ou teclado abre detalhes de Device, Subscriber e Session. Não há limite de cinco Devices no mapa; acima de quinze por célula os aliases visíveis são omitidos para reduzir sobreposição, mantendo marcadores e detalhes.

## 10. Attach

Fluxo real validado: criar e ativar Subscriber, registrar Device, confirmar ausência de sessão e executar Attach pela tela Sessions em CELL-SP-001. O mapa observou a conexão por polling, sem atualização otimista.

## 11. Handover

Executado pela tela Sessions para CELL-SP-002. O mapa passou a mostrar a associação nova. Não há trajetória GPS nem seta laranja permanente fabricada; a legenda visual foi preservada.

## 12. Identidade e IP preservados

Attach e Handover retornaram a mesma sessão `f80031d1-e6a5-4dfc-b231-fcf2a7deaf8d` e IP `10.45.0.2`. Evidência completa em `evidence/gate6/results.json`.

## 13. Detach

Executado pela tela Sessions. A sessão tornou-se DISCONNECTED, o link foi removido após observação e o Device permaneceu no registro.

## 14. Network Cells KPI

Mostra três células configuradas, a partir do catálogo. Não afirma saúde online. Active Sessions KPI continua usando Telemetry; não foi substituído pela contagem agregada do mapa.

## 15. Preview Active Sessions

Compartilha exatamente o snapshot da topologia, sem consultas adicionais. Mostra até cinco sessões confirmadas com total observado e indicação de estado parcial/stale/error. Links de gerenciamento levam à tela Sessions existente.

## 16. Validação com 1, 5 e 10 Devices

PASS: um Device sem sessão e fluxo completo; cinco conectados; dez conectados na mesma célula com posições distintas. Preview mostra cinco de dez sem limitar o mapa. Compatibilidade 5G Device com célula LTE foi preservada conforme comportamento do backend; nenhuma regra nova de elegibilidade foi introduzida.

## 17. Requisições observadas

Registro do navegador do mapa: 264 requisições, das quais 80 para sessões, todas no contrato existente `GET /api/v1/sessions?device_id={UUID}`. Esses números abrangem cenários de sucesso, falha e recuperação do teste, não representam taxa de produção. Providers existentes continuam consultando Devices, Subscribers, Health e Telemetry. Operações Attach/Handover/Detach ocorreram pela tela Sessions. Chamadas de preparação de fixtures não estão incluídas nesse registro do navegador.

## 18. Frontend build

`npm run build`: PASS, 1609 módulos, Vite 6.4.3. Build final concluído após ajuste de altura das linhas do preview para preservar o rodapé. Nenhuma dependência adicionada.

## 19. Go vet

`go vet ./...`: PASS.

## 20. Go build

`go build ./...`: PASS.

## 21. Go test e teste de navegador

`go test -count=1 ./...`: PASS. `TEST_DATABASE_URL` indisponível; testes PostgreSQL reais não executados nesta entrega. Race detector não foi repetido neste Gate; o micro-gate anterior registra sua validação.

`tests/gate6.cjs`: PASS, 13 verificações e zero erros JavaScript não tratados. Inclui concorrência, rodada lenta sem sobreposição, intervalo após conclusão, cancelamento ao sair, falhas parciais/totais e recuperação. Ambiente isolado em memória: frontend `http://127.0.0.1:5192`, backend `http://127.0.0.1:18092`; não usa o banco do usuário.

## 22. Screenshots reais

Em [evidence/gate6](evidence/gate6/), quatro estados em cada resolução 1920x1080, 1440x900 e 1366x768:

- `no-sessions-{resolução}.png`
- `connected-{resolução}.png`
- `handover-{resolução}.png`
- `detached-{resolução}.png`

Também: `five-devices.png`, `ten-devices.png`, `partial.png` e `results.json`. Inspeção visual confirmou a composição preservada e rodapé do preview visível com cinco linhas. As contagens reais naturalmente diferem da referência mock.

## 23. git diff --stat

Estado observado no repositório, incluindo mudanças anteriores a este Gate:

```text
 internal/device/handler.go                  | 14 +++++++---
 internal/device/handler_test.go             |  4 +--
 internal/device/memory_repository.go        | 18 +++++++++++++
 internal/device/postgres_repository.go      | 25 ++++++++++++++++++
 internal/device/postgres_repository_test.go | 40 +++++++++++++++++++++++++++++
 internal/device/repository.go               |  1 +
 internal/device/service.go                  |  5 ++++
 7 files changed, 101 insertions(+), 6 deletions(-)
```

Essas alterações Go são preexistentes. `web/` está untracked, portanto o diff convencional não enumera as alterações do frontend.

## 24. git status --short

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

Sem commit e sem push.

## 25. Preservação do backend

Comparação de hashes de 57 arquivos de backend com o baseline anterior ao Gate: zero alterações. Nenhum endpoint, schema, migration ou regra de Subscriber/Session/Device foi alterado. Interface original da pasta pai preservada. A organização documental solicitada separadamente arquivou relatórios antigos em `docs/archive/` e reuniu o relatório do micro-gate nesta pasta.

## 26. Limites e Human Review

Recent Events e IP Pool Usage continuam sem integração autoritativa, identificados como mock. Não foi criado feed por diferenças de snapshots nem cálculo de IPPool a partir de sessões. Simulator e regras de compatibilidade/Subscriber suspenso ficaram fora do escopo. Nenhuma integração Google Maps, autenticação, websocket ou SSE adicionada.

Gate 6 entregue para Human Review. Nenhum Gate posterior iniciado.
