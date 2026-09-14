# NEXUS Core Lab — Reference Version

Esta é a interface em desenvolvimento, com integração ao backend Go do NEXUS.

**Pasta de trabalho:** `C:\Users\joaob\OneDrive\Documentos\nexus-core-lab\web\reference-version`

A interface original está na pasta pai `web/` e deve ser preservada. Os documentos desta versão ficam nesta pasta; os históricos ficam em `docs/archive/`.

## Executar

Abra um terminal nesta pasta e execute:

```sh
npm run dev
```

O comando inicia frontend e backend. Abra a URL informada pelo Vite; a porta pode variar. Sem `DATABASE_URL`, o backend usa memória e perde os cadastros ao encerrar. Detalhes em [DEV.md](DEV.md).

## Estado atual — 14/09/2026

- Gates 2 a 5: Health/Telemetry, Subscribers, Devices e Sessions por Device integrados.
- Micro-gate de estabilização: teste global de Devices corrigido; suíte Go e race detector passaram. PostgreSQL real não foi testado nesse micro-gate por ausência de `TEST_DATABASE_URL`.
- **Gate 6 — Live Network Topology: aprovado pelo Human Review.** Topologia e preview compartilham sessões reais por Device.
- **Gate 7 — Recent Events Feed: aprovado pelo Human Review.** Overview e Events usam os últimos 100 eventos consumidos pelo worker de Telemetry, somente na execução atual da API.
- Google Maps continua pausado. IP Pool Usage ainda não tem integração autoritativa.

## Documentos atuais

- [Execução local](DEV.md)
- [PostgreSQL local e migrations](../../docs/engineering/LOCAL_POSTGRES_SETUP.md)
- [Gate 7 — Recent Events: entrega e evidências](GATE7_RECENT_EVENTS.md)
- [Gate 6 — Live Network Topology: entrega e evidências](GATE6_LIVE_NETWORK_TOPOLOGY.md)
- [Gate 5 — Sessions: entrega e limitações](GATE5_SESSIONS_IMPLEMENTATION.md)
- [Estabilização da listagem global de Devices](DEVICE_GLOBAL_LIST_STABILIZATION.md)

O relatório de estabilização foi transferido da raiz do repositório para esta pasta para reunir a documentação da interface no mesmo lugar.

## Histórico

[Documentos anteriores](docs/archive/) preservam auditorias, planos e entregas dos Gates anteriores. São registros do estado na data em que foram escritos, não instruções atuais. Afirmações antigas como “somente mock” e “nenhuma integração implementada” foram superadas pelos Gates posteriores. Os caminhos mencionados nesses registros podem se referir à localização original dos documentos.

Nenhum Gate posterior ao Gate 7 foi iniciado. O checkpoint local está documentado em [Integration checkpoint](../../docs/engineering/INTEGRATION_CHECKPOINT_GATE7.md); nenhum push foi autorizado.
