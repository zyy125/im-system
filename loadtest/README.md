# k6 Load Tests

This directory contains the first-pass k6 suite for the IM backend.

## Scenarios

- `scenarios/http-login.js`
- `scenarios/http-conversation-list.js`
- `scenarios/ws-connect.js`
- `scenarios/ws-chat-single.js`
- `scenarios/mixed-dev.js`

## Test Data

Create a real user data file from the example:

```bash
cp loadtest/data/users.example.json loadtest/data/users.json
```

The users must already exist in your environment.

For chat-heavy tests, start from the dedicated template instead:

```bash
cp loadtest/data/users.chat.template.json loadtest/data/users.chat.json
```

For `ws-chat-single.js` and `mixed-dev.js`, each user entry also needs peer metadata and a valid `conversation_id`.

Important for chat scenarios:

- Each chat VU needs a dedicated self/peer pair.
- Do not reuse the same username across multiple chat VUs.
- If `TARGET_VUS` or `CHAT_VUS` is greater than the number of configured chat pairs, the test will fail fast.
- A single JSON entry represents one dedicated chat flow, not both directions of the same pair.

## Environment Variables

- `BASE_URL`
  - default: `http://127.0.0.1:8080`
- `WS_URL`
  - default: derived from `BASE_URL`
- `WS_ORIGIN`
  - default: `http://127.0.0.1:4174` or `http://localhost:4174` when `BASE_URL` points at local dev
- `USERS_FILE`
  - default: `./data/users.json`
- `VUS`
- `DURATION`
- `TARGET_VUS`
- `RAMP_UP`
- `HOLD`
- `RAMP_DOWN`
- `SOCKET_LIFETIME_MS`
- `SOCKET_LIFETIME_SECONDS`
- `IDLE_WS_VUS`
- `HTTP_VUS`
- `CHAT_VUS`
- `IDLE_SOCKET_LIFETIME_SECONDS`
- `HTTP_SLEEP_SECONDS`
- `HTTP_POLL_EVERY`
  - default: `3`
- `HTTP_GROUP_POLL_EVERY`
  - default: `6`
- `CHAT_SLEEP_SECONDS`
- `CHAT_MESSAGES_PER_SESSION`
  - default: `5`

## Examples

Login pressure:

```bash
k6 run loadtest/scenarios/http-login.js \
  -e BASE_URL=http://127.0.0.1:8080 \
  -e USERS_FILE=loadtest/data/users.json \
  -e VUS=20 \
  -e DURATION=1m
```

Conversation list:

```bash
k6 run loadtest/scenarios/http-conversation-list.js \
  -e BASE_URL=http://127.0.0.1:8080 \
  -e USERS_FILE=loadtest/data/users.json \
  -e VUS=20 \
  -e DURATION=1m
```

WebSocket connect:

```bash
k6 run loadtest/scenarios/ws-connect.js \
  -e BASE_URL=http://127.0.0.1:8080 \
  -e USERS_FILE=loadtest/data/users.json \
  -e TARGET_VUS=200 \
  -e RAMP_UP=1m \
  -e HOLD=3m \
  -e SOCKET_LIFETIME_SECONDS=45
```

Single-chat flow:

```bash
k6 run loadtest/scenarios/ws-chat-single.js \
  -e BASE_URL=http://127.0.0.1:8080 \
  -e USERS_FILE=loadtest/data/users.json \
  -e TARGET_VUS=20 \
  -e RAMP_UP=30s \
  -e HOLD=2m \
  -e SOCKET_LIFETIME_MS=20000
```

Mixed dev traffic:

```bash
k6 run loadtest/scenarios/mixed-dev.js \
  -e BASE_URL=http://127.0.0.1:8080 \
  -e USERS_FILE=loadtest/data/users.json \
  -e IDLE_WS_VUS=50 \
  -e HTTP_VUS=10 \
  -e CHAT_VUS=10 \
  -e HTTP_POLL_EVERY=3 \
  -e HTTP_GROUP_POLL_EVERY=6 \
  -e CHAT_MESSAGES_PER_SESSION=5 \
  -e RAMP_UP=1m \
  -e HOLD=5m \
  -e RAMP_DOWN=30s
```

