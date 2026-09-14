# Gate 5 — Sessions por Device

Escopo aprovado pelo usuário em 13/09/2026: prosseguir com o julgamento de implementação, após proposta de consulta por Device, Attach, Handover e Detach.

Decisões: reutilizar Devices/Subscribers e transporte fetch; consulta individual sob demanda, sem agregação global nem polling Sessions; preservar Overview (KPI já vem da telemetria; tabela continua explicitamente mock). Alternativas: listagem global exigiria contrato backend novo; agregação por todos Devices causaria N consultas. Escolhida navegação por Device, conforme aprovação.

Aplicação local de laboratório, volume da coleção existente de Devices; seleção consulta uma sessão, refresh manual. Não adicionar autenticação, backend, DB, dependências, mapas ou persistência de sessões no navegador. Erros distinguem ausência confirmada (404 ACTIVE_SESSION_NOT_FOUND), falha e snapshot stale. Escritas aguardam servidor. API mantém autoridade sobre elegibilidade e IP. Manutenção restrita a web/reference-version.

Células: catálogo local espelhado de internal/network/cell.go (CELL-SP-001/002/003), pois não existe endpoint público de catálogo. Não é mapa Google nem leitura dinâmica de antenas. Atualizar o catálogo frontend se esse contrato mudar.

Testar API Go isolada em memória: attach, handover preservando IP/UUID, detach, ausência, erros, recovery, troca de Device, não polling; screenshots nas três resoluções; build. Ressalva Go do Gate4A continua documentada e fora desta edição.
