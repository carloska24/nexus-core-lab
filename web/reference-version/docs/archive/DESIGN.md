# NEXUS Core Lab — Reference Version

## Objetivo

Reproduzir a composição visual da referência oficial em um frontend isolado, editável e local. O alvo principal é 1920×1080, com adaptações para 1440×900 e 1366×768.

## Restrições

- Dados exclusivamente mock.
- Sem integração com backend, autenticação, endpoints ou alterações em Go.
- Sem mapas externos ou recortes da imagem de referência.
- Todos os arquivos ficam em `web/reference-version/`.

## Design

A aplicação usa React e SVG/CSS local. A grade principal mantém sidebar fixa, topbar, cinco KPIs, uma área central com topologia e duas tabelas, e três painéis analíticos inferiores. O mapa urbano é um SVG composto por distritos, vias e texturas, sobre o qual ficam células, dispositivos, conexões e o arco de handover.

Em 1920×1080, o painel inteiro cabe sem rolagem. Em alturas menores, variáveis de densidade reduzem espaços e alturas mantendo a hierarquia. Em larguras abaixo de 1500 px, a sidebar e a tipografia são reduzidas de forma moderada.

## Decisões

1. SVG/CSS local foi escolhido por preservar editabilidade e permitir fidelidade sem dependências externas.
2. A topologia usa uma malha urbana irregular, e não círculos concêntricos ou linguagem de radar.
3. Gráficos são SVG locais para garantir aparência e dimensões determinísticas.
4. Os dados são estáticos e reproduzem os valores da referência.

## Riscos

O principal risco é a densidade vertical em 1366×768. A mitigação é uma composição escalonada por media queries, sem reorganizar a ordem dos painéis.
