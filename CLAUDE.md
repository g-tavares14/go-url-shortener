# go-url-shortener

Encurtador de URLs em Go, usando apenas a biblioteca padrão (`net/http`, `log/slog`).

## Idioma

- **Mensagens de commit: em português.** Exemplo: `adiciona endpoint de health check`.
- **Código: em inglês.** Isso inclui nomes de funções, tipos, variáveis, arquivos e pacotes, mensagens de log, mensagens de erro, respostas da API e comentários.

## Estrutura

```
cmd/server/main.go     # ponto de entrada: cria o context (Ctrl+C) e chama server.Run
internal/server/
  server.go            # type Server e Run (sobe e desliga o http.Server)
  routes.go            # registerRoutes: todas as rotas registradas em um só lugar
  health.go            # GET /healthz
```

## Convenções

- Cada arquivo tem o nome do que contém (ex.: handler de `/shorten` vai em `shorten.go`).
- Handlers são métodos de `*Server` com receiver `s`, minúsculos (não exportados).
- Toda rota é registrada em `registerRoutes`, usando o padrão com método HTTP (`"GET /healthz"`).
- Dependências (ex.: storage) entram como campos de `Server` e são criadas no `main`.
- Logs com `log/slog`, em formato chave/valor (`slog.Error("msg", "error", err)`).
- Rodar `gofmt` e `go vet ./...` antes de considerar uma mudança pronta.

## Comandos

```bash
go run ./cmd/server          # sobe na porta 8080
curl -i localhost:8080/healthz
go vet ./...
```
