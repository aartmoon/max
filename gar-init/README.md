# GAR/FIAS database initializer

`gar-init` — одноразовый Go-контейнер для подготовки ГАР/ФИАС в PostgreSQL.
В режиме `demo` он загружает проверенный fixture Арбата; в режиме `full`
потоково читает XML. После успешной загрузки и построения поискового индекса
процесс завершается с кодом `0`.

## Почему большие XML не переполняют память

Файлы читаются напрямую через `encoding/xml.Decoder.Token`. Документ никогда
не загружается целиком. В памяти одновременно находятся внутренний буфер XML
decoder и не более `GAR_BATCH_SIZE` строк (по умолчанию 10 000). Каждый batch
передаётся в PostgreSQL через `pgx.CopyFrom`.

Один XML-файл загружается в одной транзакции. Если parsing или `COPY` завершится
ошибкой, все строки этого файла откатятся, а `gar_import_state` не получит
отметку об успехе. Уже завершённые файлы повторно не читаются.

## Конфигурация

Обязательные переменные:

```dotenv
POSTGRES_HOST=postgres
POSTGRES_PORT=5432
POSTGRES_DB=tvoydom
POSTGRES_USER=tvoydom
POSTGRES_PASSWORD=secret
GAR_MODE=demo
```

Дополнительные:

```dotenv
GAR_DATA_DIR=/data/gar
GAR_BATCH_SIZE=10000
GAR_LOG_EVERY=100000
```

`GAR_MODE=demo` не читает `GAR_DATA_DIR` и загружает встроенный канонический
fixture. `GAR_MODE=full` никогда не подставляет демо-данные: отсутствующая,
неполная или повреждённая выгрузка завершает процесс ошибкой.

## Полный XML-импорт вручную

В `.env` корня проекта:

```dotenv
POSTGRES_PASSWORD=надежный-пароль
GAR_XML_PATH=/home/user1/gar/xml
```

Проверить файлы и запустить отдельный контейнер с явным `GAR_MODE=full` и
read-only монтированием каталога в `/data/gar`. Обычные Compose-файлы этот
режим не включают и XML не монтируют.

```bash
find /home/user1/gar/xml -maxdepth 1 -type f -name '*.XML' | sort
docker run --rm --network max_backend \
  -e GAR_MODE=full -e GAR_DATA_DIR=/data/gar \
  -e POSTGRES_HOST=postgres -e POSTGRES_PORT=5432 \
  -e POSTGRES_DB -e POSTGRES_USER -e POSTGRES_PASSWORD \
  -v /home/user1/gar/xml:/data/gar:ro artshar/max-gar-init:latest
```

Следить за прогрессом:

```bash
docker compose -f docker-compose.prod.yml logs -f gar-init
```

Обычный production deploy использует встроенный demo-fixture:

```bash
./deploy.sh
```

## Проверка результата

Открыть `psql`:

```bash
docker compose exec postgres psql -U "$POSTGRES_USER" -d "$POSTGRES_DB"
```

Состояние файлов и всего импорта:

```sql
SELECT file_name, entity_type, row_count, completed_at
FROM gar_import_state
ORDER BY completed_at;

SELECT initialized, initialized_at, source_date, source_type
FROM gar_import_metadata;
```

Количество основных сущностей:

```sql
SELECT 'address_objects' AS entity, count(*) FROM gar_address_objects
UNION ALL SELECT 'houses', count(*) FROM gar_houses
UNION ALL SELECT 'apartments', count(*) FROM gar_apartments
UNION ALL SELECT 'rooms', count(*) FROM gar_rooms
UNION ALL SELECT 'carplaces', count(*) FROM gar_carplaces
UNION ALL SELECT 'steads', count(*) FROM gar_steads
UNION ALL SELECT 'search_addresses', count(*) FROM gar_search_addresses;
```

Проверка поискового слоя:

```sql
SELECT object_kind, count(*)
FROM gar_search_addresses
GROUP BY object_kind
ORDER BY object_kind;

SELECT object_id, object_guid, object_kind, full_address
FROM gar_search_addresses
WHERE object_kind IN ('house', 'apartment')
ORDER BY object_id
LIMIT 20;
```

## Безопасный перезапуск после падения

Исправить причину ошибки и снова выполнить:

```bash
docker compose -f docker-compose.prod.yml up gar-init
```

Текущий неуспешный файл начнётся с начала, потому что его транзакция была
откачена. Успешные файлы будут пропущены по `gar_import_state`. Если все данные
уже загружены, но упало создание индексов, повторный запуск выполнит только
финализацию. После полного успеха следующий запуск выводит:

```text
[GAR] GAR database already initialized
```

и завершается с кодом `0`.

Если файл с уже сохранённым именем изменил размер, job завершится ошибкой и не
будет смешивать разные снимки ГАР. Для загрузки другого полного снимка нужна
отдельная управляемая миграция; этот контейнер намеренно не удаляет таблицы и не
выполняет `TRUNCATE`.

## Разработка и тесты

Локальный `docker-compose.yml` загружает тот же fixture Арбата, что production:

```bash
docker compose up --build -d
```

Unit-тесты:

```bash
cd gar-init
go test ./...
```

PostgreSQL integration-тесты:

```bash
export GAR_TEST_DATABASE_URL='postgres://tvoydom:tvoydom_demo@localhost:5433/tvoydom?sslmode=disable'
go test ./internal/db -count=1
```

Полный DDL находится в `internal/db/schema.sql`, индексы — в
`internal/db/indexes.sql`, построение поискового слоя — в
`internal/db/search.sql`.
