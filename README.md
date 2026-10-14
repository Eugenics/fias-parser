# fias-parser

Импортёр адресных данных ГАР в PostgreSQL. Работает через Docker Compose: контейнер планировщика по расписанию обрабатывает готовые ZIP-архивы, распаковывает XML и записывает данные в БД.

Архивы подготавливает отдельный сервис `fias-downloader`. В стандартной конфигурации он сообщает о готовых файлах через RabbitMQ, а `fias-parser` читает сообщения и импортирует полные выгрузки и дельты.

## Состав Docker Compose

Файл развёртывания — [deployments/docker-compose.yaml](deployments/docker-compose.yaml).

| Сервис | Контейнер | Назначение |
| --- | --- | --- |
| `fias-parser-postgres` | `fias-postgres` | PostgreSQL 16, данные хранятся в именованном томе `fias-pgdata`. |
| `fias-parser-scheduler` | `fias-parser` | Импортёр и cron. Образ собирается из [Dockerfile](build/package/Dockerfile). |

RabbitMQ и загрузчик архивов запускаются отдельно. Планировщик подключается к внешней сети `fias-downloader-network`, через которую должен быть доступен брокер. С PostgreSQL он общается через внутреннюю сеть Compose по адресу `fias-parser-postgres:5432`.

Для запуска нужны Docker с Compose, готовые архивы и доступ к RabbitMQ. Go и PostgreSQL на хосте устанавливать не требуется. Команды ниже выполняются из корня репозитория.

## Подготовка

Создайте `deployments/.env`:

```dotenv
DB_USER=fias
DB_PASSWORD=fias
DB_NAME=fias
DB_PORT=5434

TZ=Asia/Novosibirsk
FIAS_ARCHIVES_DIR=../../fias-downloader/var/data/downloads
RABBITMQ_URL=amqp://fias:fias@rabbitmq:5672/

FULL_SCHEDULE="0 2 * * 0"
DELTA_SCHEDULE="*/5 * * * *"
RUN_ON_STARTUP=false
STARTUP_LOAD_TYPE=delta
```

Укажите свои реквизиты БД и RabbitMQ. `rabbitmq` в примере URL — имя или сетевой псевдоним вашего брокера в `fias-downloader-network`; замените его на фактическое имя. `localhost` внутри контейнера указывает на сам контейнер.

`FIAS_ARCHIVES_DIR` задаёт каталог архивов на хосте. Относительный путь считается от `deployments/`, где лежит Compose-файл; можно указать абсолютный путь. Каталог должен существовать и быть доступен для чтения.

Проверьте наличие общей сети:

```bash
docker network inspect fias-downloader-network
```

Если сеть ещё не создана, создайте её и подключите к ней брокер средствами его развёртывания:

```bash
docker network create fias-downloader-network
```

Проверьте настройки Compose:

```bash
docker compose --env-file deployments/.env -f deployments/docker-compose.yaml config --quiet
```

## Первый запуск

### 1. Запустить PostgreSQL

```bash
docker compose --env-file deployments/.env -f deployments/docker-compose.yaml up -d --wait fias-parser-postgres
```

Порт `DB_PORT` используется для подключения с хоста. Внутри сети Docker PostgreSQL доступен на порту `5432`.

### 2. Применить миграции

Миграции находятся в `migrations/`. Планировщик не применяет их автоматически, поэтому выполните этот шаг до первого импорта:

```bash
docker run --rm \
  --network container:fias-postgres \
  -v "$PWD/migrations:/migrations:ro" \
  migrate/migrate:v4.19.1 \
  -path=/migrations \
  -database='postgres://fias:fias@127.0.0.1:5432/fias?sslmode=disable' \
  up
```

Команда использует сеть контейнера PostgreSQL. Реквизиты в `-database` должны совпадать с `DB_USER`, `DB_PASSWORD` и `DB_NAME` в `.env`. При изменении этих значений обновите строку подключения в команде.

### 3. Собрать и запустить планировщик

```bash
docker compose --env-file deployments/.env -f deployments/docker-compose.yaml up -d --build fias-parser-scheduler
docker compose --env-file deployments/.env -f deployments/docker-compose.yaml logs -f fias-parser-scheduler
```

При `RUN_ON_STARTUP=false` первая обработка произойдёт по расписанию. Для немедленной обработки очереди в работающем контейнере:

