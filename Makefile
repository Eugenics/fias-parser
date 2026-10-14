BINARY      := fias-parser
PKG         := ./...
CMD_DIR     := .
BIN_DIR     := bin
COVERAGE    := coverage.out

GOFLAGS     ?=
LDFLAGS     ?= -s -w

GOLANGCI_LINT_VERSION ?= v1.61.0
SQLC_VERSION          ?= v1.31.1
MIGRATE_VERSION       ?= v4.19.1
GOIMPORTS_LOCAL       ?= gar_converter

GOBIN ?= $(shell go env GOPATH)/bin

DB_PORT=5434
DB_USER=fias
DB_PASSWORD=fias
DB_NAME=fias
DB_URL ?= postgres://$(DB_USER):$(DB_PASSWORD)@localhost:$(DB_PORT)/$(DB_NAME)?sslmode=disable

.PHONY: help
help:
	@echo "Targets:"
	@echo "  build          - build CLI binary into ./$(BIN_DIR)/$(BINARY)"
	@echo "  test           - run tests with race detector"
	@echo "  cover          - tests + coverage report"
	@echo "  lint           - run golangci-lint"
	@echo "  fmt            - format Go sources"
	@echo "  tidy           - update Go module files"
	@echo "  sqlc           - regenerate sqlc code"
	@echo "  migrate-up     - apply DB migrations (starts compose stack)"
	@echo "  migrate-docker-up - apply DB migrations using a Docker container"
	@echo "  migrate-down   - revert last DB migration"
	@echo "  migrate-new    - create new migration (NAME=...)"
	@echo "  tools          - install dev tools (golangci-lint, sqlc, migrate)"
	@echo "  docker-build   - rebuild image and recreate Compose containers"
	@echo "  compose-up     - start docker-compose stack"
	@echo "  compose-db-up  - start PostgreSQL and wait until healthy"
	@echo "  compose-down   - stop docker-compose stack"
	@echo "  compose-logs   - follow parser scheduler logs"
	@echo "  load-full      - unpack and import latest full archive"
	@echo "  load-delta     - unpack and import pending deltas (local) or full/delta events (RabbitMQ)"
	@echo "  clean          - remove build artifacts"

.PHONY: build
build:
	@mkdir -p $(BIN_DIR)
	go build $(GOFLAGS) -ldflags "$(LDFLAGS)" -o $(BIN_DIR)/$(BINARY) $(CMD_DIR)

.PHONY: load-full
load-full: build
	$(BIN_DIR)/$(BINARY) 2
	$(BIN_DIR)/$(BINARY) 0

.PHONY: load-delta
load-delta: build
	$(BIN_DIR)/$(BINARY) 5

.PHONY: test
test:
	go test -race -count=1 $(PKG)

.PHONY: cover
cover:
	go test -race -coverprofile=$(COVERAGE) $(PKG)
	go tool cover -func=$(COVERAGE) | tail -n 1

.PHONY: lint
lint:
	golangci-lint run

.PHONY: fmt
fmt:
	gofmt -w .
	goimports -w -local $(GOIMPORTS_LOCAL) .

.PHONY: tidy
tidy:
	go mod tidy

.PHONY: sqlc
sqlc:
	$(GOBIN)/sqlc generate

.PHONY: migrate-up
migrate-up: compose-up
	$(GOBIN)/migrate -path migrations -database "$(DB_URL)" up

.PHONY: migrate-docker-up
migrate-docker-up: compose-db-up
	docker run --rm --network container:fias-postgres \
		-v "$(CURDIR)/migrations:/migrations:ro" \
		migrate/migrate:$(MIGRATE_VERSION) -path=/migrations \
		-database="postgres://$(DB_USER):$(DB_PASSWORD)@127.0.0.1:5432/$(DB_NAME)?sslmode=disable" up

.PHONY: migrate-down
migrate-down:
	$(GOBIN)/migrate -path migrations -database "$(DB_URL)" down 1

.PHONY: migrate-new
migrate-new:
	@test -n "$(NAME)" || (echo "usage: make migrate-new NAME=<snake_case_name>" && exit 1)
	$(GOBIN)/migrate create -ext sql -dir migrations -seq $(NAME)

.PHONY: tools
tools:
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@$(GOLANGCI_LINT_VERSION)
	go install github.com/sqlc-dev/sqlc/cmd/sqlc@$(SQLC_VERSION)
	go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@$(MIGRATE_VERSION)

.PHONY: docker-build
docker-build:
	docker compose -f deployments/docker-compose.yaml --env-file deployments/.env up -d --build --force-recreate

.PHONY: compose-up
compose-up:
	docker compose -f deployments/docker-compose.yaml up -d

.PHONY: compose-db-up
compose-db-up:
	docker compose --env-file deployments/.env -f deployments/docker-compose.yaml up -d --wait fias-parser-postgres

.PHONY: compose-down
compose-down:
	docker compose -f deployments/docker-compose.yaml down

.PHONY: compose-logs
compose-logs:
	docker compose -f deployments/docker-compose.yaml logs -f fias-parser-scheduler

.PHONY: clean
clean:
	rm -rf $(BIN_DIR) $(COVERAGE)
