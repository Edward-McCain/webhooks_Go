# HookForge

Reliable webhook delivery infrastructure written in Go.

HookForge accepts inbound webhooks, persists them, fans them out through Kafka, and delivers them to customer endpoints with retries, observability, and strong security defaults.

## Features

- Versioned REST API for endpoints, events, deliveries, API keys, and stats
- Signed webhook ingestion (HMAC-SHA256) with replay protection
- Idempotent event intake (`endpoint_id` + `external_event_id`)
- Asynchronous delivery via Kafka and a bounded worker pool
- Exponential backoff with jitter
- Redis-backed per-endpoint rate limiting
- SSRF protection for outbound delivery targets
- Prometheus metrics and Grafana dashboard
- Structured JSON logging with request correlation IDs
- Demo webhook endpoint for local delivery scenarios

## Architecture

```
HTTP API
  ↓
PostgreSQL (source of truth)
  ↓
Kafka (webhook.events)
  ↓
Delivery Worker Pool
  ↓
External Webhook Target
  ↓
Success / Retry / Failed
```

Redis is used for rate limiting and short-lived idempotency helpers. Payload bodies stay in PostgreSQL; Kafka carries lightweight references.

## Tech stack

- Go
- PostgreSQL
- Redis
- Kafka
- Prometheus / Grafana
- Docker Compose
- Nuxt 4 dashboard

## Quick start

```bash
cp .env.example .env
docker compose up -d --build
```

Services:

| Service    | URL                    |
|------------|------------------------|
| API        | http://localhost:8080  |
| Dashboard  | http://localhost:3000  |
| Prometheus | http://localhost:9090  |
| Grafana    | http://localhost:3001  |

Default Grafana login: `admin` / `admin`.

Bootstrap API key (compose):

```text
hf_dev_bootstrap_key_change_me
```

### Create an endpoint

```bash
curl -s -X POST http://localhost:8080/api/v1/endpoints \
  -H "Authorization: Bearer hf_dev_bootstrap_key_change_me" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "demo",
    "target_url": "http://api:8080/demo/webhook?mode=200"
  }'
```

Save the returned `public_id` and `secret`.

### Send a webhook

```bash
# Replace PUBLIC_ID / SECRET / TIMESTAMP / SIGNATURE
curl -s -X POST "http://localhost:8080/webhooks/PUBLIC_ID" \
  -H "Content-Type: application/json" \
  -H "X-Event-ID: evt_001" \
  -H "X-Webhook-Timestamp: TIMESTAMP" \
  -H "X-Webhook-Signature: SIGNATURE" \
  -d '{"hello":"hookforge"}'
```

Signature algorithm:

```text
HMAC-SHA256(secret, timestamp + "." + raw_body)
```

## Configuration

See [`.env.example`](.env.example). Important variables:

- `DATABASE_URL`
- `REDIS_URL`
- `KAFKA_BROKERS` / `KAFKA_TOPIC`
- `WORKER_COUNT`
- `HTTP_TIMEOUT`
- `MAX_PAYLOAD_SIZE`
- `MAX_RETRIES`
- `RATE_LIMIT`
- `ALLOW_PRIVATE_TARGETS` (dev/demo only)

## Retry behavior

Default schedule:

1. 10s
2. 30s
3. 2m
4. 5m
5. 15m

Retries happen for `408`, `429`, `5xx`, timeouts, and connection errors. Ordinary `4xx` responses are not retried.

## Security

- API keys stored hashed (SHA-256)
- Constant-time HMAC comparison
- Timestamp tolerance window against replay
- Request size limits and timeouts
- Outbound SSRF protections (localhost / private ranges / metadata IPs)

## Observability

- `GET /metrics` — Prometheus metrics
- Grafana dashboard: **HookForge Overview**
- JSON logs with `request_id`, `event_id`, and delivery timing

## Testing

```bash
make test
make test-race
make vet
```

## Project structure

```text
backend/
  cmd/api
  cmd/worker
  cmd/seed
  internal/
  migrations/
  api/openapi.yaml
frontend/
deploy/
  prometheus/
  grafana/
docker-compose.yml
```

## License

MIT
