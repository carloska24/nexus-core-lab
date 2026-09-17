# NEXUS CORE LAB — Gate 13A Public Demo Foundation

## Status

Gate 13A implementado e validado localmente. Nenhum commit ou push foi realizado. Gate 13B e Gate 13C não foram iniciados.

## Baseline

- Branch: `feat/demo-ui`
- HEAD inicial: `4b9dbbeacc5d676441f68d13f929d022d1252b84`
- Storage público aprovado para esta etapa: MEMORY
- Public Demo Mode: desabilitado por padrão
- `TEST_DATABASE_URL`: indisponível no ambiente de validação

O working tree não estava clean no início. Foram encontrados dois itens conhecidos e anteriores ao Gate 13A:

- `web/reference-version/src/comparison.css`: correção visual pendente da página Network;
- `web/reference-version/PUBLIC_INTERACTIVE_DEMO_ARCHITECTURE_REPORT.md`: relatório arquitetural solicitado antes deste Gate.

A correção visual não foi alterada como parte do Gate 13A. Durante o checkpoint final, o relatório arquitetural recebeu somente a correção editorial exigida pelo Human Review: a Free Instance do Koyeb não deve ser descrita como garantia de cadastro sem cartão.

## Arquitetura escolhida

O isolamento foi introduzido no composition root da API, sem adicionar conceitos de tenancy aos domínios Subscriber, Device ou Session.

Quando o modo público está ligado, um registry específico resolve um cookie anônimo e seleciona um contexto MEMORY completo. Cada contexto possui:

- Subscriber Memory Repository;
- Device Memory Repository;
- Session Memory Repository;
- Subscriber, Device e Session Services;
- IPPool próprio;
- Telemetry Worker próprio;
- Recent Events próprios;
- métricas e contador de requests próprios;
- limitadores de mutação e reset próprios.

Quando o modo público está desligado, o composition root anterior continua sendo utilizado integralmente. O caminho PostgreSQL não recebe visitor isolation.

## Public Demo Mode

Configuração principal:

```text
PUBLIC_DEMO_MODE=true
```

Default: `false`.

Configurações opcionais e pequenas:

| Variável | Default |
|---|---:|
| `PUBLIC_DEMO_IDLE_TTL` | `30m` |
| `PUBLIC_DEMO_ABSOLUTE_TTL` | `2h` |
| `PUBLIC_DEMO_MAX_CONTEXTS` | `100` |
| `PUBLIC_DEMO_MAX_SUBSCRIBERS` | `20` |
| `PUBLIC_DEMO_MAX_DEVICES` | `20` |
| `PUBLIC_DEMO_MAX_CONNECTED_SESSIONS` | `10` |
| `PUBLIC_DEMO_MUTATIONS_PER_MINUTE` | `30` |
| `PUBLIC_DEMO_RESETS_PER_MINUTE` | `3` |

Configurações inválidas falham no startup. O modo público não pode ser combinado com `DATABASE_URL`; essa combinação falha de forma explícita para impedir isolamento falso sobre PostgreSQL.

## Visitor cookie

Nome:

```text
nexus_demo_session
```

Propriedades:

- 16 bytes aleatórios gerados por `crypto/rand`;
- 128 bits de entropia;
- representação Base64 URL-safe sem padding;
- `HttpOnly`;
- `SameSite=Strict`;
- `Path=/`;
- `Secure` quando TLS está ativo ou o proxy informa `X-Forwarded-Proto: https`;
- utilizável sem `Secure` no HTTP local de desenvolvimento e testes.

O cookie contém somente um identificador opaco. Não armazena nome, e-mail, IP, fingerprint, GPS ou outro dado pessoal. Cookie ausente, inválido, expirado ou desconhecido resulta em um novo contexto e em um novo identificador.

## Context registry

O registry é protegido por mutex e mantém no máximo 100 contextos por default. Ele executa:

- resolução do cookie;
- criação criptograficamente segura do identificador;
- prevenção de colisão de token;
- atualização de `lastSeen`;
- expiração idle e absoluta;
- remoção lazy de contextos expirados;
- shutdown dos Telemetry Workers removidos;
- reset atômico do contexto atual;
- encerramento de todos os contextos no shutdown da API.

Nenhum scheduler, event bus, Redis ou framework genérico de tenancy foi criado.

## State ownership

| Estado | Public Demo Mode | Modo normal |
|---|---|---|
| Subscribers | Por visitante | MEMORY ou PostgreSQL existente |
| Devices | Por visitante | MEMORY ou PostgreSQL existente |
| Sessions | Por visitante | MEMORY ou PostgreSQL existente |
| Recent Events | Por visitante | Worker único existente |
| Telemetry event counters | Por visitante | Worker único existente |
| Telemetry request counter | Por visitante | Processo normal existente |
| IPPool | Por visitante | Pool normal existente |
| Health/liveness | Processo | Processo |
| Storage diagnostics | Resposta MEMORY sem estado de domínio | Diagnóstico existente |

