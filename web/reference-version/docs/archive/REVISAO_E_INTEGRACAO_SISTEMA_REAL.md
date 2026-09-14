# NEXUS Core Lab — Revisão da interface e preparação para integração real

## Objetivo deste documento

Consolidar a revisão visual e funcional da interface atual e orientar a próxima etapa: conectar o frontend ao sistema real existente.

Este documento descreve o estado observado e propõe próximos passos. Não significa que a integração ou as correções listadas já tenham sido implementadas.

## Contexto do projeto

- Repositório local: `C:\Users\joaob\OneDrive\Documentos\nexus-core-lab`.
- Interface original: `web/`.
- Nova interface independente: `web/reference-version/`.
- A versão original deve ser preservada.
- A nova interface foi construída em React, TypeScript, Vite, CSS e SVG, com ícones Lucide e alguns SVGs próprios.
- A referência visual oficial é a imagem `ChatGPT Image 11 de set. de 2026, 00_11_48.png`, originalmente em 1536 × 1024.
- O dashboard usa essa composição como base e ajusta sua escala à área útil do navegador.
- A navegação atual usa fragmentos de URL, como `#overview`, `#subscribers` e `#network`.
- Atualmente, os dados de telecom apresentados nessa interface são mock. Não há integração com o backend real.
- O repositório contém código Go para assinantes, dispositivos, sessões, rede, telemetria, simulador e persistência PostgreSQL. Os contratos, endpoints e comportamentos disponíveis precisam ser inspecionados antes da integração; não devem ser presumidos a partir da aparência da interface.

## Mapa real de Campinas

É possível colocar um mapa interativo real de Campinas, carregado por um serviço cartográfico, com zoom, arraste e marcadores clicáveis.

O fundo atual já representa Campinas, mas é uma imagem estática armazenada localmente. Ele não é um mapa interativo consultado por API.

### Abordagem recomendada

Usar Leaflet com cartografia OpenStreetMap ou um provedor compatível. As antenas e dispositivos devem utilizar latitude e longitude para acompanhar corretamente o mapa durante zoom e deslocamento.

Leaflet suporta marcadores, linhas, pop-ups e eventos de interação. O serviço de mapas escolhido precisa respeitar atribuição, limites e regras de uso. OpenStreetMap fornece dados cartográficos; o servidor de tiles usado pela aplicação deve ser escolhido conscientemente, sobretudo para produção.

Referências oficiais:

- Leaflet: https://leafletjs.com/examples/quick-start/
- Referência da API Leaflet: https://leafletjs.com/reference
- Política dos tiles públicos OpenStreetMap: https://operations.osmfoundation.org/policies/tiles/
- Atribuição OpenStreetMap: https://www.openstreetmap.org/copyright

### Distinção importante

Cartografia real não significa que as antenas ou posições dos dispositivos sejam reais. Os marcadores atuais são demonstrativos. Na integração, suas posições devem vir de dados do sistema ou de uma configuração geográfica explicitamente identificada como demonstrativa. Não inventar coordenadas de infraestrutura real.

A integração cartográfica, isoladamente, não exige alterar o backend Go nem configurações do computador. O frontend pode consumir um serviço de mapas diretamente, conforme os requisitos de autenticação e licença do provedor.

Preservar a aparência noturna, as conexões verdes, o handover laranja, a legenda e os controles. Evitar deformar a cartografia para encaixá-la no container.

## Revisão das 12 telas

A análise foi feita com base nas capturas fornecidas pelo usuário e confirmada no código da nova interface. O Overview está mais desenvolvido; as telas internas ainda precisam de conteúdo específico e acabamento.

