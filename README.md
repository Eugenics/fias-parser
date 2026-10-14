# fias-parser

Консольный импортёр адресных данных ГАР из XML в PostgreSQL. Программа распаковывает локальные архивы полной выгрузки и обновлений, загружает поддерживаемые сущности и хранит состояние версий в таблице `version_info`.

## Требования

- Go 1.25 для локальной сборки;
- PostgreSQL 16 (или совместимый сервер);
- архивы ГАР в формате ZIP, подготовленные `fias-downloader`;
- `golang-migrate` для применения миграций (`make tools` устанавливает его вместе с другими инструментами разработки).

Все команды ниже запускаются из корня репозитория: программа читает `./configs/config.yaml` и использует относительные пути к XML.

## Быстрый запуск локально

1. Создайте `deployments/.env` для Docker Compose:

   ```dotenv
   DB_USER=fias
   DB_PASSWORD=fias
   DB_NAME=fias
   DB_PORT=5434
   ```

2. Запустите PostgreSQL, примените миграции и соберите программу:

   ```bash
   docker network create fias-downloader-network
   docker compose --env-file deployments/.env -f deployments/docker-compose.yaml up -d fias-parser-postgres
   make tools
   "$(go env GOPATH)/bin/migrate" -path migrations -database 'postgres://fias:fias@localhost:5434/fias?sslmode=disable' up
   make build
   ```

   Сеть создаётся один раз; если она уже существует, пропустите эту команду. Проверьте, что `database.dsn` в `configs/config.yaml` указывает на вашу БД; в примере используется порт `5434`. Пароль и порт в команде миграции должны совпадать с `deployments/.env`.

3. Укажите в `configs/config.yaml` каталог с архивами (`fias.archives_dir`). Ожидаемые имена: `<версия>_full.zip` и `<версия>_delta.zip`, например `20260804_full.zip`. Версия в имени должна совпадать с датой в `version.txt` внутри архива.

4. Загрузите полную выгрузку, затем обновления:

   ```bash
   make load-full
   make load-delta
   ```

`make load-full` распаковывает последний полный архив и импортирует его. `make load-delta` обрабатывает ещё не импортированные локальные дельты по возрастанию версии. Программа сама не скачивает архивы из сети.

## Команды CLI

После `make build` исполняемый файл находится в `bin/fias-parser`.

| Команда | Действие |
| --- | --- |
| `bin/fias-parser 0` | Импортировать уже распакованную полную выгрузку из `source/xml/full`. |
| `bin/fias-parser 1` | Импортировать уже распакованную дельту из `source/xml/delta`. |
| `bin/fias-parser 2` | Распаковать последнюю локальную полную выгрузку. |
| `bin/fias-parser 3` | Распаковать последнюю локальную дельту. |
| `bin/fias-parser 5` | Импортировать ожидающие дельты по порядку; в режиме RabbitMQ обработать сообщения очереди. |

Распаковка заменяет соответствующий каталог `source/xml/full` или `source/xml/delta` и записывает версию со статусом `extracted`. После успешного импорта статус становится `imported`. Повторный импорт версии со статусом `imported` пропускается. Если предыдущая распакованная версия ещё не импортирована, новая распаковка блокируется.

Для ручного импорта XML нужен `version.txt` в корне соответствующего каталога. Первая строка содержит дату версии (`2026.08.04`, `2026-08-04`, `04.08.2026` или `20260804`); вторая строка может содержать её текстовое название. XML размещаются в каталогах регионов, например `source/xml/full/77/*.XML`; файлы прямо в `source/xml/full` также читаются. Имя каждого XML выбирается при обходе каталога, поэтому конкретные имена файлов задавать не требуется.

## Конфигурация

Основной файл — `configs/config.yaml`. Пример ключевых параметров:

```yaml
database:
  dsn: postgres://fias:fias@localhost:5434/fias?sslmode=disable
importer:
  batch_size: 1000
  workers: 4
rabbitmq:
  mode: local
fias:
  archives_dir: ../fias-downloader/var/data/downloads
  exp_delta: 168h
  imported_file_prefixes:
    - AS_ADDR_OBJ_
    - AS_ADDR_OBJ_PARAMS_
    - AS_ADM_HIERARCHY_
```

Полный список импортируемых префиксов находится в `configs/config.yaml`. `batch_size` задаёт число записей в пакете записи, `workers` — число одновременно обрабатываемых XML-файлов. Если `database.dsn` пуст, используется переменная окружения `GAR_DATABASE_DSN`. Для `importer` значения по умолчанию — 1000 и 4; `rabbitmq.mode` по умолчанию `local`.

В режиме `rabbitmq` команда `5` читает очередь `rabbitmq.queue` по адресу `rabbitmq.url` (его можно задать через `RABBITMQ_URL`). Сообщения указывают на готовые архивы полной выгрузки или дельты. Остальные команды CLI в этом режиме не выполняются. Конфигурация контейнера для такого запуска находится в `deployments/parser-config.yaml`.

## Контейнер и расписание

`deployments/docker-compose.yaml` поднимает PostgreSQL и планировщик импорта. Перед запуском нужны `deployments/.env`, доступный каталог архивов (`FIAS_ARCHIVES_DIR`) и внешняя сеть `fias-downloader-network`. Для интерполяции переменных используйте `docker compose --env-file deployments/.env -f deployments/docker-compose.yaml up -d`. По умолчанию контейнер использует режим RabbitMQ и подключается к брокеру через `RABBITMQ_URL` из `.env`.

Расписание задаётся переменными `FULL_SCHEDULE` (по умолчанию `0 2 * * 0`) и `DELTA_SCHEDULE` (по умолчанию `*/5 * * * *`). `RUN_ON_STARTUP=true` запускает загрузку при старте; тип выбирает `STARTUP_LOAD_TYPE=full|delta`. В текущем режиме RabbitMQ команда `5` обрабатывает и полные выгрузки, и дельты, а задача `full` вызывает команды `2` и `0`, которые в этом режиме пропускаются. Логи: `make compose-logs`. Остановка: `make compose-down`.

## Разработка

```bash
make test       # go test -race -count=1 ./...
make cover      # тесты с отчётом о покрытии
make lint       # golangci-lint
make fmt        # gofmt и goimports
make sqlc       # пересоздать код запросов из sql/queries
```

Схема БД хранится в `migrations/`, SQL-запросы — в `sql/queries/`, сгенерированный код — в `internal/db/`. Импорт XML выполняется в `internal/app/`, запись в PostgreSQL — в `internal/repository/`.