## Telemetry semantics

Em Public Demo Mode:

- `active_sessions` e `connected_devices` são derivados somente do Session Repository do visitante;
- contadores de Attach, Handover, Detach e Stale Disconnect pertencem ao worker daquele visitante;
- `dropped_events_total` pertence ao worker daquele visitante;
- `requests_total` conta somente requests roteados ao contexto daquele visitante;
- `/api/v1/events/recent` nunca lê eventos de outro contexto;
- `/health` permanece liveness process-level e não cria contexto.

## IPPool semantics

Cada visitante recebe um `network.IPPool` novo e independente. Um Attach de A reduz somente o pool de A. Handover preserva o IP e Detach devolve o endereço somente ao pool do mesmo contexto. Reset substitui o pool atual por um pool vazio.

## Lifecycle e cleanup

Defaults:

- idle expiration: 30 minutos;
- absolute lifetime: 2 horas;
- capacidade: 100 contextos.

O cleanup é lazy e ocorre durante a resolução de requests. Para a escala aprovada de até 100 contextos, isso evita uma goroutine ou scheduler adicional. Ao atingir o limite sem contextos expirados, novos visitantes recebem `503 DEMO_CAPACITY_REACHED`; contextos ativos não são removidos silenciosamente.

Workers removidos são encerrados de forma graciosa. O shutdown da API encerra todos os workers remanescentes.

## Resource limits

Os limites pertencem à camada de composição da demo e não alteram invariantes dos domínios:

- 20 Subscribers por visitante;
- 20 Devices por visitante;
- 10 Sessions conectadas por visitante.

Mutações do mesmo visitante são serializadas enquanto o limite é verificado e a operação é executada. Isso impede que requests concorrentes ultrapassem o teto por check-then-act. Excesso retorna `429 DEMO_RESOURCE_LIMIT` com mensagem específica da demo.

## Rate limiting

Foi implementado rate limiting MEMORY por contexto, com janela fixa e estrutura limitada pelo próprio número máximo de contextos:

- 30 mutações por minuto por default;
- 3 resets por minuto por default;
- leituras do dashboard não consomem o limite de mutações;
- o limitador de reset é preservado quando o estado é substituído, impedindo bypass por resets sucessivos.

Não foram adicionados Redis, banco ou rate limiter distribuído. Uma única instância continua sendo premissa desta versão.

## Request safety

Somente no Public Demo Mode:

- limite de body: 64 KiB;
- `Content-Type: application/json` obrigatório nos endpoints com body JSON obrigatório;
- mutações cross-origin são rejeitadas;
- localhost/127.0.0.1 continuam compatíveis com o proxy Vite local;
- nenhum CORS permissivo foi adicionado;
- `Content-Security-Policy` considera o futuro OpenFreeMap sem implementar o mapa;
- `X-Content-Type-Options: nosniff`;
- `Referrer-Policy: no-referrer`;
- `X-Frame-Options: DENY`;
- `Permissions-Policy` bloqueia geolocation, camera e microphone.

## Reset

Endpoint disponível somente no Public Demo Mode:

```http
POST /api/v1/demo/reset
```

Semântica:

- idempotente;
- preserva o token do visitante;
- substitui atomicamente o contexto completo;
- limpa Subscribers, Devices, Sessions, Events, Telemetry e IPPool;
- não afeta outros visitantes;
- não existe no roteador normal;
- resets concorrentes não deixam workers órfãos.

## Concurrency

O registry, repositories MEMORY, IPPool, Telemetry Worker e limitadores são race-safe. Mutações de um contexto usam lock próprio; visitantes diferentes continuam independentes e podem operar em paralelo.

O ajuste final do reset verifica se o contexto ainda é o atual antes da troca. Isso evita que dois resets concorrentes substituam o mesmo contexto e deixem um worker intermediário sem shutdown.

## Hero Flow validado

O teste do Gate usa um único visitor context e executa o fluxo oficial completo:

```text
Provision Subscriber
→ Activate Subscriber
→ Register Device
→ Attach
→ Handover
→ Detach
```

Durante o mesmo teste, um segundo visitante permanece vazio e não observa Subscriber, Device, Session, Events, Telemetry ou alocação de IP do primeiro.

## Isolation tests