## What To Watch

Pair k6 output with your existing dashboards:

- `IM System Overview`
- `IM Redis Overview`
- `IM Redis Business Overview`
- `IM MySQL Overview`

Key signals:

- HTTP request rate / P95 latency
- WebSocket connect failure count and connect latency
- `message.sent` / `message.created` / `message.read` completion counts
- `sync.required` frequency and whether sync can keep up
- Hub connections / queue pressure
- Redis module-level ops and duration
- MySQL queries, threads, slow queries

## Suggested Dev-Env Progression

1. Start with `http-login.js` and `http-conversation-list.js` to get a safe latency baseline.
2. Run `ws-connect.js` to find the sustainable online connection count.
3. Run `ws-chat-single.js` with dedicated chat pairs to validate single-chat real-time correctness under load.
4. Run `mixed-dev.js` for a more realistic development-environment stress mix.

## Recommended Starting Points

Based on a local machine around 20 CPU cores and 7.6 GiB RAM:

1. Safe baseline
   `http-login.js`: `VUS=10 DURATION=1m`
   `http-conversation-list.js`: `VUS=10 DURATION=1m`
   `ws-connect.js`: `TARGET_VUS=50 RAMP_UP=1m HOLD=2m SOCKET_LIFETIME_SECONDS=45`
   `ws-chat-single.js`: `TARGET_VUS=4 RAMP_UP=30s HOLD=90s SOCKET_LIFETIME_MS=20000`
   `mixed-dev.js`: `IDLE_WS_VUS=20 HTTP_VUS=5 CHAT_VUS=4 RAMP_UP=45s HOLD=3m`

2. Medium pressure
   `http-login.js`: `VUS=30 DURATION=2m`
   `http-conversation-list.js`: `VUS=30 DURATION=2m`
   `ws-connect.js`: `TARGET_VUS=120 RAMP_UP=90s HOLD=3m SOCKET_LIFETIME_SECONDS=60`
   `ws-chat-single.js`: `TARGET_VUS=8 RAMP_UP=45s HOLD=2m SOCKET_LIFETIME_MS=25000`
   `mixed-dev.js`: `IDLE_WS_VUS=50 HTTP_VUS=10 CHAT_VUS=8 RAMP_UP=1m HOLD=5m`

3. Aggressive dev-env probe
   `http-login.js`: `VUS=60 DURATION=3m`
   `http-conversation-list.js`: `VUS=60 DURATION=3m`
   `ws-connect.js`: `TARGET_VUS=200 RAMP_UP=2m HOLD=5m SOCKET_LIFETIME_SECONDS=75`
   `ws-chat-single.js`: `TARGET_VUS=10 RAMP_UP=1m HOLD=3m SOCKET_LIFETIME_MS=30000`
   `mixed-dev.js`: `IDLE_WS_VUS=80 HTTP_VUS=15 CHAT_VUS=10 RAMP_UP=90s HOLD=6m`

Stop increasing when any of these appears:

- `sync.required` starts showing up steadily.
- `ws_connect_failure_total` is no longer zero.
- `message.sent` / `message.created` / `message.read` completion counts stop matching expected chat VU volume.
- MySQL slow queries, Redis latency, or hub queue pressure rise sharply.

## Notes

- `ws-chat-single.js` now measures the path `message.send -> message.sent -> peer message.created -> message.read`.
- The backend protocol does not push a downstream `message.delivered` event, so that metric is tracked as client ACK send count only.
- If you see `sync.required`, treat it as a signal that the real-time path is under pressure even if HTTP latency still looks healthy.
- `mixed-dev.js` now reuses login tokens, avoids repeated conversation open calls, and samples HTTP polling instead of issuing list requests on every iteration by default.
