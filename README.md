# IM System

`IM System` 是一个基于 `Go + Gin + React + WebSocket + MySQL + Redis` 的单体即时通讯项目。

当前版本采用单机架构，覆盖实时消息链路、会话游标设计、离线补推、监控、压测和部署相关能力。

## 总体架构

[![IM System Overall Architecture](./docs/images/image.png)](./docs/images/image.png)

## 能力范围

- 支持注册登录、好友关系、单聊、群聊、头像上传、在线状态
- WebSocket 实时消息链路和 HTTP 补洞链路分离
- 用 `msg_id` 保证发送幂等，用会话内 `seq` 统一历史、同步、ACK、已读
- 基于 `conversation_members` 建模成员可见性、送达游标、已读游标
- 提供 Prometheus + Grafana 监控和 k6 压测脚本
- 提供 `Docker Compose + Nginx + MySQL + Redis` 的单机部署方案

## 技术栈

- Backend: Go, Gin, GORM, JWT, Gorilla WebSocket
- Frontend: React, TypeScript, Vite
- Storage: MySQL, Redis
- Deploy: Docker Compose, Nginx, Let's Encrypt
- Observability: Prometheus, Grafana, mysqld-exporter, redis-exporter
- Load Test: k6

## 文档索引

项目级文档：

- [docs/architecture.md](./docs/architecture.md)
- [docs/database-design.md](./docs/database-design.md)
- [docs/websocket-flow.md](./docs/websocket-flow.md)
- [docs/performance-test.md](./docs/performance-test.md)
- [docs/monitoring.md](./docs/monitoring.md)

后端协议文档：

- [backend/docs/ws-protocol.md](./backend/docs/ws-protocol.md)
- [backend/docs/error-codes.md](./backend/docs/error-codes.md)

## 已实现能力

- 认证
  注册、登录、刷新令牌、登出
- 关系链
  好友申请、好友列表、删除好友
- 会话
  单聊会话、群聊创建、改名、邀请、移除成员、退出、解散
- 消息
  WebSocket 实时消息、离线补推、历史消息、按 `seq` 补洞同步
- 用户
  个人信息、头像上传、在线状态
- 工程化
  Swagger、监控面板、告警规则、压测脚本、部署模板

## 仓库结构

- `backend/`
  后端服务、业务逻辑、Swagger、协议文档
- `frontend/`
  前端应用
- `deploy/`
  生产部署入口和配置模板
- `loadtest/`
  k6 压测脚本和结果
- `monitoring/`
  Prometheus 和 Grafana 监控资源
- `docs/`
  项目级设计文档与架构说明

## 相关入口

- 部署说明见 [deploy/README.md](./deploy/README.md)
- 压测汇总见 [docs/performance-test.md](./docs/performance-test.md)
- 监控资源在 `monitoring/`

## 当前架构边界

当前版本采用单体架构，边界如下：

- WebSocket Hub 仍是单进程内存模型
- MySQL 和 Redis 当前按单机部署设计
- 会话列表摘要聚合是当前最先需要继续优化的链路

当规模继续增长时，可进一步演进到多实例部署和更明确的读写分离模型。
