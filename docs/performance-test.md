# Performance Test

## 1. 文档目的

本文档汇总 `2026-06-03` 至 `2026-06-05` 的线上实例压测记录，描述测试环境、覆盖链路、阶段性结论和适用边界。

## 2. 测试环境

测试目标站点：

- `https://zhu-ying.online`

测试环境如下：

- 被测机器：`4C4G`，系统盘 `40GB`
- 部署方式：单机部署，`MySQL` 和 `Redis` 均为同机容器
- 压测发起位置：同一台机器本机发起
- 真实流量：压测窗口内无其他真实业务流量
- 数据规模：约 `40` 个测试用户，约 `20` 组好友关系

原始结果文件：

- [loadtest/loadtest-report-2026-06-03.md](../loadtest/loadtest-report-2026-06-03.md)
- [loadtest/loadtest-report-2026-06-04.md](../loadtest/loadtest-report-2026-06-04.md)
- [loadtest/loadtest-report-2026-06-05.md](../loadtest/loadtest-report-2026-06-05.md)

## 3. 测试范围

本轮压测累计覆盖以下链路：

- 登录链路
- 会话列表链路
- WebSocket 建连链路
- WebSocket 递增连接压测
- 单聊实时消息链路
- 混合连接与消息吞吐压测尝试

测试目标包括：

- 验证基础 HTTP 接口稳定性
- 识别会话摘要聚合链路是否先成为瓶颈
- 验证长连接与实时消息链路是否稳定

## 4. 测试资产

压测脚本位于 `loadtest/`：

- `scenarios/http-login.js`
- `scenarios/http-conversation-list.js`
- `scenarios/ws-connect.js`
- `scenarios/ws-chat-single.js`
- `scenarios/mixed-dev.js`

配置和用户数据模板位于：

- `loadtest/config.js`
- `loadtest/data/users.example.json`
- `loadtest/data/users.chat.template.json`

压测工具：

- `k6`

## 5. 结果摘要

| 场景 | 压测参数 | 结果 |
| --- | --- | --- |
| 登录链路 | `20 VUs / 1m` | `P95 193.4ms`，`0%` 错误 |
| 会话列表链路 | `20 VUs / 1m` | `P95 421.4ms`，`max 609.44ms` |
| WebSocket 建连基线 | `50 target VUs` | `failure=0`，`P95 238.5ms` |
| WebSocket 递增压测 | `80 / 120 / 160 / 240 / 480 / 640 / 800 / 1000` | 全部通过，`failure=0`；其中 `640` 时 `P95 613.2ms`，`800` 时 `P95 202ms`，`1000` 时 `P95 242ms` |
| 单聊实时消息链路 | `8 VUs / 3m` | `message.send -> message.read P95 424ms`，`sync.required=0`，`0` 错误 |
| 单聊实时消息链路补充 | `16 VUs` | `message.send -> message.read P95 1403.6ms`，链路可跑通但延迟明显退化 |
| 混合连接与消息吞吐 Smoke Test | `TOTAL_CONNECTIONS=20`，`CHAT_VUS=4` | `ws_message_round_trip_ms P95 = 494.75ms`，`sync.required=0` |

## 6. 测试结论

### 6.1 登录链路

登录链路在当前测试条件下保持稳定，未出现明显错误或延迟异常。

### 6.2 会话列表链路

会话列表链路是当前测试中最先出现性能退化的接口。该链路涉及以下聚合读取：

- 当前用户可见会话
- 最新消息
- 未读数
- 单聊对端资料
- 在线状态

### 6.3 WebSocket 建连

从 `50` 基线到 `1000` 并发连接的多轮递增测试中，WebSocket 建连均未出现失败。当前实例条件下，连接建立、注册、上线状态刷新和 bootstrap 链路保持稳定。

### 6.4 单聊实时消息链路

单聊测试验证了以下完整链路：

- `message.send`
- 服务端入库并下发
- 接收端收到 `message.created`
- 接收端回 ACK / 已读
- 发送端收到已读回执

`8 VUs` 场景下，`sync.required=0`，说明在当前压力下 Hub 队列未进入补洞降级状态。  
`16 VUs` 场景下，链路仍可跑通，但 `P95` 已上升到 `1403.6ms`，不再适合作为稳定值。

### 6.5 混合连接与消息吞吐

混合压测脚本已经完成基本打通：

- `setup()` 预登录方案可用
- `idle_ws` 与 `ws_chat_flow` 的职责已拆分
- 消息吞吐和闭环延迟指标可以正常产出

当前正式吞吐结论尚未固定，原因是高连接数场景下的预登录成本和压测口径仍需进一步收敛。

## 7. 阶段性结论

当前汇总结果可以归纳为：

- 会话列表聚合读取仍是当前实例最先出现性能退化的链路。
- WebSocket 空闲长连接在当前实例上已验证到 `1000` 并发连接，建连本身不是当前首要瓶颈。
- 单聊实时消息链路的稳定区间明显低于空闲长连接上限；`8 VUs` 可作为当前稳定值参考，`16 VUs` 已出现显著时延退化。
- 混合连接与消息吞吐脚本已基本成型，但结果仍以探索记录为主，暂不纳入正式性能上界结论。

## 8. 结果边界

本次结果适用于当前实例条件下的链路验证，不直接用于推导系统容量上限。边界如下：

- 测试环境为单机实例
- 压测从同机发起，网络时延因素被弱化
- 当前数据规模较小
- 未叠加复杂真实流量
- 混合吞吐场景仍处于脚本与口径收敛阶段

## 9. 后续测试方向

后续测试可继续覆盖以下场景：

- 群聊消息链路压测
- 混合流量压测口径固化
- 更高并发连接压测
- 注入 Redis / MySQL 抖动，观察 `sync.required` 和恢复路径

## 10. 相关文档

- [architecture.md](./architecture.md)
- [monitoring.md](./monitoring.md)
- [../loadtest/loadtest-report-2026-06-03.md](../loadtest/loadtest-report-2026-06-03.md)
- [../loadtest/loadtest-report-2026-06-04.md](../loadtest/loadtest-report-2026-06-04.md)
- [../loadtest/loadtest-report-2026-06-05.md](../loadtest/loadtest-report-2026-06-05.md)
