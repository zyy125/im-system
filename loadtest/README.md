# k6 Load Tests

This directory contains the first-pass k6 suite for the IM backend.

## Scenarios

- `scenarios/http-login.js`
- `scenarios/http-conversation-list.js`
- `scenarios/ws-connect.js`
- `scenarios/ws-chat-single.js`

## Test Data

Create a real user data file from the example:

```bash
cp loadtest/data/users.example.json loadtest/data/users.json
```

The users must already exist in your environment.

For `ws-chat-single.js`, each user entry also needs peer metadata and a valid `conversation_id`.

## Environment Variables

- `BASE_URL`
  - default: `http://127.0.0.1:8080`
- `WS_URL`
  - default: derived from `BASE_URL`
- `USERS_FILE`
  - default: `./data/users.json`
- `VUS`
- `DURATION`
- `TARGET_VUS`
- `RAMP_UP`
- `HOLD`
- `RAMP_DOWN`
- `SOCKET_LIFETIME_MS`

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
  -e HOLD=3m
```

Single-chat flow:

```bash
k6 run loadtest/scenarios/ws-chat-single.js \
  -e BASE_URL=http://127.0.0.1:8080 \
  -e USERS_FILE=loadtest/data/users.json \
  -e TARGET_VUS=20 \
  -e RAMP_UP=30s \
  -e HOLD=2m
```

## What To Watch

Pair k6 output with your existing dashboards:

- `IM System Overview`
- `IM Redis Overview`
- `IM Redis Business Overview`
- `IM MySQL Overview`

Key signals:

- HTTP request rate / P95 latency
- Hub connections / queue pressure
- Redis module-level ops and duration
- MySQL queries, threads, slow queries
- `sync.required` frequency
