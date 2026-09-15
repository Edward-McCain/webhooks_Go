#!/usr/bin/env bash
set -euo pipefail

API="${API:-http://localhost:18080}"
KEY="${KEY:-hf_hz97dupC37Crm281dt_6HZxOldO9baEnTMumbXl2nFM}"
MODE="${1:-200}"
EVENT_ID="${2:-evt_$(date +%s)}"
BODY=$(printf '{"hello":"hookforge","mode":"%s"}' "$MODE")

echo "==> Creating endpoint (target: demo mode=$MODE)"
EP_JSON=$(curl -sS -X POST "$API/api/v1/endpoints" \
  -H "Authorization: Bearer $KEY" \
  -H "Content-Type: application/json" \
  -d "{\"name\":\"demo-$MODE\",\"target_url\":\"http://api:8080/demo/webhook?mode=$MODE\",\"rate_limit\":100}")

PUBLIC_ID=$(python3 -c 'import json,sys; print(json.load(sys.stdin)["endpoint"]["public_id"])' <<<"$EP_JSON")
SECRET=$(python3 -c 'import json,sys; print(json.load(sys.stdin)["secret"])' <<<"$EP_JSON")
echo "public_id=$PUBLIC_ID"

TS=$(date +%s)
SIG=$(BODY="$BODY" TS="$TS" SECRET="$SECRET" python3 - <<'PY'
import hmac, hashlib, os
secret = os.environ["SECRET"].encode()
payload = os.environ["BODY"].encode()
ts = os.environ["TS"].encode()
print(hmac.new(secret, ts + b"." + payload, hashlib.sha256).hexdigest())
PY
)

echo "==> Sending signed webhook"
RESP=$(curl -sS -X POST "$API/webhooks/$PUBLIC_ID" \
  -H "Content-Type: application/json" \
  -H "X-Event-ID: $EVENT_ID" \
  -H "X-Event-Type: demo.ping" \
  -H "X-Webhook-Timestamp: $TS" \
  -H "X-Webhook-Signature: $SIG" \
  -d "$BODY")
echo "$RESP"

echo "==> Waiting for worker..."
sleep 4

echo "==> Stats"
curl -sS -H "Authorization: Bearer $KEY" "$API/api/v1/stats"
echo
echo "==> Recent events"
curl -sS -H "Authorization: Bearer $KEY" "$API/api/v1/events?limit=5"
echo
echo "==> Recent deliveries"
curl -sS -H "Authorization: Bearer $KEY" "$API/api/v1/deliveries?limit=5"
echo