```bash
docker compose --env-file deployments/.env -f deployments/docker-compose.yaml exec fias-parser-scheduler run-scheduled.sh delta
```

В режиме RabbitMQ эта команда импортирует и полные выгрузки, и дельты из очереди. После обработки доступных сообщений процесс завершается; следующий запуск выполняет cron.

## Конфигурация контейнера

Планировщик читает [deployments/parser-config.yaml](deployments/parser-config.yaml), который монтируется в `/app/configs/config.yaml`. Файл `configs/config.yaml` используется при локальном запуске Go-программы и не является конфигурацией сервиса Compose.

| Параметр | Значение в конфигурации | Назначение |
| --- | --- | --- |
| `database.dsn` | `""` | Пустое значение позволяет взять `GAR_DATABASE_DSN`, который Compose формирует из переменных БД. |
| `importer.batch_size` | `1000` | Число записей в одном пакете записи. |
| `importer.workers` | `4` | Число одновременно обрабатываемых XML-файлов. |
| `rabbitmq.mode` | `rabbitmq` | Обработка событий очереди; значение `local` включает поиск архивов в каталоге. |
| `rabbitmq.queue` | `fias.download.completed` | Очередь готовых архивов. |
| `fias.archives_dir` | `/archives` | Каталог архивов внутри контейнера. |
| `fias.exp_delta` | `168h` | Интервал от регистрации версии до записываемого `exp_date`; автоматического удаления данных не выполняет. |
| `fias.imported_file_prefixes` | Список в YAML | Префиксы XML-файлов, которые извлекаются из архива. |

`RABBITMQ_URL` из `.env` имеет приоритет над `rabbitmq.url` в YAML. Значения `importer.batch_size` и `importer.workers` должны быть положительными.

После изменения `.env` пересоздайте планировщик, чтобы обновить окружение и расписание:

```bash
docker compose --env-file deployments/.env -f deployments/docker-compose.yaml up -d --force-recreate fias-parser-scheduler
```

## Архивы и хранилище

| На хосте | В контейнере | Доступ |
| --- | --- | --- |
| `FIAS_ARCHIVES_DIR` | `/archives` | Только чтение. |
| `source/xml` | `/app/source/xml` | Распаковка полной выгрузки в `full/`, дельты в `delta/`. |
| `deployments/parser-config.yaml` | `/app/configs/config.yaml` | Только чтение. |
| Том `fias-pgdata` | `/var/lib/postgresql/data` | Хранилище PostgreSQL. |

Ожидаемые имена архивов — `YYYYMMDD_full.zip` и `YYYYMMDD_delta.zip`, например `20260804_full.zip`. В архиве нужен `version.txt`: первая строка содержит дату, совпадающую с версией в имени архива. Поддерживаются форматы `2026.08.04`, `2026-08-04`, `04.08.2026` и `20260804`.

XML располагаются в каталогах регионов, а общие справочники могут лежать рядом с `version.txt`. Обёртка вида `gar_xml/` или `gar_delta_xml/`, содержащая каталоги регионов, снимается при распаковке. Файлы извлекаются во временный каталог; после проверки версии он заменяет предыдущую распакованную выгрузку.

Импортируются адресные объекты, их параметры, административная иерархия и справочники из списка префиксов в YAML. Данные домов, квартир, комнат, участков и муниципальной иерархии текущий импортёр пропускает; справочники типов домов, квартир и комнат поддерживаются.

## RabbitMQ

Загрузчик должен публиковать в настроенную очередь JSON-сообщения следующего вида:

```json
{
  "event": "download.completed",
  "version_id": 20260804,
  "kind": "delta",
  "file_path": "/data/downloads/20260804_delta.zip"
}
```

`kind` принимает `full` или `delta`. Имя файла должно соответствовать `version_id` и `kind`. Импортёр берёт имя файла из `file_path` и ищет его в `/archives`; исходный каталог загрузчика внутри контейнера импортёра не требуется.

Сообщение подтверждается после успешной обработки. При ошибке оно возвращается в очередь, а текущий запуск завершается с ошибкой. Сообщения обрабатываются в порядке получения, без сортировки по версии. Загрузчик должен обеспечить нужный порядок полной выгрузки и последующих дельт.

## Импорт локальных архивов без RabbitMQ

