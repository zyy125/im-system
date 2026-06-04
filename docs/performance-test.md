# Performance Test

## 1. 文档目的

本文档记录 `2026-06-03` 对 `IM System` 的一次线上实例压测，描述测试环境、测试范围、结果摘要、结论和适用边界。

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

## 3. 测试范围

本次测试覆盖以下链路：

- 登录链路
- 会话列表链路
- WebSocket 建连链路
- WebSocket 递增连接压测
- 单聊实时消息链路

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
| WebSocket 递增压测 | `80 / 120 / 160` | 全部通过，`failure=0`，`P95 ~214 / 213.65 / 213ms` |
| 单聊实时消息链路 | `8 VUs / 3m` | `message.send -> message.read P95 424ms`，`sync.required=0`，`0` 错误 |

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

WebSocket 建连在 50 基线和 80 / 120 / 160 递增连接测试下均未出现失败，说明当前连接建立、注册、上线状态刷新和 bootstrap 链路在该实例上运行稳定。

### 6.4 单聊实时消息链路

单聊测试验证了以下完整链路：

- `message.send`
- 服务端入库并下发
- 接收端收到 `message.created`
- 接收端回 ACK / 已读
- 发送端收到已读回执

本轮结果中 `sync.required=0`，说明在当前压力下 Hub 队列未进入补洞降级状态。

## 7. 结果边界

本次结果适用于当前实例条件下的链路验证，不直接用于推导系统容量上限。边界如下：

- 测试环境为单机实例
- 压测从同机发起，网络时延因素被弱化
- 当前数据规模较小
- 未叠加复杂真实流量

## 8. 后续测试方向

后续测试可继续覆盖以下场景：

- 群聊消息链路压测
- 混合流量压测
- 更高并发连接压测
- 注入 Redis / MySQL 抖动，观察 `sync.required` 和恢复路径

## 9. 相关文档

- [architecture.md](./architecture.md)
- [monitoring.md](./monitoring.md)
- [../loadtest/loadtest-report-2026-06-03.md](../loadtest/loadtest-report-2026-06-03.md)
