# NEXUS CORE LAB — Gate 13B Real Map Experience

Data da validação: 17/09/2026

Estado: implementação concluída, aguardando Human Review

Commit e push: **não executados**

## 1. Baseline confirmado

Antes da implementação:

```text
branch: feat/demo-ui
HEAD: 4ba6c03454fc2680fa124ba297ea51b1927b4cd2
origin/feat/demo-ui: 4ba6c03454fc2680fa124ba297ea51b1927b4cd2
working tree: CLEAN
```

O Gate 13B partiu exatamente do checkpoint aprovado do Gate 13A.

## 2. Auditoria inicial

A auditoria confirmou:

- frontend canônico em React 18, TypeScript e Vite;
- topologia local anterior renderizada como SVG sobre a imagem estática de Campinas;
- `TopologyProvider` e `useTopology` já consumindo o snapshot real de Devices e Sessions;
- `TopologyOverlay` já derivando associação de uma Session observada, sem estado geográfico persistente;
- Attach, Handover e Detach operando pelos endpoints reais existentes;
- Active Sessions e Recent Events já alimentados pelo backend;
- CSP e Permissions-Policy do Gate 13A já preparadas para `*.openfreemap.org` e com geolocalização bloqueada;
- corte vertical na página Network provocado pela altura fixa de `680px`;
- nenhuma biblioteca de mapas instalada no baseline.

O dashboard aprovado foi preservado. Não houve redesign global.

## 3. Solução implementada

O painel **Network Topology** agora usa cartografia real de Campinas fornecida pelo OpenFreeMap e renderizada diretamente pelo MapLibre GL JS.

A distinção exibida na própria interface é:

> Simulated telecom topology over real Campinas cartography

Portanto:

- ruas, bairros e geografia são cartografia real;
- Session, IP, Attach, Handover, Detach e Events vêm do estado HTTP/domínio NEXUS;
- coordenadas de Cells e Devices, cobertura e vínculo visual são apenas apresentação educacional;
- não há GPS, geolocalização do navegador, posição de antena real ou cálculo de RF.

## 4. Tecnologia e dependências

Dependência adicionada:

```text
maplibre-gl 6.10.0
```

A versão foi confirmada na documentação oficial do MapLibre antes da instalação. Não foi usado wrapper React porque a API direta atende ao ciclo de vida necessário com menos abstração.

Não foram adicionados Leaflet, Google Maps, Mapbox SDK, OpenLayers ou bibliotecas duplicadas.

Referências oficiais:

