# go-url-shortener

Encurtador de URLs em Go. Biblioteca padrão (`net/http`, `log/slog`) para todo o resto, mais duas dependências externas, só para os bancos: `github.com/jackc/pgx/v5` (Postgres) e `github.com/redis/go-redis/v9` (Redis). Não adicionar outras sem necessidade.

## Idioma

- **Mensagens de commit: em português.** Exemplo: `adiciona endpoint de health check`.
- **Código: em inglês.** Isso inclui nomes de funções, tipos, variáveis, arquivos e pacotes, mensagens de log, mensagens de erro, respostas da API e comentários.

## Estrutura

```
cmd/server/main.go     # ponto de entrada: lê o ambiente, escolhe o storage e chama server.Run
internal/server/
  server.go            # type Server e Run (sobe e desliga o http.Server)
  routes.go            # registerRoutes: todas as rotas registradas em um só lugar
  health.go            # GET /healthz
  shorten.go           # POST /shorten
  code.go              # generateCode: código curto aleatório
  redirect.go          # GET /{code}
  stats.go             # GET /{code}/stats
internal/storage/
  storage.go           # interface Storage, type Link e os erros
  memory.go            # Memory: map em memória (desenvolvimento e testes)
  postgres.go          # Postgres: fonte da verdade (schema.sql é aplicado ao conectar)
  cached.go            # Cached: Redis na frente de outro Storage (cache-aside)
internal/clicks/
  counter.go           # Counter: conta cliques numa goroutine, sem travar o redirect
docker-compose.yml     # Postgres e Redis para desenvolvimento
```

## Convenções

- Cada arquivo tem o nome do que contém (ex.: handler de `/shorten` vai em `shorten.go`).
- Handlers são métodos de `*Server` com receiver `s`, minúsculos (não exportados).
- Toda rota é registrada em `registerRoutes`, usando o padrão com método HTTP (`"GET /healthz"`).
- Dependências (ex.: storage) entram como campos de `Server` e são criadas no `main`.
- Todo método de `Storage` recebe `context.Context` como primeiro argumento. Handlers passam `r.Context()`.
- Logs com `log/slog`, em formato chave/valor (`slog.Error("msg", "error", err)`).
- Imports em dois grupos: stdlib e, depois de uma linha em branco, os demais.
- Rodar `gofmt`, `go vet ./...` e `go test -race ./...` antes de considerar uma mudança pronta.

## Comandos

```bash
docker compose up -d                              # Postgres e Redis
set -a; source .env; set +a; go run ./cmd/server  # sobe com as variáveis do .env
go run ./cmd/server                               # sem .env: porta 8080, links em memória
curl -i localhost:8080/healthz
go vet ./...
go test -race ./...                               # testes de Postgres e Redis são pulados
TEST_DATABASE_URL='postgres://shortener:shortener@localhost:5432/shortener?sslmode=disable' \
TEST_REDIS_URL='redis://localhost:6379/15' go test -race ./...   # inclui Postgres e Redis
```
