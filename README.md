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

| Condição | Status | Corpo |
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

### Via Docker

```bash
docker build -f deployments/Dockerfile -t clima-cep:dev .
docker run --rm -p 8080:8080 --env-file deployments/.env clima-cep:dev
```

```bash
curl localhost:8080/01310100
curl localhost:8080/1234567    # 422
curl localhost:8080/99999999   # 404
```

### Sem Docker

```bash
export WEATHER_API=sua-chave
go run ./cmd/server
```

`PORT` controla a porta (padrão `8080`); no Cloud Run ela é injetada.

## Testes

```bash
go test ./...
```

Rodam offline e sem chave: as chamadas ao ViaCEP e à WeatherAPI são servidas
por `httptest`, incluindo os casos que só acontecem em produção — o ViaCEP
respondendo **200 OK** com `{"erro": true}` para CEP inexistente, e as duas
formas (`true` e `"true"`) que esse campo já teve.

Para exercitar as APIs de verdade:

```bash
set -a && . ./deployments/.env && set +a
go test -tags roundtrip -count=1 ./internal/api/
```

## Organização

    cmd/server/            binário
    internal/cep/          resolve CEP em cidade/UF (ViaCEP)
    internal/weather/      temperatura atual da cidade (WeatherAPI)
    internal/temperature/  conversões entre escalas
    internal/api/          borda HTTP: rotas, contrato JSON, status
    deployments/Dockerfile imagem de produção

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
