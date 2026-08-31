# AGENTS.md — gar_reader

## Overview
Go CLI that imports Russian GAR address XML files into PostgreSQL. Module `gar_converter`, Go 1.25, uses pgx/v5. No tests exist in the repo.

## Commands
```bash
go build -o myapp main.go        # build from repo root
./compile_linux.bat              # cross-compile GOOS=linux GOARCH=amd64 → myapp
./compile_win.bat                # cross-compile → myapp.exe
make build                       # builds bin/gar_reader
make import-full                 # go run . 0 — full import
make import-delta                # go run . 1 — delta import
make download-full               # go run . 2 — download full GAR XML to source/xml/full
make download-delta              # go run . 3 — download GAR XML delta to source/xml/delta
make compose-up                  # start local Postgres (requires deployments/.env)
make lint / test / cover / fmt   # need golangci-lint tool; fmt uses goimports -local fias
```

## Data flow
- `main.go` — XML paths (`./source/xml/full`, `./source/xml/delta`); arg `0` full import, `1` delta import, `2`/`3` download full/delta XML, `4` print last-version info from FIAS (Russian usage otherwise).
- `internal/app/support.go` — `GetStucturedData` walks `<path>/<region>/*.XML`; files placed directly in `<path>/` (no region subdir) get assigned to a pseudo-region `"00"`.
- `internal/app/app.go` — one goroutine per XML file, limited by `config.Importer.Workers` semaphore (`app.go:27`).
- `internal/app/xml.go` → `Handle` → `internal/app/handlers/import/*` — one parser handler per XML entity type (`addressObjects.go`, `params.go`, `admHierarchyItems.go`, etc.).
- `internal/app/objectHandler.go` + `internal/helpers` — dispatch utilities.
- `internal/repository/postgres.go` — DB writes. Automatic word of caution: **errors during parse/write are only printed, goroutines in `app.go` return no errors and `Run` returns nil on success**, so a zero exit code does NOT mean a clean import.
- Version recording: after a successful run of args `0`–`3`, `main.go` calls `recordVersion` → `downloader.LastInfo()` → `repository.SaveVersionInfo` (upsert into `version_info` via sqlc-generated `db.UpsertVersionInfo`; status `"imported"` for 0/1, `"downloaded"` for 2/3). Best-effort — failures are only logged.
- `internal/download/download.go` — `Downloader` (config-driven, `internal/config.FiasConfig`): `LastInfo` fetches `fias.last_info_url` (FIAS `GetLastDownloadFileInfo`, returns JSON, XML fallback) → `FiasCompleteXmlUrl`/`FiasDeltaXmlUrl` (fallback `GarXMLFullURL`/`GarXMLDeltaURL`, then `fias.url` + `fias.full_name`/`fias.delta_name`); downloads the zip, strips the single top-level wrapper dir (`gar_xml/`, `gar_delta_xml/`) so region folders land directly in the target dir.

## Config
- `internal/config/config.go` reads YAML via `gopkg.in/yaml.v3`. Path is hardcoded in `app.go:20` — **must run `/...` `./configs/config.yaml` from repo root**.
- Parses `database.dsn`, `importer.batch_size`/`importer.workers` and the `fias:` block (`url`, `all_info_url`, `last_info_url`, `delta_name`, `full_name`). `FiasConfig` defaults zip names to `gar_xml.zip`/`gar_delta_xml.zip` when empty.
- Defaults in code if missing: batch_size 1000, workers 4.
- `database.dsn` is required; falls back to `GAR_DATABASE_DSN` env var only if the YAML value is empty.

## Gotchas
- No tests — Makefile `test`/`cover` targets will fail until those pieces exist.
- `make compose-up` uses `deployments/.env` (DB_USER=fias, DB_PASSWORD=fias, DB_NAME=fias, DB_PORT=5432) — advent of variables in docker-compose.
- XML input must follow the exact layout `source/xml/{full,delta}/<region>/*.XML`; the downloader flattens the zip wrapper dir so this holds after `make download-*`.
- XML file names embed UUIDs and dates, so file names can't be hardcoded — always iterate directory scan like `support.go`.
- No README.
```