| Tela | Estado observado e lacunas |
|---|---|
| **Overview** | Boa composição visual. Os números ainda são estáticos e não refletem ações do simulador. |
| **Subscribers** | Busca e detalhes existem, mas os assinantes são derivados das cinco sessões mock. Falta uma lista própria coerente com o total anunciado de 128. |
| **Devices** | Estrutura adequada, porém só mostra cinco dispositivos e não diferencia claramente registro de conexão. |
| **Sessions** | Colunas pertinentes, mas “View all” continua mostrando apenas cinco das 17 sessões anunciadas. |
| **Network** | Precisa corrigir o transbordamento vertical e vincular marcadores a coordenadas geográficas. |
| **Events** | Há duas colunas chamadas “Details”. A última deveria ser “Actions”. Faltam filtros por tipo de evento e dispositivo. |
| **Telemetry** | Os indicadores são pertinentes, mas apenas repetem o Overview. O donut não corresponde corretamente às cores e proporções da legenda. |
| **Simulator** | Os botões só acrescentam texto ao histórico. Não alteram sessões, dispositivos, mapa ou indicadores. O histórico desaparece ao sair da tela. |
| **System** | Precisa mostrar módulos e estado do ambiente, em vez de repetir os KPIs gerais. |
| **API Status** | Precisa apresentar serviços, disponibilidade e respostas. Atualmente repete o conteúdo de System. |
| **Database** | Precisa apresentar informações de armazenamento e tabelas que o backend efetivamente disponibilizar. Atualmente também repete System. |
| **Configuration** | Os controles mudam visualmente, mas não afetam o sistema nem persistem ao sair da tela. |

### Avaliação visual geral

As telas internas mantêm as cores do dashboard, mas apresentam grandes áreas vazias, textos genéricos e pouca hierarquia entre indicadores, filtros e conteúdo.

Recomenda-se preservar a identidade visual já aprovada — sidebar, topbar, contorno diagonal, cores e profundidade dos painéis — e melhorar a organização interna de cada tela conforme sua finalidade.

### Limites da implementação atual

- Navegação clicável não equivale a integração funcional com o sistema real.
- Os indicadores de API, banco de dados e sistema online não comprovam disponibilidade de serviços reais.
- Os registros exibidos nas listas não representam integralmente os totais dos KPIs.
- A configuração atual é um estado local da tela, sem efeito operacional.
- O simulador atual é uma demonstração de interface, não uma execução do simulador Go.
- O mapa estático contém geografia real, mas a topologia de telecom sobreposta é demonstrativa.

## Mudança de direção para a próxima etapa

A recomendação anterior era conectar as telas a um estado mock compartilhado para que uma simulação alterasse os mesmos dados exibidos no restante da interface.

**A prioridade agora definida pelo usuário é conectar a interface ao sistema real existente.** Portanto, um estado mock compartilhado não deve substituir a integração nem se tornar uma etapa obrigatória extensa. Mocks podem continuar úteis para testes ou desenvolvimento isolado, desde que estejam separados e claramente identificados.

O objetivo é ter uma fonte consistente de dados do backend para Overview, listas, detalhes, eventos, telemetria e simulador, sem inventar contratos ou apresentar dados fictícios como reais.

## Preparação recomendada para integração

### 1. Inspecionar o backend existente

Ler as rotas HTTP, handlers, serviços, modelos, testes, documentação e configuração de inicialização.

Identificar, para cada operação:

- Método e rota existentes.
- Parâmetros, corpo da requisição e formato de resposta.
- Estados de domínio e validações.
- Tratamento de erros.
- Paginação e filtros, quando disponíveis.
- Recursos de health check e telemetria.
- Como o simulador Go é executado e quais operações já expõe.
- Configuração de persistência em memória ou PostgreSQL.

Não assumir que todos os elementos visuais já possuem um endpoint correspondente.

### 2. Mapear cada tela aos dados reais

Produzir uma matriz relacionando tela, informação exibida, origem no backend e eventual lacuna.

Quando faltar uma operação, registrar a ausência e propor a menor solução coerente com a arquitetura existente. Não criar endpoints redundantes sem antes verificar o que já existe.

### 3. Criar a camada de comunicação do frontend

- Centralizar configuração da URL da API e chamadas HTTP.
- Usar tipos compatíveis com os contratos reais.
- Tratar carregamento, erro, ausência de dados e indisponibilidade.
- Garantir que falhas não apareçam como sucesso ou indicadores verdes fixos.
- Evitar credenciais de banco ou segredos no frontend.
- Configurar proxy de desenvolvimento ou CORS conforme necessário.

