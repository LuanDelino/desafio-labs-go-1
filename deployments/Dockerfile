FROM golang:1.26-alpine AS build

WORKDIR /src

COPY go.mod go.sum* ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build \
    -trimpath -ldflags="-s -w" \
    -o /out/server ./cmd/server

# :nonroot já traz usuário sem privilégio e os certificados raiz, obrigatórios
# porque ViaCEP e WeatherAPI só respondem em HTTPS.
FROM gcr.io/distroless/static-debian12:nonroot

COPY --from=build /out/server /server

EXPOSE 8080
USER nonroot:nonroot

ENTRYPOINT ["/server"]
