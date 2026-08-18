IMAGEM  ?= clima-cep:dev
ENVFILE ?= deployments/.env
PORTA   ?= 8080

.PHONY: help
help: ## Lista os alvos
	@grep -hE '^[a-z-]+:.*?##' $(MAKEFILE_LIST) | awk -F':.*?## ' '{printf "  %-16s %s\n", $$1, $$2}'

.PHONY: fmt
fmt: ## Formata o código
	gofmt -w ./cmd ./internal

.PHONY: vet
vet: ## Análise estática (inclui a tag roundtrip)
	go vet ./...
	go vet -tags roundtrip ./...

.PHONY: test
test: ## Testes offline, sem chave
	go test ./...

.PHONY: test-roundtrip
test-roundtrip: ## Testes contra ViaCEP e WeatherAPI reais (exige $(ENVFILE))
	set -a && . ./$(ENVFILE) && set +a && go test -tags roundtrip -count=1 ./internal/api/

.PHONY: check
check: fmt vet test ## Formata, analisa e testa

.PHONY: run
run: ## Sobe o servidor sem container (exige $(ENVFILE))
	set -a && . ./$(ENVFILE) && set +a && go run ./cmd/server

.PHONY: build
build: ## Compila o binário em bin/server
	CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o bin/server ./cmd/server

.PHONY: docker-build
docker-build: ## Constrói a imagem de produção
	docker build -f deployments/Dockerfile -t $(IMAGEM) .

.PHONY: up
up: docker-build ## Sobe via docker compose
	docker compose -f deployments/docker-compose.yml up --build

.PHONY: down
down: ## Derruba o docker compose
	docker compose -f deployments/docker-compose.yml down
