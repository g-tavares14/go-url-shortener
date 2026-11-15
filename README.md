# go-url-shortener

Encurtador de URLs escrito em Go. Recebe uma URL longa, devolve um código curto e redireciona quem abre o link curto. Conta os cliques sem atrasar o redirect, aceita links que expiram e guarda tudo no Postgres, com Redis como cache.

```bash
$ curl -X POST localhost:8080/shorten -d '{"url":"https://go.dev/doc"}'
{"code":"aBDsfFAj","short_url":"http://localhost:8080/aBDsfFAj"}

$ curl -i localhost:8080/aBDsfFAj
HTTP/1.1 302 Found
Location: https://go.dev/doc
```

## Como rodar

Pré-requisitos: Go 1.27+ e Docker.

```bash
docker compose up -d                              # sobe Postgres e Redis
cp .env.example .env                              # variáveis de ambiente
set -a; source .env; set +a; go run ./cmd/server  # servidor na porta 8080
```

O Go não lê o `.env` sozinho: o `set -a; source .env; set +a` exporta as variáveis antes de rodar.

Para experimentar sem Docker, rode só `go run ./cmd/server`. Sem `DATABASE_URL`, os links ficam em memória e somem quando o servidor para.

## API

| Método | Rota | Sucesso | Erros |
|---|---|---|---|
| `POST` | `/shorten` | `201` com o código e a URL curta | `400` JSON, URL ou `expires_in` inválidos |
| `GET` | `/{code}` | `302` para a URL original | `404` código não existe · `410` link expirado |
| `GET` | `/{code}/stats` | `200` com URL, cliques e expiração | `404` código não existe |
| `GET` | `/healthz` | `200` `{"status":"ok"}` | |

Qualquer falha interna responde `500` com `internal server error`. O detalhe vai para o log do servidor.

### `POST /shorten`

```json
{ "url": "https://go.dev", "expires_in": 3600 }
```

- `url`: obrigatória, só `http` ou `https`.
- `expires_in`: opcional, em segundos, até 1 ano. Sem ele, ou com `0`, o link nunca expira.

```json
{ "code": "aBDsfFAj", "short_url": "http://localhost:8080/aBDsfFAj", "expires_at": "2026-11-15T13:00:00Z" }
```

`expires_at` só aparece quando o link expira.

### `GET /{code}/stats`

```json
{ "code": "aBDsfFAj", "url": "https://go.dev", "clicks": 42, "expires_at": "2026-11-15T13:00:00Z" }
```

## Configuração

Tudo por variáveis de ambiente. O `.env.example` traz os valores para o `docker-compose.yml`.

| Variável | Padrão | Para que serve |
|---|---|---|
| `PORT` | `8080` | Porta HTTP do servidor. |
| `BASE_URL` | `http://localhost:$PORT` | Endereço público usado para montar o `short_url`. |
| `DATABASE_URL` | vazio | Conexão com o Postgres. Vazio: links em memória. |
| `REDIS_URL` | vazio | Conexão com o Redis. Vazio: sem cache. |

Se `DATABASE_URL` ou `REDIS_URL` estiverem definidos e o serviço estiver fora do ar, o servidor não sobe e diz qual conexão falhou.

## Arquitetura

```mermaid
flowchart LR
    C[Cliente] --> H["Handlers HTTP<br/>shorten · redirect · stats"]
    H --> S{{"interface Storage"}}
    S --> R[("Redis<br/>cache")]
    R -. miss .-> P[("Postgres<br/>fonte da verdade")]
    H -- clique --> Q[["canal com buffer"]]
    Q --> G(["goroutine do contador"])
    G --> S
```

**Criar um link (`POST /shorten`).** O handler valida a URL, sorteia um código de 8 letras com `crypto/rand` e salva. Se o código já existir, sorteia outro, até 5 tentativas. O link é gravado no Postgres e já colocado no Redis, porque costuma ser aberto logo depois de criado.

**Abrir um link (`GET /{code}`).** O handler busca o código, primeiro no Redis e, se não estiver lá, no Postgres, e então preenche o Redis. Se o link expirou, responde `410`. Senão, registra o clique e responde `302`.

**Contar cliques.** O redirect só coloca o código num canal com buffer de 1024 posições e segue em frente. Uma goroutine tira os cliques do canal e grava no Postgres. Se a fila encher, o clique é descartado: perder uma contagem é melhor que deixar o redirect lento. Ao desligar com Ctrl+C, o servidor termina as requests em andamento e grava os cliques que ainda estavam na fila.

### Decisões

- **`Storage` é uma interface.** Os handlers não sabem se os dados estão em memória, no Postgres ou atrás do Redis. Cada implementação tem a mesma interface, e o mesmo conjunto de testes roda contra as três.
- **O cache embrulha outro storage.** `Cached` também implementa `Storage` e recebe o Postgres por dentro (cache-aside). Se o Redis cair com o servidor rodando, as leituras vão direto ao Postgres e o servidor só registra um aviso no log.
- **Os cliques ficam fora do `Link`.** Um link (URL e expiração) nunca muda depois de criado, por isso pode ficar no cache sem risco de ficar desatualizado. Os cliques mudam a cada visita e sempre vão direto ao Postgres.
- **O cache respeita a expiração.** Um link fica no Redis por até 1 hora, e nunca além do seu `expires_at`.
- **`302`, não `301`.** O navegador guarda um `301` para sempre e deixaria de passar pelo servidor, o que impediria contar cliques e fazer o link expirar.
- **`410` para link expirado.** O `404` diz que o link nunca existiu. O `410` diz que ele existiu e acabou.
- **Códigos aleatórios.** Com `crypto/rand`, os códigos não são sequenciais nem adivinháveis. Com 8 letras (a-z, A-Z), são cerca de 53 trilhões de combinações.

## Testes

```bash
go test -race ./...
```

Os testes de Postgres e Redis são pulados quando as variáveis abaixo não estão definidas. Para rodar tudo, com o `docker compose` de pé:

```bash
TEST_DATABASE_URL='postgres://shortener:shortener@localhost:5432/shortener?sslmode=disable' \
TEST_REDIS_URL='redis://localhost:6379/15' \
go test -race ./...
```

Esses testes **apagam a tabela `links` e o banco 15 do Redis**. Não aponte essas variáveis para dados que você queira guardar.

O `-race` faz parte: ele verifica que o mutex do storage em memória e o contador de cliques funcionam com várias goroutines ao mesmo tempo.

## Estrutura

```
cmd/server/main.go       ponto de entrada: lê o ambiente, escolhe o storage e sobe o servidor
internal/server/         handlers HTTP, rotas e o type Server
internal/storage/        interface Storage e as implementações Memory, Postgres e Cached
internal/clicks/         contador de cliques em segundo plano
docker-compose.yml       Postgres e Redis para desenvolvimento
```

## Dependências

Fora a biblioteca padrão, só duas: [`pgx`](https://github.com/jackc/pgx) para o Postgres e [`go-redis`](https://github.com/redis/go-redis) para o Redis.
