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
	@echo "  db-setup       - compose-up + migrate-up + sqlc"
	@echo "  migrate-up     - apply DB migrations (starts compose stack)"
	@echo "  migrate-down   - revert last DB migration"
	@echo "  migrate-new    - create new migration (NAME=...)"
	@echo "  tools          - install dev tools (golangci-lint, sqlc, migrate)"
	@echo "  docker-build   - build Docker image"
	@echo "  compose-up     - start docker-compose stack"
	@echo "  compose-down   - stop docker-compose stack"
	@echo "  compose-logs   - follow parser scheduler logs"
	@echo "  import-full    - import full XML data"
	@echo "  import-delta   - import delta XML data"
	@echo "  unpack-full    - unpack latest full archive into source/xml/full"
	@echo "  unpack-delta   - unpack latest delta archive into source/xml/delta"
	@echo "  unpack-all     - unpack latest full and delta archives"
	@echo "  import-all     - import full XML data, then delta XML data"
	@echo "  load-full      - unpack and import latest full archive"
	@echo "  load-delta     - unpack and import latest delta archive"
	@echo "  clean          - remove build artifacts"

.PHONY: build
build:
	@mkdir -p $(BIN_DIR)
	go build $(GOFLAGS) -ldflags "$(LDFLAGS)" -o $(BIN_DIR)/$(BINARY) $(CMD_DIR)

.PHONY: import-full
import-full: build
	$(BIN_DIR)/$(BINARY) 0

.PHONY: import-delta
import-delta: build
	$(BIN_DIR)/$(BINARY) 1

.PHONY: unpack-full
unpack-full: build
	$(BIN_DIR)/$(BINARY) 2

.PHONY: unpack-delta
unpack-delta: build
	$(BIN_DIR)/$(BINARY) 3

.PHONY: unpack-all
unpack-all: build
	$(BIN_DIR)/$(BINARY) 2
	$(BIN_DIR)/$(BINARY) 3

.PHONY: import-all
import-all: build
	$(BIN_DIR)/$(BINARY) 0
	$(BIN_DIR)/$(BINARY) 1

.PHONY: load-full
load-full: build
	$(BIN_DIR)/$(BINARY) 2
	$(BIN_DIR)/$(BINARY) 0

.PHONY: load-delta
load-delta: build
	$(BIN_DIR)/$(BINARY) 3
	$(BIN_DIR)/$(BINARY) 1

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

.PHONY: db-setup
db-setup: compose-up migrate-up sqlc
	@echo "DB ready: migrations applied, sqlc code generated"

.PHONY: migrate-up
migrate-up: compose-up
	$(GOBIN)/migrate -path migrations -database "$(DB_URL)" up

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
	docker build -t $(BINARY):dev -f build/package/Dockerfile .

.PHONY: compose-up
compose-up:
	docker compose -f deployments/docker-compose.yml up -d

.PHONY: compose-down
compose-down:
	docker compose -f deployments/docker-compose.yml down

.PHONY: compose-logs
compose-logs:
	docker compose -f deployments/docker-compose.yml logs -f fias-parser

.PHONY: clean
clean:
	rm -rf $(BIN_DIR) $(COVERAGE)
