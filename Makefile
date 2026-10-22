BINARY   := bin/podcast
IMAGE    ?= podcast:local
GO       ?= go
COMPOSE  ?= docker compose

# golangci-lint is run through `go run` so it is always built with the module's Go toolchain.
GOLANGCI_LINT ?= go run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.15.0

.PHONY: build web run migrate test test-short test-int web-test e2e-ui e2e-ui-clean lint arch-lint up down dev-db tidy

## build: compile the server binary (the admin UI is not rebuilt).
build:
	$(GO) build -trimpath -o $(BINARY) .

## web: install the web app dependencies and build them into their dist folders.
web:
	cd web/admin && npm ci && npm run build
	cd web/player && npm ci && npm run build

## run: run the server from source (expects PostgreSQL on localhost:5432).
run:
	$(GO) run . serve

## migrate: apply the database migrations and exit.
migrate:
	$(GO) run . migrate

## test: run every test with the race detector.
test:
	$(GO) test -race -count=1 ./...

## test-short: run the unit tests, skipping container-based integration tests.
test-short:
	$(GO) test -short ./...

## test-int: run the container-backed integration and e2e API tests.
test-int:
	$(GO) test -count=1 -run . ./tests/... ./internal/catalog/outbound/pgadapter/... ./internal/listening/outbound/pgadapter/...

## web-test: run the admin and listener UI tests once.
web-test:
	cd web/admin && npm test -- --run
	cd web/player && npm test -- --run

## e2e-ui: run the Playwright suite against the docker compose stack.
e2e-ui:
	$(COMPOSE) up -d --build --wait
	cd tests/e2e-ui && npm ci && npx playwright install chromium && npm run e2e

## e2e-ui-clean: stop the stack and delete its volumes (screenshots stay).
e2e-ui-clean:
	$(COMPOSE) down -v

## lint: run go vet, golangci-lint and the arch-lint boundaries.
lint:
	$(GO) vet ./...
	$(GOLANGCI_LINT) run
	go-arch-lint check

## arch-lint: check the go-arch-lint boundaries.
arch-lint:
	go-arch-lint check

## up: build and start the full stack (PostgreSQL, migrations, app).
up:
	$(COMPOSE) up --build

## down: stop the stack and keep its volumes.
down:
	$(COMPOSE) down

## dev-db: start only the development PostgreSQL on localhost:5432.
dev-db:
	$(COMPOSE) -f compose.dev.yml up -d

## tidy: synchronise go.mod and go.sum.
tidy:
	$(GO) mod tidy