- [MapLibre GL JS — fullscreen map example](https://maplibre.org/maplibre-gl-js/docs/examples/view-a-fullscreen-map/)
- [OpenFreeMap — quick start](https://openfreemap.org/quick_start/)

O `package-lock.json` fixa a árvore instalada. `npm ci` concluiu com 97 pacotes e zero vulnerabilidades conhecidas pelo audit executado.

## 5. Provider e configuração

Toda a configuração variável do provider está isolada em `src/map-provider.ts`:

```text
provider: OpenFreeMap
style: https://tiles.openfreemap.org/styles/liberty
center: longitude -47.062, latitude -22.868
zoom: 11.45
min zoom: 10
max zoom: 16
```

Não existe API key, token ou credencial. O frontend não usa `tile.openstreetmap.org` como provider principal.

A atribuição do provider e dos dados permanece visível no canto inferior do mapa. A atribuição adicional informa que as posições telecom são simuladas.

O MapLibre 6 resolve seu worker como módulo relativo ao pacote. `maplibre-gl` foi excluído somente do dependency pre-bundle do Vite para preservar essa resolução durante desenvolvimento; a produção continua sendo empacotada normalmente.

## 6. Campinas e coordenadas ilustrativas

As três Cells configuradas permanecem:

| Cell | Região apresentada | Tecnologia | Coordenada ilustrativa |
|---|---|---:|---:|
| CELL-SP-001 | Centro | LTE | -47.0608, -22.9056 |
| CELL-SP-002 | Barão Geraldo | 5G | -47.0714, -22.8238 |
| CELL-SP-003 | Cambuí | 5G | -47.0508, -22.8970 |

As coordenadas representam aproximadamente as regiões nomeadas. Elas não foram obtidas de cadastros de antenas e não representam infraestrutura telecom real.

O catálogo conserva também as coordenadas `x/y` usadas pelo fallback local. Nenhuma coordenada foi persistida no backend ou adicionada ao domínio.

## 7. Visualização de Cells e cobertura

Cada Cell é representada por:

- ring com cor correspondente a LTE ou 5G;
- núcleo visível;
- Cell ID;
- nome da região;
- tecnologia;
- popup inspecionável com estado e aviso sobre coordenada ilustrativa;
- cobertura circular discreta, explicitamente tratada como apresentação ilustrativa.

Não existem RSRP, RSRQ, SINR, azimuth, banda, setor ou propagação simulada com falsa precisão.

## 8. Devices e Sessions

`buildMapPresentation` transforma somente Sessions `CONNECTED` observadas no snapshot existente.

Para cada associação válida:

- o Device aparece próximo da Cell corrente;
- a posição é derivada por hash determinístico de Session/Device/Cell;
- a mesma associação permanece visualmente estável entre renders;
- o vínculo Device → Cell é uma linha verde;
- Device, IMEI, Session ID, IP e Cell podem ser inspecionados;
- uma Session desconectada não cria elemento dinâmico.

Nenhum Device é inventado. Não existe randomização por render e não foi criado polling adicional.

## 9. Attach, Handover e Detach

### Attach

Após confirmação real do backend:

- a Session `CONNECTED` aparece no mapa;
- o Device aparece ligado à Cell escolhida;
- o IP alocado pelo NEXUS fica disponível para inspeção;
- o ATTACH continua aparecendo no Recent Events.

### Handover

Após confirmação real do backend:

- o vínculo passa para a nova Cell;
- Session ID permanece igual;
- IP permanece igual;
- não é criada trajetória física entre bairros;
- CELL_HANDOVER permanece nos Events.

### Detach

Após confirmação real do backend:

- a associação dinâmica desaparece;
- o Device conectado desaparece do mapa;
- as três Cells permanecem;
- o IP retorna ao pool;
- DETACH permanece nos Events.

## 10. Fallback local

O SVG/mapa local anterior foi preservado e evoluído como fallback. Ele é selecionado em caso de:

- WebGL indisponível;
- falha de inicialização do MapLibre;
- falha relevante no style/provider.

O fallback mantém:

- as três Cells;
- o snapshot real de associação;
- estado de Attach/Handover/Detach quando disponível;
- controles locais de zoom;
- texto explicando a indisponibilidade;
- botão para tentar carregar novamente o mapa real;
- atribuição própria do recurso estático.

A falha externa não deixa a página vazia nem derruba o dashboard.

## 11. CSP e segurança

O Gate 13B auditou a política introduzida no Gate 13A. Nenhuma ampliação foi necessária:

```text
connect-src 'self' https://*.openfreemap.org
img-src 'self' data: blob: https://*.openfreemap.org
style-src 'self' 'unsafe-inline'
script-src 'self'
font-src 'self' data:
worker-src 'self' blob:
```

Observações:

- não foi introduzido wildcard `*`;
- não foi introduzido `unsafe-eval`;
- `unsafe-inline` em `style-src` já existia e atende aos estilos aplicados pelo MapLibre;
- Origin validation, cookie, SameSite e demais headers não foram modificados;
- `Permissions-Policy` continua com `geolocation=(), camera=(), microphone=()`;
- o código e os testes confirmam que nenhuma API de geolocalização é solicitada;
- popups são montados com DOM e `textContent`, sem injeção de HTML do domínio.

Os testes Go existentes dos security headers continuam passando.

## 12. Correção de clipping da página Network

A altura fixa de `680px` foi removida do workspace de Network. O ajuste é restrito à página:

- workspace Network usa coluna flexível;
- header conserva sua altura natural;
- área do mapa ocupa o espaço restante;
- `min-height: 0` permite o dimensionamento correto dos filhos;
- overflow indevido foi eliminado.

Não houve alteração global do grid do dashboard.

Medições automatizadas confirmaram `scrollWidth == clientWidth` e `scrollHeight == clientHeight` no workspace Network nas três resoluções aprovadas.

## 13. Responsividade validada

| Viewport | Resultado |
|---|---|
| 1920x1080 | PASS — página utilizável, mapa completo, sem clipping do workspace |
| 1440x900 | PASS — página utilizável, mapa completo, sem clipping do workspace |
| 1366x768 | PASS — página utilizável, mapa completo, sem clipping do workspace |

O produto continua com a composição desktop NOC aprovada. Um redesign mobile completo não fazia parte deste Gate.

## 14. Acessibilidade

- Cell ID, região e tecnologia são textos legíveis, não dependem somente de cor;
- o estado conectado possui representação textual paralela para leitores de tela;
- Device, Session, IP e Cell permanecem disponíveis no registro da topologia;
- mensagens de loading e fallback usam status textual;
- botões de zoom e retry possuem nomes acessíveis;
- o mapa é complementar e não é a única fonte do estado operacional.

## 15. Ciclo de vida e desempenho

- uma instância MapLibre é criada por montagem/tentativa do painel;
- polling não recria o mapa;
- atualizações usam `GeoJSONSource.setData`;
- o snapshot existente alimenta o mapa, sem polling novo;
- não foram adicionados WebSocket ou SSE;
- `ResizeObserver` chama `resize` quando necessário;
- fetch, observer e MapLibre são encerrados no unmount;
- uma nova instância só é criada após retry explícito ou nova montagem da rota.

## 16. Arquivos modificados e adicionados

### Modificados

- `index.html` — favicon local para retirar o 404 incidental do navegador;
- `package.json` — MapLibre e script de teste do mapa;
- `package-lock.json` — lock da dependência;
- `vite.config.ts` — resolução correta do worker MapLibre no dev server;
- `src/App.tsx` — integração do novo painel, preservando a composição existente;
- `src/comparison.css` — correção restrita do clipping de Network;
- `src/network-catalog.ts` — regiões e coordenadas ilustrativas.

### Adicionados

- `src/map-provider.ts` — configuração isolada do provider;
- `src/map-presentation.ts` — projeção determinística do domínio para GeoJSON;
- `src/MapTopology.tsx` — mapa real, layers, popups e fallback;
- `src/map-topology.css` — apresentação do mapa e controles;
- `tests/map-presentation.test.mjs` — testes unitários da projeção;
- `tests/gate13b.cjs` — validação real de navegador e Hero Flow;
- `evidence/gate13b/results.json` — evidência estruturada do navegador;
- `GATE13B_REAL_MAP_EXPERIENCE.md` — este relatório.

Os PNGs de validação foram gerados localmente em `evidence/gate13b`, mas permanecem ignorados pelo Git conforme a política existente.

## 17. Testes de apresentação

Comando:

```text
npm run test:map
```

Resultado: **PASS — 6 testes, 0 falhas**.

Cobertura comportamental:

1. Cells possuem coordenadas explícitas de apresentação;
2. Session `CONNECTED` cria Device e vínculo;
3. Attach, Handover e Detach seguem o snapshot autoritativo;
4. Session ID e IP são preservados no Handover;
5. coordenadas derivadas são determinísticas;
6. seleção de fallback cobre falhas relevantes;
7. implementação não usa geolocation.

## 18. Hero Flow em navegador real

O teste foi executado no Chrome em modo visível, usando Public Demo Mode local.

Resultado: **PASS**.

Fluxo validado:

```text
Provision Subscriber
→ Activate Subscriber
→ Register Device
→ Attach em CELL-SP-001
→ Handover para CELL-SP-002
→ Detach
```

Evidência da execução:

- visitante iniciou sem Device conectado;
- mapa de Campinas e as três Cells carregaram;
- Attach apresentou Session, Cell e IP autoritativos;
- Handover preservou o mesmo Session ID e o mesmo IP;
- vínculo visual mudou para CELL-SP-002;
- CELL_HANDOVER apareceu em Events;
- Detach removeu a associação;
- IP Pool voltou a zero alocações;
- DETACH apareceu em Events;
- segundo visitante permaneceu isolado.

Arquivos principais de evidência local:

- `evidence/gate13b/hero-attach-cell-001.png`;
- `evidence/gate13b/hero-handover-cell-002.png`;
- `evidence/gate13b/provider-fallback.png`;
- `evidence/gate13b/network-empty-1920x1080.png`;
- `evidence/gate13b/network-empty-1440x900.png`;
- `evidence/gate13b/network-empty-1366x768.png`;
- `evidence/gate13b/results.json`.

## 19. Fallback validado

O teste interceptou somente as chamadas a `https://tiles.openfreemap.org/**` e simulou indisponibilidade do provider.

Resultado: **PASS**.

- aplicação continuou renderizando;
- fallback local apareceu;
- três Cells permaneceram visíveis;
- camada de estado permaneceu disponível;
- indicação de simulação permaneceu explícita;
- dashboard não apresentou tela vazia.

## 20. Console do navegador

Resultado:

```text
uncaught JavaScript exceptions: 0
unexpected console errors: 0
```

Foram observadas duas respostas HTTP 404 esperadas do contrato existente de consulta de Session ativa quando ainda não existe sessão ou após o Detach. Elas aparecem no DevTools como `Failed to load resource`, foram classificadas separadamente na evidência e não são exceções JavaScript.

## 21. Validações Go

```text
go fmt ./...                 PASS
go vet ./...                 PASS
go build ./...               PASS
go test -count=1 ./...       PASS
```

`go fmt` não criou alteração em arquivo Go. O Gate 13B não modifica backend.

## 22. Race detector

A execução nativa do Windows informou que `-race` exige CGO. Foi aplicado o método isolado já utilizado pelo projeto:

```text
image: golang:1.27-bookworm
network: none
CGO_ENABLED: 1
GOTOOLCHAIN: local
GOPROXY: off
source mount: read-only
module cache mount: read-only
go test -race -count=1 ./...
```

Resultado: **PASS — zero data races**.

## 23. Frontend canônico

```text
npm ci                  PASS
npm run test:map        PASS — 6/6
npm run build           PASS
browser Gate 13B        PASS — 8/8 cenários
```

Build de produção:

```text
CSS: 119.51 kB · gzip 20.10 kB
JS: 1,269.73 kB · gzip 354.62 kB
```

## 24. Limitações conhecidas

- OpenFreeMap exige internet e WebGL para a experiência real; o fallback local cobre indisponibilidade.
- Tiles não são armazenados offline pelo projeto.
- posições e cobertura telecom são ilustrações e não devem ser interpretadas como GPS, antenas ou engenharia de RF.
- o bundle JavaScript ultrapassa o limiar de aviso de 500 kB do Vite após incluir MapLibre; lazy loading/code splitting pode ser avaliado em Gate futuro.
- a composição continua otimizada para desktop NOC; não houve redesign mobile.
- respostas 404 da consulta por Session ativa fazem parte do contrato existente para ausência de sessão e continuam visíveis no console de rede.

## 25. Estado do Git para Human Review

`git diff --check`: **PASS**, sem whitespace errors. Os avisos de conversão LF/CRLF são da configuração local do Git no Windows e não representam erro de diff.

O diff rastreado antes da inclusão deste relatório mostrava:

```text
7 files changed, 248 insertions(+), 47 deletions(-)
```

Há também arquivos novos do mapa, testes, evidência JSON e este relatório, listados na seção 16 e pelo `git status --short` final da entrega.

```text
 M web/reference-version/index.html
 M web/reference-version/package-lock.json
 M web/reference-version/package.json
 M web/reference-version/src/App.tsx
 M web/reference-version/src/comparison.css
 M web/reference-version/src/network-catalog.ts
 M web/reference-version/vite.config.ts
?? web/reference-version/GATE13B_REAL_MAP_EXPERIENCE.md
?? web/reference-version/evidence/gate13b/
?? web/reference-version/src/MapTopology.tsx
?? web/reference-version/src/map-presentation.ts
?? web/reference-version/src/map-provider.ts
?? web/reference-version/src/map-topology.css
?? web/reference-version/tests/gate13b.cjs
?? web/reference-version/tests/map-presentation.test.mjs
```

## 26. Confirmações de escopo

- nenhuma alteração funcional em Go;
- nenhuma migration;
- nenhuma mudança de schema;
- nenhuma API nova;
- nenhum GPS ou browser geolocation;
- nenhuma coordenada de antena real;
- nenhuma credencial ou token;
- nenhum Dockerfile ou deploy;
- nenhum commit;
- nenhum push;
- Gate 13C **não iniciado**.

**PARE PARA HUMAN REVIEW.**