В `deployments/parser-config.yaml` установите `rabbitmq.mode: local`. В этом режиме архивы также читаются из `/archives`, а сеть `fias-downloader-network` остаётся требованием текущего Compose-файла.

Полная загрузка — распаковка последнего полного архива и его импорт:

```bash
docker compose --env-file deployments/.env -f deployments/docker-compose.yaml exec fias-parser-scheduler run-scheduled.sh full
```

Загрузка всех ещё не импортированных дельт по возрастанию версии:

```bash
docker compose --env-file deployments/.env -f deployments/docker-compose.yaml exec fias-parser-scheduler run-scheduled.sh delta
```

Для первичного наполнения БД сначала выполните полную загрузку. Локальный режим выбирает дельты по наличию статуса `imported`, поэтому в каталоге следует держать обновления, соответствующие вашей базовой выгрузке. Проверки непрерывности цепочки обновлений нет.

## Расписание

| Переменная `.env` | По умолчанию | Действие |
| --- | --- | --- |
| `DELTA_SCHEDULE` | `*/5 * * * *` | Каждые пять минут: обработка очереди в режиме RabbitMQ или ожидающих локальных дельт. |
| `FULL_SCHEDULE` | `0 2 * * 0` | Каждое воскресенье в 02:00: полная загрузка в режиме `local`. |
| `TZ` | `Asia/Novosibirsk` | Часовой пояс расписания. |
| `RUN_ON_STARTUP` | `false` | При `true` запускает задачу перед стартом cron. |
| `STARTUP_LOAD_TYPE` | `delta` | Задача при старте: `delta` или `full`. |

В режиме RabbitMQ используйте `STARTUP_LOAD_TYPE=delta`: команда обработки очереди принимает события обоих типов. Задача `full` вызывает команды CLI `2` и `0`, которые в режиме RabbitMQ пропускаются; полные выгрузки поступают через очередь.

Скрипт `run-scheduled.sh` использует общую блокировку в контейнере. Если загрузка ещё выполняется, очередной запуск задачи пропускается. Ручные команды выше используют ту же блокировку, что и cron.

## Проверка результата и обслуживание

Состояние контейнеров и логи:

```bash
docker compose --env-file deployments/.env -f deployments/docker-compose.yaml ps
docker compose --env-file deployments/.env -f deployments/docker-compose.yaml logs --tail=100 -f fias-parser-scheduler
```

Просмотр зарегистрированных версий с реквизитами из окружения контейнера БД:

```bash
docker compose --env-file deployments/.env -f deployments/docker-compose.yaml exec fias-parser-postgres \
  sh -c 'psql -U "$POSTGRES_USER" -d "$POSTGRES_DB" -c "SELECT version_id, file_type, status, updated_at FROM version_info ORDER BY version_id DESC, file_type;"'
```

Версия учитывается отдельно для `full` и `delta`. После распаковки она получает статус `extracted`, после успешного импорта — `imported`. Повторно импортированные версии пропускаются. Ошибки чтения XML и записи в БД завершают запуск с ошибкой; уже записанные пакеты при этом не откатываются целиком за всю выгрузку.

Остановить сервисы с сохранением тома БД:

```bash
docker compose --env-file deployments/.env -f deployments/docker-compose.yaml down
```

После обновления кода остановите планировщик, примените новые миграции командой из раздела первого запуска и пересоберите образ:

```bash
docker compose --env-file deployments/.env -f deployments/docker-compose.yaml stop fias-parser-scheduler
# Примените миграции перед следующей командой.
docker compose --env-file deployments/.env -f deployments/docker-compose.yaml up -d --build fias-parser-scheduler
```

| Симптом | Что проверить |
| --- | --- |
| Внешняя сеть не найдена | Создана ли `fias-downloader-network`. |
| Ошибка подключения к RabbitMQ | `RABBITMQ_URL`, реквизиты и доступность брокера в общей сети. |
| Архив не найден | Путь `FIAS_ARCHIVES_DIR`, имя файла и доступность файла внутри `/archives`. |
| Не найдена таблица `version_info` | Применены ли все миграции к той БД, которую использует контейнер. |
| Дата `version.txt` не совпадает с архивом | Соответствие содержимого архива его имени и версии события. |
| `Permission denied` при распаковке | Права на `source/xml` и его подкаталоги на хосте. |

