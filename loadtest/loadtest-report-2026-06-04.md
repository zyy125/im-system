# 压测说明

日期：2026-06-04  
目标：`https://zhu-ying.online`

## 压测范围

- WebSocket 长连接递增压测
- WebSocket 高并发连接稳定性验证

## 压测目的

- 在已有业务链路压测基础上，继续验证系统在更高并发下的 WebSocket 长连接承载能力
- 找出当前环境下“已验证的最大稳定并发连接数”
- 观察在高连接数下建连延迟是否出现明显退化

## 执行命令与参数

### 1. WebSocket 240 并发连接压测

```bash
k6 run loadtest/scenarios/ws-connect.js \
  -e BASE_URL=https://zhu-ying.online \
  -e WS_ORIGIN=https://zhu-ying.online \
  -e USERS_FILE=loadtest/data/users.json \
  -e TARGET_VUS=240 \
  -e RAMP_UP=90s \
  -e HOLD=2m \
  -e RAMP_DOWN=30s \
  -e SOCKET_LIFETIME_SECONDS=75
```

### 2. WebSocket 480 并发连接压测

```bash
k6 run loadtest/scenarios/ws-connect.js \
  -e BASE_URL=https://zhu-ying.online \
  -e WS_ORIGIN=https://zhu-ying.online \
  -e USERS_FILE=loadtest/data/users.json \
  -e TARGET_VUS=480 \
  -e RAMP_UP=90s \
  -e HOLD=60s \
  -e RAMP_DOWN=30s \
  -e SOCKET_LIFETIME_SECONDS=75
```

### 3. WebSocket 640 并发连接压测

```bash
k6 run loadtest/scenarios/ws-connect.js \
  -e BASE_URL=https://zhu-ying.online \
  -e WS_ORIGIN=https://zhu-ying.online \
  -e USERS_FILE=loadtest/data/users.json \
  -e TARGET_VUS=640 \
  -e RAMP_UP=60s \
  -e HOLD=30s \
  -e RAMP_DOWN=20s \
  -e SOCKET_LIFETIME_SECONDS=75
```

## 压测结果

| 场景 | 压测参数 | 结果 |
|---|---:|---|
| WebSocket 240 并发连接 | `TARGET_VUS=240` | `ws_connect_failure_total = 0`，`ws_connect_duration_ms P95 = 222.75ms` |
| WebSocket 480 并发连接 | `TARGET_VUS=480` | `ws_connect_failure_total = 0`，`ws_connect_duration_ms P95 = 213.3ms` |
| WebSocket 640 并发连接 | `TARGET_VUS=640` | `ws_connect_failure_total = 0`，`ws_connect_duration_ms P95 = 613.2ms` |

## 结果分析

- `240` 并发连接下，WebSocket 建连保持稳定，未出现失败，建连延迟仍处于较低水平。
- `480` 并发连接下，系统仍然可以稳定建立连接，且建连延迟未明显恶化。
- `640` 并发连接下，系统依然没有出现建连失败，但建连 `P95` 已从之前 `200ms` 左右上升到 `613.2ms`，说明系统虽然还能承载，但已经出现明显退化趋势。

## 结论

- 当前单机生产环境下，已验证系统可稳定支撑至少 `640` 个并发 WebSocket 长连接。
- 在 `640` 并发连接下，建连失败数仍为 `0`，但建连延迟已出现明显上升。
- 在当前这轮记录范围内，`640` 可视为当时的已验证上界；是否继续上探取决于后续更高档位测试结果。

## 建议表述

- 基于 k6 对 WebSocket 长连接进行递增压测，记录到单机生产环境在 `640` 并发连接下建连失败数为 `0`；在高压场景下建连 `P95` 约为 `613ms`。
