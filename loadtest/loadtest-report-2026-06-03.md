# 压测说明

日期：2026-06-03  
目标：`https://zhu-ying.online`

## 压测范围

- 登录链路
- 会话列表链路
- WebSocket 建连链路
- 单聊实时消息链路
- WebSocket 递增连接压测

## 压测结果

| 场景 | 压测参数 | 结果 |
|---|---:|---|
| 登录链路 | `20 VUs / 1m` | `P95 193.4ms`，`17.6 req/s`，`0%` 错误 |
| 会话列表链路 | `20 VUs / 1m` | `P95 421.4ms`，`40.1 req/s`，`max 609.44ms` |
| WebSocket 建连基线 | `50 target VUs` | `failure=0`，`P95 238.5ms` |
| WebSocket 递增压测 | `80 / 120 / 160` | 全部通过，`failure=0`，`P95 ~214 / 213.65 / 213ms` |
| 单聊实时消息链路 | `8 VUs / 3m` | `message.send -> message.read P95 424ms`，`6.6 msg/s`，`sync.required=0` |

## 执行命令与参数明细

### 1. 登录链路

执行命令：

```bash
k6 run loadtest/scenarios/http-login.js \
  -e BASE_URL=https://zhu-ying.online \
  -e USERS_FILE=loadtest/data/users.json \
  -e VUS=20 \
  -e DURATION=1m
```

关键参数：

- `VUS=20`
- `DURATION=1m`
- 脚本每轮包含一次登录和一次 `sleep(1)`

关键结果：

- `http_reqs = 1045`
- `http_req_duration avg = 153.6ms`
- `http_req_duration p95 = 193.4ms`
- `http_req_failed = 0%`

### 2. 会话列表链路

执行命令：

```bash
k6 run loadtest/scenarios/http-conversation-list.js \
  -e BASE_URL=https://zhu-ying.online \
  -e USERS_FILE=loadtest/data/users.json \
  -e VUS=20 \
  -e DURATION=1m
```

关键参数：

- `VUS=20`
- `DURATION=1m`
- 每轮包含：登录、`/api/v1/conversations`、`/api/v1/conversations/groups`、`sleep(1)`

关键结果：

- `http_reqs = 2403`
- `http_req_duration avg = 174.4ms`
- `http_req_duration p95 = 421.4ms`
- `http_req_duration max = 609.44ms`
- `http_req_failed = 0%`

### 3. WebSocket 建连基线

执行命令：

```bash
k6 run loadtest/scenarios/ws-connect.js \
  -e BASE_URL=https://zhu-ying.online \
  -e WS_ORIGIN=https://zhu-ying.online \
  -e USERS_FILE=loadtest/data/users.json \
  -e TARGET_VUS=50 \
  -e RAMP_UP=1m \
  -e HOLD=2m \
  -e RAMP_DOWN=30s \
  -e SOCKET_LIFETIME_SECONDS=45
```

关键参数：

- `TARGET_VUS=50`
- `RAMP_UP=1m`
- `HOLD=2m`
- `RAMP_DOWN=30s`
- `SOCKET_LIFETIME_SECONDS=45`

关键结果：

- `ws_connect_success_total = 207`
- `ws_connect_failure_total = 0`
- `ws_connect_duration_ms p95 = 238.5ms`

### 4. WebSocket 递增连接压测

执行命令模板：

```bash
k6 run loadtest/scenarios/ws-connect.js \
  -e BASE_URL=https://zhu-ying.online \
  -e WS_ORIGIN=https://zhu-ying.online \
  -e USERS_FILE=loadtest/data/users.json \
  -e TARGET_VUS=<80|120|160> \
  -e RAMP_UP=2m \
  -e HOLD=4m \
  -e RAMP_DOWN=1m \
  -e SOCKET_LIFETIME_SECONDS=75
```

关键参数：

- `RAMP_UP=2m`
- `HOLD=4m`
- `RAMP_DOWN=1m`
- `SOCKET_LIFETIME_SECONDS=75`

关键结果：

- `80` 连接：`ws_connect_failure_total = 0`，`p95 = 214ms`
- `120` 连接：`ws_connect_failure_total = 0`，`p95 = 213.65ms`
- `160` 连接：`ws_connect_failure_total = 0`，`p95 = 213ms`

### 5. 单聊实时消息链路

执行命令：

```bash
k6 run loadtest/scenarios/ws-chat-single.js \
  -e BASE_URL=https://zhu-ying.online \
  -e WS_ORIGIN=https://zhu-ying.online \
  -e USERS_FILE=loadtest/data/users.chat.json \
  -e TARGET_VUS=8 \
  -e RAMP_UP=30s \
  -e HOLD=2m \
  -e RAMP_DOWN=30s \
  -e SOCKET_LIFETIME_MS=20000
```

关键参数：

- `TARGET_VUS=8`
- `RAMP_UP=30s`
- `HOLD=2m`
- `RAMP_DOWN=30s`
- `SOCKET_LIFETIME_MS=20000`

关键结果：

- `ws_message_sent_ack_total = 1159`
- `ws_message_created_total = 1159`
- `ws_message_read_total = 1159`
- `ws_message_round_trip_ms avg = 368ms`
- `ws_message_round_trip_ms p95 = 424ms`
- `ws_sync_required_total = 0`
- `ws_error_events_total = 0`

## 结论

- 当前系统最先出现性能退化的是会话列表链路，而不是登录或 WebSocket 建连。
- 单机环境下，WebSocket 空闲长连接已验证可稳定支撑至少 `160` 并发。
- 单聊实时消息链路在本次压力下稳定，未出现 `sync.required` 补洞压力。
- 下一步优化优先级应放在会话列表的摘要聚合、最新消息查询和未读数统计上。

## 说明

- 这次压测重点是业务链路验证与瓶颈定位，不再纳入纯基准测试结果。
- 详细 `k6` 摘要已导出到 `/tmp/im-k6-results/`。