## Команды Make

Все цели выполняются из корня репозитория. `make help` выводит список доступных команд.

### Docker Compose

| Команда | Действие |
| --- | --- |
| `make docker-build` | Собирает образ и пересоздаёт сервисы Compose с `--build --force-recreate`, явно читает `deployments/.env`. |
| `make compose-up` | Запускает весь стек в фоне, включая планировщик. |
| `make compose-down` | Останавливает и удаляет контейнеры стека, сохраняя том БД. |
| `make compose-logs` | Показывает логи планировщика в режиме наблюдения. |

Все эти цели используют `deployments/docker-compose.yaml`. У целей `compose-up`, `compose-down` и `compose-logs` в Makefile нет явного `--env-file`. Чтобы явно выбрать тот же файл окружения, что и в инструкциях выше, используйте:

```bash
COMPOSE_ENV_FILES="$PWD/deployments/.env" make compose-up
COMPOSE_ENV_FILES="$PWD/deployments/.env" make compose-logs
COMPOSE_ENV_FILES="$PWD/deployments/.env" make compose-down
```

### Миграции

| Команда | Действие |
| --- | --- |
| `make migrate-up` | Сначала выполняет `compose-up`, затем применяет все ожидающие миграции локальным инструментом `migrate`. |
| `make migrate-down` | Откатывает одну последнюю миграцию. |
| `make migrate-new NAME=add_index` | Создаёт пару последовательных файлов миграции `.up.sql` и `.down.sql` в `migrations/`. |

Эти команды требуют установленного на хосте `migrate`; путь к нему определяется через `GOBIN` (по умолчанию каталог `bin` внутри `GOPATH`). Строка подключения задаётся переменной Make `DB_URL`, по умолчанию — `postgres://fias:fias@localhost:5434/fias?sslmode=disable`. Makefile не загружает реквизиты БД из `deployments/.env` в `DB_URL`.

```bash
COMPOSE_ENV_FILES="$PWD/deployments/.env" make migrate-up \
  DB_URL='postgres://fias:fias@localhost:5434/fias?sslmode=disable'
make migrate-new NAME=add_address_index
```

Для первого развёртывания используйте порядок из раздела «Первый запуск»: запуск БД, миграции, запуск планировщика. `make migrate-up` запускает планировщик до применения миграций и не добавляет отдельного ожидания готовности БД. Запуск миграций через контейнер, описанный выше, не требует локального `migrate`.

### Локальный импорт и разработка

Для локальной сборки нужен Go 1.25. Модуль называется `gar_converter`. Локальный импортёр читает `configs/config.yaml` и запускается из корня проекта.

| Команда | Действие |
| --- | --- |
| `make build` | Собирает локальный бинарный файл `bin/fias-parser`. |
| `make load-full` | Собирает бинарный файл, выполняет команды `2` и `0`: распаковку последнего полного архива и импорт. Применяется в режиме `local`. |
| `make load-delta` | Собирает бинарный файл и выполняет команду `5`: импорт ожидающих локальных дельт или обработку очереди RabbitMQ, в зависимости от конфигурации. |
| `make test` | Запускает `go test -race -count=1 ./...`. |
| `make cover` | Запускает тесты с детектором гонок, записывает покрытие в `coverage.out` и выводит итоговый процент. |
| `make lint` | Запускает `golangci-lint run`. |
| `make fmt` | Форматирует Go-файлы через `gofmt` и `goimports`; локальный префикс импортов — `gar_converter`. |
| `make tidy` | Обновляет зависимости в `go.mod` и `go.sum` через `go mod tidy`. |
| `make sqlc` | Генерирует Go-код SQL-запросов по `sqlc.yaml`. |
| `make tools` | Устанавливает `golangci-lint`, `sqlc` и `migrate` с поддержкой PostgreSQL. |
| `make clean` | Удаляет каталог `bin` и файл `coverage.out`. |

`make load-full` и `make load-delta` выполняют импорт на хосте. Для импорта в контейнере используйте команды `docker compose exec ... run-scheduled.sh`, приведённые выше.

Для `make lint` нужен `golangci-lint` в `PATH`, для `make fmt` — `goimports` в `PATH`. `make tools` не устанавливает `goimports`. Цель `make sqlc` использует исполняемый файл из `GOBIN`.
