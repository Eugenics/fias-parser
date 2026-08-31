# GAR Reader

CLI-утилита на Go для скачивания и импорта XML-выгрузок Государственного адресного реестра (ГАР) в PostgreSQL.

## Возможности

- скачивание полной или дельта-выгрузки ГАР с сервисов ФНС;
- распаковка XML по регионам в `source/xml/full` или `source/xml/delta`;
- параллельный импорт XML-данных в PostgreSQL;
- применение миграций и генерация запросов `sqlc`;
- запись информации о загруженной версии в таблицу `version_info`;
- пропуск повторного импорта версии, уже отмеченной как импортированная.

## Требования

- Go 1.25 или новее;
- PostgreSQL 16+;
- Docker и Docker Compose — для локальной базы данных;
- для служебных целей: `golangci-lint`, `sqlc` и `migrate`.

## Быстрый старт

Все команды необходимо выполнять из корня репозитория.

1. Создайте файл `deployments/.env` для PostgreSQL:

   ```dotenv
   DB_USER=fias
   DB_PASSWORD=change-me
   DB_NAME=fias
   DB_PORT=5432
   ```

2. Укажите строку подключения в `configs/config.yaml`:

   ```yaml
   database:
     dsn: postgres://fias:change-me@localhost:5432/fias?sslmode=disable
   ```

   Если `database.dsn` пуст, приложение использует переменную окружения `GAR_DATABASE_DSN`.

3. Установите инструменты разработки и подготовьте БД:

   ```bash
   make tools
   make db-setup
   ```

   `make db-setup` запускает PostgreSQL, применяет миграции и генерирует код `sqlc`.

4. Скачайте и импортируйте полную выгрузку:

   ```bash
   make download-full
   make import-full
   ```

## Команды CLI

| Аргумент | Действие | Make-цель |
| --- | --- | --- |
| `0` | Импорт полной выгрузки из `source/xml/full` | `make import-full` |
| `1` | Импорт дельта-выгрузки из `source/xml/delta` | `make import-delta` |
| `2` | Скачать полную XML-выгрузку | `make download-full` |
| `3` | Скачать дельта XML-выгрузку | `make download-delta` |
| `4` | Показать сведения о последней выгрузке ФНС | `make download-info` |

Примеры запуска без Makefile:

```bash
go run . 4
go run . 2
go run . 0
```

## Конфигурация

Конфигурация расположена в `configs/config.yaml`.

```yaml
database:
  dsn: postgres://USER:PASSWORD@HOST:5432/DATABASE?sslmode=disable

importer:
  batch_size: 1000
  workers: 4

fias:
  all_info_url: http://fias.nalog.ru/WebServices/Public/GetAllDownloadFileInfo
  last_info_url: http://fias.nalog.ru/WebServices/Public/GetLastDownloadFileInfo
  full_name: gar_xml.zip
  delta_name: gar_delta_xml.zip
```

- `database.dsn` — обязательная строка подключения к PostgreSQL;
- `importer.batch_size` — размер пакета при записи в БД;
- `importer.workers` — максимальное число одновременно обрабатываемых XML-файлов;
- `fias.*` — адреса сервисов ФНС и имена архивов. При отсутствии `full_name` и `delta_name` используются значения `gar_xml.zip` и `gar_delta_xml.zip`.

## Структура данных

После скачивания данные располагаются так:

```text
source/xml/
├── full/
│   ├── version.txt
│   └── <код-региона>/*.XML
└── delta/
    ├── version.txt
    └── <код-региона>/*.XML
```

Импортер ожидает XML-файлы в подкаталогах регионов. XML-файлы, размещённые непосредственно в каталоге выгрузки, обрабатываются как регион `00`.

## Сборка и проверка

```bash
make build       # bin/gar_reader
make test        # go test с race detector
make lint        # golangci-lint
make fmt         # gofmt и goimports
make docker-build
```

Для Windows также доступны `compile_win.bat` и `compile_linux.bat` для кросс-компиляции в `amd64`.

## Работа с базой данных

```bash
make compose-up
make migrate-up
make migrate-down
make migrate-new NAME=add_example_field
make sqlc
make compose-down
```

По умолчанию команды миграций используют `DB_URL`. При необходимости передайте собственную строку подключения:

```bash
make migrate-up DB_URL='postgres://USER:PASSWORD@HOST:5432/DATABASE?sslmode=disable'
```

## Важное замечание

После обработки приложение записывает метаданные выгрузки в `version_info`. При наличии `version.txt` повторный импорт версии, уже имеющей статус `imported`, пропускается.

Ошибки обработки отдельных XML-файлов и записи данных выводятся в консоль. Проверяйте вывод импорта: успешный код завершения сам по себе не гарантирует, что все файлы были импортированы без ошибок.
