# 补充压测记录

日期：2026-06-05  
目标：`https://zhu-ying.online`

## 记录范围

- WebSocket 长连接递增压测补充结果
- 单聊实时消息链路更高并发压测结果
- 混合连接与消息吞吐压测尝试记录

## 一、WebSocket 长连接递增压测补充

### 1. WebSocket 800 并发连接压测

执行命令：

```bash
k6 run loadtest/scenarios/ws-connect.js \
  -e BASE_URL=https://zhu-ying.online \
  -e WS_ORIGIN=https://zhu-ying.online \
  -e USERS_FILE=loadtest/data/users.json \
  -e TARGET_VUS=800 \
  -e RAMP_UP=90s \
  -e HOLD=60s \
  -e RAMP_DOWN=30s \
  -e SOCKET_LIFETIME_SECONDS=75
```

关键结果：

- `ws_connect_failure_total = 0`
- `ws_connect_duration_ms p95 = 202ms`
- `ws_connect_success_total = 1693`
- `http_req_failed = 0%`

结论：

- 在当前单机生产环境下，`800` 并发 WebSocket 长连接可稳定建立并维持。
- 建连延迟未出现异常抬升，仍处于较健康区间。

### 2. WebSocket 1000 并发连接压测

执行命令：

```bash
k6 run loadtest/scenarios/ws-connect.js \
  -e BASE_URL=https://zhu-ying.online \
  -e WS_ORIGIN=https://zhu-ying.online \
  -e USERS_FILE=loadtest/data/users.json \
  -e TARGET_VUS=1000 \
  -e RAMP_UP=60s \
  -e HOLD=30s \
  -e RAMP_DOWN=20s \
  -e SOCKET_LIFETIME_SECONDS=75
```

关键结果：

- `ws_connect_failure_total = 0`
- `ws_connect_duration_ms p95 = 242ms`
- `ws_connect_success_total = 1432`
- `http_req_failed = 0%`

结论：

- 在当前单机生产环境下，`1000` 并发 WebSocket 长连接仍可稳定建立并维持。
- 与 `800` 并发相比，建连延迟有所上升，但仍未出现失败。

### 3. 当前 WebSocket 长连接结论

- 当前已验证系统可稳定支撑 `1000` 并发 WebSocket 长连接。
- 从 `50 / 80 / 120 / 160 / 240 / 480 / 800 / 1000` 这些档位来看，WebSocket 长连接建连本身不是当前系统首要瓶颈。

## 二、单聊实时消息链路补充压测

### 1. 单聊实时链路 16 VUs 压测

执行命令：

```bash
k6 run loadtest/scenarios/ws-chat-single.js \
  -e BASE_URL=https://zhu-ying.online \
  -e WS_ORIGIN=https://zhu-ying.online \
  -e USERS_FILE=loadtest/data/users.chat.json \
  -e TARGET_VUS=16 \
  -e RAMP_UP=45s \
  -e HOLD=90s \
  -e RAMP_DOWN=30s \
  -e SOCKET_LIFETIME_MS=20000
```

关键结果：

- `ws_connect_failure_total = 0`
- `ws_error_events_total = 0`
- `ws_sync_required_total = 0`
- `ws_message_sent_ack_total = 1419`
- `ws_message_created_total = 1423`
- `ws_message_read_total = 1419`
- `ws_message_round_trip_ms p95 = 1403.6ms`

结论：

- `16 VUs` 下单聊实时链路仍可跑通，但消息闭环时延已经明显退化。
- 由于 `p95` 已超过 `1000ms` 阈值，因此 `16 VUs` 不适合作为当前稳定值。
- 相比之下，之前记录的 `8 VUs`、`p95 = 424ms` 更适合作为当前实例下的稳定参考值。

## 三、混合连接与消息吞吐压测尝试

### 1. 压测目标

目标是验证：

- 在指定总连接数下，每秒处理多少条消息
- 消息从 `message.send` 到 `message.read` 的闭环延迟是多少

### 2. 脚本调整

- 重写了 `loadtest/scenarios/mixed-dev.js`
- 将脚本改为：
  - `idle_ws` 负责撑总连接数
  - `ws_chat_flow` 负责真实发送消息
  - 在 `setup()` 中预登录，避免压测阶段混入大量实时登录请求

### 3. Smoke Test

执行参数：

- `TOTAL_CONNECTIONS=20`
- `CHAT_VUS=4`
- `CHAT_MESSAGES_PER_SESSION=2`

关键结果：

- `mixed_chat_flow_failure_total = 0`
- `ws_message_sent_total = 372`
- `ws_message_read_total = 372`
- `ws_message_round_trip_ms p95 = 494.75ms`
- `ws_sync_required_total = 0`

结论：

- 脚本逻辑已打通
- 预登录方案有效
- 消息吞吐与延迟指标可以正常产出

### 4. 800 连接正式消息吞吐压测尝试

执行参数：

- `TOTAL_CONNECTIONS=800`
- `CHAT_VUS=40`
- `CHAT_MESSAGES_PER_SESSION=5`

结果说明：

- 正式压测阶段未形成最终可用结论
- 主要问题不是业务链路失败，而是：
  - `setup()` 预登录账号数量过多，准备成本较高
  - 压测口径仍需进一步优化后再用于正式吞吐结论

结论：

- 这轮结果不纳入正式性能结论
- 仅作为脚本探索与后续压测方案演进记录

## 四、当前综合结论

- WebSocket 空闲长连接：已验证稳定支撑 `1000` 并发连接
- 单聊实时消息链路：`8 VUs` 可视为稳定值，`16 VUs` 已出现明显时延退化
- 混合连接下的消息吞吐压测脚本已基本成型，但正式结论仍需后续进一步优化后再采纳

## 相关结果文件

- `/tmp/im-k6-results/ws-connect-800-final.json`
- `/tmp/im-k6-results/ws-connect-1000-final.json`
- `/tmp/im-k6-results/mixed-connections-smoke-2.json`
