# HookForge — структура проекта

## Как это работает в целом 

```
Внешняя система
    → POST /webhooks/:public_id   (API)
    → PostgreSQL (событие + delivery)
    → Kafka (лёгкое сообщение)
    → Worker pool
    → HTTP POST на target_url клиента
```

Dashboard ходит только в `/api/v1/*` (с API key).  
Приём webhook’ов — отдельно, через HMAC.

---

## Корень репозитория

| Путь | Зачем |
|------|--------|
| `docker-compose.yml` | Поднимает всё: API, worker, frontend, Postgres, Redis, Kafka, Prometheus, Grafana |
| `Makefile` | Короткие команды: `test`, `build`, `compose-up` |
| `README.md` | Документация продукта |
| `.env.example` | Пример переменных окружения |
| `scripts/send-webhook.sh` | Локальный демо-скрипт: создать endpoint + подписать + отправить webhook |
| `deploy/` | Конфиги Prometheus/Grafana |
| `ARCHITECTURE.md` | Этот файл — описание структуры и ответственности модулей |

---

## Backend (`backend/`)

### Точки входа — `cmd/`

| Бинарь | Роль |
|--------|------|
| `cmd/api` | HTTP-сервер: management API + приём webhook’ов + `/demo` + `/metrics` |
| `cmd/worker` | Читает Kafka, гоняет worker pool, доставляет webhook’и |
| `cmd/seed` | Наполняет БД демо-данными |

Это два процесса специально: API можно масштабировать отдельно от доставки.

### Домен и бизнес-логика — `internal/`

| Пакет | Отвечает за |
|-------|-------------|
| `domain/` | Сущности (`Endpoint`, `Event`, `Delivery`…) и ошибки API |
| `service/` | Бизнес-сценарии: создать endpoint, ingest webhook, retry, API keys |
| `repository/` | SQL к PostgreSQL (CRUD по таблицам) |
| `api/` | HTTP handlers + маршруты `/api/v1`, `/webhooks`, `/demo` |
| `middleware/` | Request ID, логи, CORS, API-key auth, recover, metrics, лимит тела |
| `auth/` | Генерация/хеш API keys, public_id, secrets |
| `hmac/` | Подпись и проверка `X-Webhook-Signature` |
| `ssrf/` | Запрет доставки на localhost/private/metadata IP |
| `delivery/` | HTTP-клиент доставки + backoff/retry правила |
| `worker/` | Пул воркеров: Kafka → job channel → N goroutines |
| `kafka/` | Producer (API) / Consumer (worker) |
| `redis/` | Rate limit (и мелкий cache/idempotency helper) |
| `postgres/` | Connection pool + миграции |
| `config/` | Чтение env |
| `logger/` | Structured JSON logs |
| `metrics/` | Prometheus-метрики |

### Схема БД — `migrations/`

`00001_init.sql` — таблицы:

- `endpoints` — куда слать
- `events` — что пришло
- `deliveries` — прогресс доставки
- `delivery_attempts` — каждая попытка
- `api_keys` — ключи dashboard/API (только hash)

### Документация API

`backend/api/openapi.yaml` — OpenAPI-спека.

---

## Frontend (`frontend/`)

Nuxt 4 dashboard.

| Путь | Зачем |
|------|--------|
| `app/layouts/default.vue` | Sidebar + top bar |
| `app/pages/index.vue` | Dashboard / stats |
| `app/pages/endpoints` | Список/создание endpoints |
| `app/pages/events` | Список + детали события/timeline |
| `app/pages/deliveries` | Исходящие доставки |
| `app/pages/api-keys` | Управление ключами |
| `app/pages/analytics` | Сводка метрик |
| `app/pages/settings` | UI настроек (отражает env-конфиг) |
| `app/composables/useApi.ts` | HTTP-клиент к backend |
| `app/components/*` | Badge, modal, empty state, cards |

В development ключ подставляется сам из `NUXT_PUBLIC_DEFAULT_API_KEY`.

---

## Кто с кем говорит

```
Browser (Nuxt :3000)
    → Bearer API key
    → HookForge API (:18080) → PostgreSQL / Redis

External system
    → HMAC headers
    → POST /webhooks/:public_id
    → API → Postgres + Kafka

Worker
    → Kafka consumer
    → Postgres (payload/status)
    → HTTP POST target_url
    → Postgres (attempts/status) + metrics
```

---

## Короткая ментальная модель

- **API** — приём и управление
- **Worker** — доставка и retry
- **Postgres** — source of truth
- **Kafka** — очередь между ними
- **Redis** — rate limit
- **Frontend** — UI над management API
- **Prometheus/Grafana** — наблюдаемость

---

## Пример потока: webhook → DELIVERED

1. Клиент шлёт `POST /webhooks/:public_id` с JSON-телом и заголовками:
   - `X-Event-ID`
   - `X-Webhook-Timestamp`
   - `X-Webhook-Signature`
2. `api` + `middleware` принимают request, режут размер тела, пишут request_id.
3. `service.Ingest`:
   - находит endpoint по `public_id`
   - проверяет active / rate limit (Redis)
   - проверяет HMAC (`hmac`)
   - пишет `events` + `deliveries` в Postgres (`repository`)
   - публикует в Kafka только IDs (`kafka.Producer`)
4. `worker` читает сообщение из Kafka.
5. Worker pool берёт payload из Postgres, валидирует URL (`ssrf`), шлёт HTTP (`delivery.Client`).
6. Результат пишется в `delivery_attempts`, статусы event/delivery обновляются.
7. При успехе — `DELIVERED`; при retryable ошибке — `RETRYING` + backoff; иначе — `FAILED`.
8. Метрики уходят в Prometheus, UI читает состояние через `/api/v1/*`.
