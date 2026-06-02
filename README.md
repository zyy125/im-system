# IM System

一个基于 `Go + Gin + React + WebSocket + MySQL + Redis` 的单体即时通讯项目，支持注册登录、单聊实时消息、群聊、头像上传、在线状态、监控与压测。

## Features

- 用户注册、登录、刷新令牌、登出
- 单聊实时消息与已读/送达状态
- 群聊创建、改名、邀请、移除成员、退出、解散
- 用户头像上传与静态资源托管
- 在线状态与好友列表
- Prometheus + Grafana 监控
- k6 登录、会话列表、WebSocket 建连、单聊与混合流量压测

## Stack

- Backend: Go, Gin, GORM, JWT, WebSocket
- Frontend: React, TypeScript, Vite
- Storage: MySQL, Redis
- Deploy: Docker Compose, Nginx, Let's Encrypt
- Observability: Prometheus, Grafana, mysqld-exporter, redis_exporter
- Load Test: k6

## Project Layout

- `backend/`: 后端业务代码与开发配置
- `frontend/`: 前端应用代码
- `deploy/`: 生产部署入口与生产配置模板
- `monitoring/`: Grafana dashboards、本地监控栈与监控资源
- `loadtest/`: k6 压测脚本、模板数据与压测说明

## Deploy

生产部署入口在 [deploy/README.md](/home/zhuyin/im-system/deploy/README.md:1)。

当前部署思路：

- 单机 Linux 服务器
- `Docker Compose + Nginx + MySQL + Redis`
- 前后端同域部署
- HTTPS 由项目内 `gateway` 容器处理
- Prometheus / Grafana 仅监听 `127.0.0.1`，通过 SSH 隧道访问

## Monitoring

监控资源位于：

- `deploy/prometheus.prod.yml`
- `deploy/prometheus.rules.prod.yml`
- `monitoring/grafana/dashboards/`

线上推荐通过 SSH 隧道访问：

- Grafana: `http://127.0.0.1:3000`
- Prometheus: `http://127.0.0.1:9091`

## Load Testing

压测脚本位于 `loadtest/`，详细说明见 [loadtest/README.md](/home/zhuyin/im-system/loadtest/README.md:1)。

常见场景：

- `http-login.js`: 登录接口压力
- `http-conversation-list.js`: 会话列表读取压力
- `ws-connect.js`: WebSocket 在线连接压力
- `ws-chat-single.js`: 单聊实时消息链路
- `mixed-dev.js`: 长连接 + HTTP + 聊天混合流量

## Notes

- 仓库保留部署模板与压测模板，不保留真实生产配置和真实压测账号数据。
- 开发环境通常使用 `go run` 读取 `backend/config/config.yaml`。
- 生产环境通过 `deploy/docker-compose.prod.yml` 启动，并读取服务器本地的：
  - `deploy/.env`
  - `deploy/backend.config.prod.yaml`