### 4. Manter consistência entre telas

- KPIs devem corresponder aos dados reais, considerando paginação e critérios de contagem.
- “View all” deve abrir a listagem correspondente, sem reutilizar apenas as cinco linhas do resumo.
- Operações devem atualizar os dados afetados nas demais telas.
- Eventos, sessões e estado dos dispositivos devem refletir as mesmas transições de domínio.
- Usar polling, SSE ou WebSocket apenas conforme o suporte e os requisitos reais; não presumir um mecanismo existente.

### 5. Integrar o simulador corretamente

Verificar a interface real do simulador Go antes de criar controles. A UI deve representar execução, progresso, resultado e falhas de operações efetivas, respeitando as regras do domínio.

Não considerar a geração de uma mensagem no navegador como prova de que ocorreu um ATTACH, HANDOVER ou DETACH no backend.

### 6. Revisar System, API Status, Database e Configuration

Cada tela deve apresentar conteúdo próprio e sustentado por dados disponíveis:

- **System:** módulos e estado do ambiente.
- **API Status:** disponibilidade e respostas de verificações reais.
- **Database:** informações de persistência autorizadas e expostas pelo backend, sem conexão direta do navegador ao banco.
- **Configuration:** separar preferências locais de configurações operacionais; garantir persistência e efeito real quando aplicável.

### 7. Validar o resultado

- Executar frontend e backend no ambiente de desenvolvimento.
- Testar navegação, listas, filtros, detalhes e operações disponíveis.
- Verificar cenários de API indisponível e respostas de erro.
- Comparar KPIs e listas com os dados retornados pelo sistema.
- Validar o mapa e a localização dos marcadores.
- Revisar visualmente todas as telas, incluindo Network e seus controles inferiores.
- Testar diferentes áreas úteis de navegador, mantendo zoom em 100% e evitando cortes.
- Preservar a interface original durante o trabalho.

## Texto pronto para enviar ao ChatGPT

> Estou desenvolvendo o NEXUS Core Lab. O repositório já possui um backend Go com módulos de assinantes, dispositivos, sessões, rede, telemetria, simulador e persistência PostgreSQL. Foi criada uma nova interface React/TypeScript/Vite independente em `web/reference-version/`, preservando a versão original em `web/`.
>
> A interface já tem dashboard e navegação, mas ainda usa dados mock. Leia a revisão deste documento e me ajude a planejar a integração com o sistema real. A prioridade agora é consumir os contratos existentes do backend, e não ampliar a demonstração mock.
>
> Antes de propor alterações, inspecione os arquivos do projeto disponibilizados, identifique os endpoints e modelos reais e monte uma matriz de integração por tela. Não invente rotas, campos, métricas ou funcionalidades do backend. Se não tiver acesso ao repositório, indique quais arquivos precisa receber.
>
> Quero também substituir o mapa estático por um mapa interativo real de Campinas, preferencialmente usando Leaflet e cartografia OpenStreetMap ou um provedor compatível. Preserve a aparência noturna e use coordenadas geográficas para os marcadores. Diferencie geografia real de antenas ou dispositivos demonstrativos.
>
> Revise o conteúdo e o visual de todas as telas: Overview, Subscribers, Devices, Sessions, Network, Events, Telemetry, Simulator, System, API Status, Database e Configuration. Corrija a inconsistência entre KPIs e listas, a duplicação de conteúdo entre System/API Status/Database, a coluna duplicada em Events, o transbordamento de Network, a inconsistência do donut e as configurações sem persistência ou efeito.
>
> Preserve a versão original, mantenha a nova interface em `web/reference-version/` e proponha mudanças de backend somente quando uma lacuna real tiver sido comprovada. Entregue um plano incremental, com dependências, critérios de aceite e testes de integração, distinguindo claramente o que já existe do que precisa ser implementado.
