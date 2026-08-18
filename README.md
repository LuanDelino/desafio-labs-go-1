# Clima por CEP

API em Go que recebe um CEP brasileiro, descobre a cidade e devolve a
temperatura atual em Celsius, Fahrenheit e Kelvin.

## URL no Cloud Run

<https://clima-cep-172943883906.us-central1.run.app>

```bash
curl https://clima-cep-172943883906.us-central1.run.app/01310100
```

## Contrato

### `GET /{cep}`

Também aceito como `GET /?cep=01310100`.

**200 OK**

```json
{
  "temp_C": 21.7,
  "temp_F": 71.06,
  "temp_K": 294.7
}
```

**Falhas**

Sucesso e falha usam o mesmo formato: quem consome decodifica JSON sempre, sem
ramificar por status.

```json
{ "message": "invalid zipcode" }
```

| Condição | Status | `message` |
| --- | --- | --- |
| CEP fora do formato de 8 dígitos | `422` | `invalid zipcode` |
| CEP bem formado, mas inexistente | `404` | `can not find zipcode` |
| ViaCEP ou WeatherAPI indisponível | `502` | `upstream unavailable` |

Há ainda `GET /health`, usado como sonda.

### Conversões

    F = C * 1.8 + 32
    K = C + 273

O valor é publicado com duas casas decimais. O enunciado traz `K = C + 273`
na fórmula e `301.65` no exemplo (que corresponde a `C + 273.15`); este
projeto segue a fórmula. A constante está isolada em
`internal/temperature/temperature.go`.

## Rodando localmente

Requer a chave gratuita da [WeatherAPI](https://www.weatherapi.com/).

```bash
cp deployments/.env.example deployments/.env
# edite deployments/.env e preencha WEATHER_API=
```

### Via Docker Compose

```bash
make up          # sobe em http://localhost:8080
make down
```

Se a porta 8080 já estiver ocupada na sua máquina:

```bash
PORTA=8091 make up
```

`PORTA` troca só a porta do host — dentro do container o servidor continua em
8080, como no Cloud Run.

O compose constrói a **mesma imagem** que vai para produção, e falha na hora se
`WEATHER_API` não estiver definida, em vez de subir um servidor que devolveria
500 em toda requisição.

### Via Docker, sem compose

```bash
make docker-build
docker run --rm -p 8080:8080 --env-file deployments/.env clima-cep:dev
```

### Sem Docker

```bash
make run
```

`PORT` controla a porta do servidor (padrão `8080`); no Cloud Run ela é injetada.

### Experimentando

```bash
curl localhost:8080/01310100   # 200
curl localhost:8080/1234567    # 422
curl localhost:8080/99999999   # 404
```

### Atalhos

`make help` lista todos. Os que importam:

| Alvo | O que faz |
| --- | --- |
| `make up` / `make down` | sobe e derruba via compose |
| `make test` | testes offline, sem chave |
| `make test-roundtrip` | testes contra ViaCEP e WeatherAPI reais |
| `make check` | `fmt` + `vet` + `test` |
| `make run` | servidor sem container |
| `make build` | binário em `bin/server` |
| `make docker-build` | imagem de produção |

## Testes

```bash
make test        # ou: go test ./...
```

Rodam offline e sem chave: as chamadas ao ViaCEP e à WeatherAPI são servidas
por `httptest`, incluindo os casos que só acontecem em produção — o ViaCEP
respondendo **200 OK** com `{"erro": true}` para CEP inexistente, e as duas
formas (`true` e `"true"`) que esse campo já teve.

Para exercitar as APIs de verdade:

```bash
make test-roundtrip
```

## Organização

    cmd/server/            binário
    internal/cep/          resolve CEP em cidade/UF (ViaCEP)
    internal/weather/      temperatura atual da cidade (WeatherAPI)
    internal/temperature/  conversões entre escalas
    internal/api/          borda HTTP: rotas, contrato JSON, status
    deployments/Dockerfile imagem de produção
    deployments/docker-compose.yml  stack de desenvolvimento
    Makefile               atalhos (make help)

`cep` e `weather` não se conhecem — quem costura é `api`, que declara as
interfaces de que precisa no próprio pacote (consumer-side), e é por isso que
o handler é testável com dublês sem que os clientes saibam da borda.

A imagem final é `distroless/static:nonroot` (~16 MB): sem shell, sem
gerenciador de pacotes e sem compilador.

## Deploy

A imagem publicada é a mesma que o `docker build` local produz — não há build
separado para produção.

```bash
IMG=us-central1-docker.pkg.dev/<PROJETO>/apps/clima-cep:$(git rev-parse --short HEAD)

docker build -f deployments/Dockerfile -t "$IMG" .
docker push "$IMG"

gcloud run deploy clima-cep \
  --image="$IMG" --region=us-central1 \
  --allow-unauthenticated --port=8080 \
  --cpu=1 --memory=128Mi --min-instances=0 --max-instances=1 \
  --set-secrets=WEATHER_API=WEATHER_API:latest
```

A chave vai por Secret Manager, não por `--set-env-vars`: assim ela não
aparece em `gcloud run services describe` nem no log de deploy. A tag da
imagem é o hash do commit, para que a revisão em produção seja rastreável até
o código que a gerou.
