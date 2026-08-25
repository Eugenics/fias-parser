BINARY      := gar_reader
PKG         := ./...
CMD_DIR     := .
BIN_DIR     := ./bin
COVERAGE    := coverage.out

GOFLAGS     ?=
LDFLAGS     ?= -s -w

GOLANGCI_LINT_VERSION ?= v1.61.0
SQLC_VERSION          ?= v1.31.1
MIGRATE_VERSION       ?= v4.19.1

GOBIN ?= $(shell go env GOPATH)/bin

DB_URL ?= postgres://fias:fias@localhost:5432/fias?sslmode=disable

.PHONY: help
help:
# 	@echo "Targets:"
	@echo "  build          - build CLI binary into ./$(BIN_DIR)/$(BINARY)"
# 	@echo "  test           - run tests with race detector"
# 	@echo "  cover          - tests + coverage report"
# 	@echo "  lint           - run golangci-lint"
# 	@echo "  fmt            - gofmt + goimports"
# 	@echo "  tidy           - go mod tidy"
	@echo "  sqlc           - regenerate sqlc code"
	@echo "  db-setup       - compose-up + migrate-up + sqlc"
	@echo "  migrate-up     - apply DB migrations (starts compose stack)"
	@echo "  migrate-down   - revert last DB migration"
	@echo "  migrate-new    - create new migration (NAME=...)"
	@echo "  tools          - install dev tools (golangci-lint, sqlc, migrate)"
	@echo "  docker-build   - build Docker image"
	@echo "  compose-up     - start docker-compose stack"
	@echo "  compose-down   - stop docker-compose stack"
	@echo "  import-full   	- full import of gar data"
	@echo "  import-delta  	- delta import of gar data"
	@echo "  download-full  - download full GAR XML into source/xml/full"
	@echo "  download-delta - download GAR XML delta into source/xml/delta"
	@echo "  download-info  - show info about full/delta files from last_info_url"
# 	@echo "  clean          - remove build artifacts"

.PHONY: build
build:
	@mkdir -p $(BIN_DIR)
	@mkdir -p $(CMD_DIR)
	go build $(GOFLAGS) -ldflags "$(LDFLAGS)" -o $(BIN_DIR)/$(BINARY) $(CMD_DIR)

.PHONY: import-full
import-full:
	go run $(CMD_DIR) 0

.PHONY: import-delta
import-delta:
	go run $(CMD_DIR) 1

.PHONY: download-full
download-full:
	go run $(CMD_DIR) 2

.PHONY: download-delta
download-delta:
	go run $(CMD_DIR) 3

.PHONY: download-info
download-info:
	go run $(CMD_DIR) 4

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
	goimports -w -local fias .

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
	go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest
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

.PHONY: clean
clean:
	rm -rf $(BIN_DIR) $(COVERAGE)
