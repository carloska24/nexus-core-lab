# Desenvolvimento local — PostgreSQL e migrations

A API já lê DATABASE_URL do ambiente. Sem essa variável, a API usa memória. O comando de migration exige DATABASE_URL ou o argumento -url explícito, sem credenciais padrão.

## PostgreSQL local

Na raiz do repositório, copie .env.example para .env e substitua change_me por uma senha local exclusiva. .env é ignorado pelo Git. O Compose lê esse arquivo automaticamente; o programa Go não o lê.

```powershell
Copy-Item .env.example .env
# Edite .env antes de iniciar o banco.
docker compose up -d postgres
```

Mantidos os valores existentes de desenvolvimento: usuário nexus, banco nexus_core_lab e porta do host 5433 (ajustável por POSTGRES_PORT). A senha agora é obrigatória via POSTGRES_PASSWORD, sem fallback no Compose.

## Migrations e API

Exporte DATABASE_URL no terminal que executará Go. Use a senha local escolhida; caracteres especiais na senha devem ser codificados para URL. Não compartilhe a URL nem a inclua em documentação ou commits. O placeholder abaixo não é uma credencial operacional:

```powershell
$env:DATABASE_URL = 'postgres://nexus:change_me@localhost:5433/nexus_core_lab?sslmode=disable'
go run ./cmd/migrate -up
cd web/reference-version
npm run dev
```

Troque o placeholder antes de executar. DATABASE_URL é herdada pelo backend iniciado pelo script combinado. O argumento -url permanece compatível, mas a variável evita incluir a credencial diretamente na linha de comando. Não execute migrations sem intenção de alterar o banco configurado.

Volumes PostgreSQL já inicializados conservam a senha existente: alterar POSTGRES_PASSWORD no Compose não altera automaticamente a senha do banco. Não remova volumes para aplicar esta correção. Este checkpoint não inicia banco, não executa migrations nem muda senhas de instâncias existentes.

As credenciais literais anteriores foram removidas do estado atual, mas permanecem nos commits anteriores. O histórico não foi reescrito. Caso tenham sido reutilizadas fora do desenvolvimento local, precisam ser rotacionadas antes de compartilhar o repositório.
