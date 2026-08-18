# Estagio de build: compila estatico, sem CGO, para caber no distroless.
FROM golang:1.26-alpine AS build

WORKDIR /src

# go.mod/go.sum primeiro: as dependencias so' sao rebaixadas quando eles mudam.
COPY go.mod go.sum* ./
RUN go mod download

COPY . .

# -trimpath tira o caminho da maquina do binario; -s -w descartam tabela de
# simbolos e DWARF, que nao servem em producao.
RUN CGO_ENABLED=0 GOOS=linux go build \
    -trimpath -ldflags="-s -w" \
    -o /out/server ./cmd/server

# Estagio final: sem shell, sem gerenciador de pacotes, sem compilador.
# O :nonroot ja' vem com usuario sem privilegio e com os certificados raiz,
# que sao obrigatorios — ViaCEP e WeatherAPI so' respondem em HTTPS.
FROM gcr.io/distroless/static-debian12:nonroot

COPY --from=build /out/server /server

EXPOSE 8080
USER nonroot:nonroot

ENTRYPOINT ["/server"]