| Requisito | Evidência automatizada |
|---|---|
| A. Subscriber isolation | A cria; B lista zero |
| B. Device isolation | A registra; B lista zero |
| C. Session isolation | B não encontra sessão ativa de A |
| D. Recent Events isolation | eventos de A; feed de B vazio |
| E. Telemetry isolation | A possui contadores; B permanece zero |
| F. IPPool isolation | A aloca 1; B permanece com 0 |
| G. Reset isolation | reset A preserva dados de B |
| H. Idle expiration | relógio controlado expira contexto inativo |
| I. Absolute expiration | acessos intermediários não evitam expiração absoluta |
| J. Context cap | visitante excedente recebe 503 |
| K. Resource limits | Subscribers, Devices e Sessions cobertos |
| L. Rate limiting | mutações/reset limitados; leitura preservada |
| M. Cookie inválido/ausente | novo cookie válido de 128 bits |
| N. Public Demo OFF | default normal e reset ausente |
| O. MEMORY normal | suíte existente completa passou |
| P. PostgreSQL normal | suíte compilou; integração real não executada sem URL |
| Q. Concorrência no mesmo visitante | 10 provisions concorrentes consistentes |
| R. Concorrência entre visitantes | dois contextos concorrentes permanecem isolados |

Também foram testados Content-Type inválido, Origin externo, body excessivo e presença dos security headers.

## Regressions

- Public Demo Mode permanece OFF por default.
- MEMORY normal continua usando os repositories e worker anteriores.
- PostgreSQL continua usando o composition root anterior.
- Nenhuma migration foi criada.
- Nenhuma regra de Subscriber, Device ou Session foi modificada.
- Nenhum endpoint existente teve contrato alterado no modo normal.
- `/health` continua sendo somente liveness.

## Validações Go

Executadas após a implementação final:

```text
go fmt ./...                 PASS
go vet ./...                 PASS
go build ./...               PASS
go test -count=1 ./...       PASS
```

## Race detector

O Windows local não possui CGO habilitado. A suíte foi executada no container oficial Linux:

```text
golang:1.27-bookworm
go test -race -count=1 ./...
```

Resultado: **PASS — zero data races**.

## Frontend build

Executado em `web/reference-version`:

```text
npm run build
```

Resultado: **PASS**.

Nenhum código de frontend foi alterado pelo Gate 13A.

## PostgreSQL validation status

`TEST_DATABASE_URL` estava ausente. Os testes de integração PostgreSQL reais não foram executados e nenhum banco pessoal foi criado. A suíte PostgreSQL existente compilou e os testes que não dependem de uma instância externa passaram.

## Arquivos do Gate 13A

- `cmd/api/main.go`
- `cmd/api/public_demo.go`
- `cmd/api/public_demo_test.go`
- `web/reference-version/GATE13A_PUBLIC_DEMO_FOUNDATION.md`

## Itens anteriores, fora do Gate

- `web/reference-version/src/comparison.css`
- `web/reference-version/PUBLIC_INTERACTIVE_DEMO_ARCHITECTURE_REPORT.md` (preexistente; wording de pagamento corrigido no checkpoint)

## Known limitations

- estado da demo permanece local a um único processo;
- restart remove todos os contextos;
- cookie anônimo oferece isolamento de demonstração, não autenticação;
- rate limiting é por visitor context e pode ser contornado criando novos contextos, embora a capacidade global limite o crescimento total;
- atingir 100 contextos ativos retorna 503 em vez de expulsar um visitante ativo;
- cleanup depende da chegada de requests;
- não há persistência pública;
- não há rate limiter distribuído;
- mapa, Docker, deploy, Koyeb, domínio público, jornada guiada e README Live Demo permanecem fora deste Gate;
- o limite de Sessions conectadas rejeita novo Attach quando o teto já foi atingido, inclusive um possível re-attach no teto.

## Git

Nenhum commit ou push foi realizado.

Comandos finais executados:

```text
git diff --check        PASS
git diff --stat
git status --short
```

`git diff --stat` para arquivos já rastreados:

```text
cmd/api/main.go                          | 26 ++++++++++++++++++++++----
web/reference-version/src/comparison.css |  2 +-
2 files changed, 23 insertions(+), 5 deletions(-)
```

O stat nativo não inclui arquivos untracked. Os novos arquivos do Gate possuem:

```text
cmd/api/public_demo.go                                  515 linhas
cmd/api/public_demo_test.go                             491 linhas
web/reference-version/GATE13A_PUBLIC_DEMO_FOUNDATION.md 328 linhas antes desta atualização final
```

Status final observado:

```text
 M cmd/api/main.go
 M web/reference-version/src/comparison.css
?? cmd/api/public_demo.go
?? cmd/api/public_demo_test.go
?? web/reference-version/GATE13A_PUBLIC_DEMO_FOUNDATION.md
?? web/reference-version/PUBLIC_INTERACTIVE_DEMO_ARCHITECTURE_REPORT.md
```

Os avisos de conversão futura LF/CRLF emitidos pelo Git não representam erro de whitespace; `git diff --check` terminou com sucesso.

**PARE PARA HUMAN REVIEW. NÃO INICIAR GATE 13B.**
