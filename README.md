# WB Tech L0 Store Service

Сервис принимает сообщения о заказах из Kafka, сохраняет их в PostgreSQL и отдаёт через HTTP API и простой веб-интерфейс. Для ускорения повторных запросов используется in-memory LRU‑кеш.

## Архитектура

- **Kafka (KRaft)** – источник сообщений с информацией о заказах.
- **PostgreSQL** – основное хранилище данных.
- **Go-сервис** (`cmd/storesvc`) – обрабатывает сообщения, управляет кешем, запускает HTTP API и UI.
- **Goose** (`cmd/migrator`) – инструмент миграций БД.
- **LRU‑кеш** (`internal/cache`) – хранит последние заказы, наполняется из консьюмера и HTTP слоя.
- **Веб-интерфейс** – одна HTML‑страница для ручного запроса по `order_uid`.

## Используемый стек

- Go 1.25
- Kafka 4.1.x (официальный образ `apache/kafka`, режим KRaft)
- PostgreSQL 17
- Goose для миграций
- Docker / Docker Compose

## Конфигурация

Переменные окружения читаются в `cmd/storesvc/main.go`. Пример – в `.env.example`.

| Переменная | Значение по умолчанию | Описание |
|------------|-----------------------|----------|
| `PG_DSN` | `postgres://user:pass@localhost:5432/orders?sslmode=disable` | DSN для PostgreSQL |
| `KAFKA_BROKERS` | `localhost:9092` | Список брокеров Kafka |
| `KAFKA_TOPIC` | `orders` | Топик, из которого читаются сообщения |
| `KAFKA_GROUP` | `ordersvc` | Идентификатор consumer group |
| `HTTP_ADDR` | `:8081` | Адрес, на котором слушает HTTP‑сервер |
| `WARMUP_LIMIT` | `1000` | Кол-во записей для прогрева кеша из БД (0 – отключено) |
| `CACHE_CAPACITY` | `1000` | Максимум ключей в LRU‑кеше |
| `AUTO_MIGRATE` | `false` | Автоматически запускать миграции при старте сервиса |

## Локальный запуск через Docker Compose

```bash
docker compose build
docker compose up
```

Compose поднимает:

- `postgres` – с volume `pgdata` (персистентно).
- `kafka` – Kafka 4.1.0 в режиме KRaft.
- `storesvc` – наш сервис, миграции выполняются автоматически (`AUTO_MIGRATE=true`).

Топик `store` создаётся сервисом, однако можно подготовить вручную:

```bash
docker compose exec kafka /opt/kafka/bin/kafka-topics.sh \
  --bootstrap-server kafka:9092 \
  --create --topic store --partitions 1 --replication-factor 1
```

### Отправка тестового сообщения

```bash
docker compose exec -T kafka /opt/kafka/bin/kafka-console-producer.sh \
  --bootstrap-server kafka:9092 \
  --topic store <<'EOF'
{"order_uid":"b563feb7b2b84b6test", ...}
EOF
```

После обработки:
- данные появятся в таблицах `orders`, `deliveries`, `payments`, `items`;
- на `http://localhost:8081/order/<order_uid>` вернётся JSON;
- веб-страница на `http://localhost:8081/` покажет информацию при вводе `order_uid`.

## Работа с миграциями

SQL‑скрипты лежат в `migrations/`. Один файл содержит блоки `-- +goose Up/Down`.

- Автозапуск при старте (`AUTO_MIGRATE=true`).
- Ручное применение/откат:

```bash
go run ./cmd/migrator up
go run ./cmd/migrator down        # откат на один шаг
go run ./cmd/migrator down 2      # откат на два шага
```

## Тесты и генерация моков

```bash
go generate ./...
go test ./...
```

`go generate` обновляет моки для интерфейсов (`internal/mocks`). Тесты покрывают логіку LRU‑кеша и ключевые сценарии HTTP API / Kafka консьюмера.

## Cache & Warmup

- LRU ограничен параметром `CACHE_CAPACITY` (по умолчанию 1000). При переполнении выбрасывает самые старые ключи.
- При запуске консьюмер создает топик, подключается и после каждого сообщения обновляет кеш (`X-Cache: HIT/MISS` можно отследить в ответах HTTP).
- Прогрев из БД включается при `WARMUP_LIMIT > 0` – в актуальной compose‑конфигурации отключён (`0`), чтобы при чистом старте не тратить время.

## Полезные команды

```bash
docker compose down -v          # остановить и удалить volumes (чистый запуск)
docker compose logs storesvc    # посмотреть логи сервиса
docker compose exec postgres psql -U store -d store -c '\dt'   # таблицы
```

## Структура репозитория

- `cmd/storesvc` – основной сервер.
- `cmd/migrator` – CLI для goose.
- `internal/cache` – LRU кеш.
- `internal/domain` – модели данных заказа.
- `internal/repo` – Postgres репозиторий.
- `internal/httpapi` – HTTP обработчики и UI.
- `internal/kafkaconsumer` – Kafka consumer с валидацией входящих сообщений.
- `internal/validation` – обёртка над `go-playground/validator`.
- `internal/migrate` – обёртка для миграций (Ensures таблицу версий).
- `internal/mocks` – автогенерируемые моки (не редактировать вручную).

---

Если при старте сервис не получает партицию (Kafka ещё не готова), `ensureTopic` создаёт топик и ретраит подключение. Это защищает от ситуации «консьюмер стартовал раньше топика». Для уверенности всегда дождитесь в логах `consumer START` и проверьте, что новое сообщение появилось в таблицах БД или выдаётся API.
