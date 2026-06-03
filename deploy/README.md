# IM System 部署方案

这套方案面向一台 Linux 服务器首发上线，采用 `Docker Compose + Nginx + MySQL + Redis + Go Backend + React Frontend`。

## 架构

- `gateway`：对外唯一入口，负责托管前端静态资源，并把 `/api`、`/api/v1/ws/`、`/uploads/avatars/` 反向代理到后端。
- `backend`：Gin 单体服务，内部同时暴露业务端口 `8080` 和监控端口 `9090`。
- `mysql`：持久化业务数据。
- `redis`：在线状态、刷新会话、消息序列状态。
- `prometheus`、`grafana`：可选监控组件，默认通过 Compose profile 启动。
- `redis-exporter`：把 Redis 运行指标暴露给 Prometheus。
- `mysql-exporter`：把 MySQL 实例指标暴露给 Prometheus。

## 上线前准备

1. 准备一台 Linux 服务器，建议 `2C4G` 起步。
2. 安装 Docker Engine 和 Docker Compose 插件。
3. 准备好域名，并确认已经解析到当前服务器。
4. 开放公网端口：
   - `80`：站点入口
   - `443`：HTTPS 入口
5. 不建议对公网暴露：
   - `3306`
   - `6379`
   - `9090`
   - `9091`
   - `3000`

## 首次部署

1. 拉取代码后进入 `deploy/` 目录。

2. 复制环境变量模板：

```bash
cd deploy
cp .env.example .env
```

3. 复制后端生产配置模板：

```bash
cp backend.config.prod.yaml.example backend.config.prod.yaml
```

4. 复制 MySQL 与 Redis 生产配置模板：

```bash
cp mysql.prod.cnf.example mysql.prod.cnf
cp redis.prod.conf.example redis.prod.conf
```

5. 修改 `.env` 里的关键配置：
   - `MYSQL_ROOT_PASSWORD`
   - `MYSQL_PASSWORD`
   - `IM_JWT_SECRET`
   - `IM_MYSQL_DSN`
   - `GRAFANA_ADMIN_PASSWORD`
   - `MYSQL_EXPORTER_PASSWORD`

6. 修改 `backend.config.prod.yaml` 里的关键配置：
   - `http.allowed_origins`
   - `ws.allowed_origins`
   - `redis.addr`
   - `mysql.max_open_conns`
   - `redis.pool_size`

7. 按机器规格调整 `mysql.prod.cnf` 和 `redis.prod.conf`：
   - `mysql.prod.cnf` 负责 MySQL 服务端参数，例如连接上限、InnoDB buffer pool、慢查询日志
   - `redis.prod.conf` 负责 Redis 服务端参数，例如持久化、最大内存、淘汰策略

8. 为 `mysqld-exporter` 预留监控账号：
   - `.env` 中的 `MYSQL_EXPORTER_USER`
   - `.env` 中的 `MYSQL_EXPORTER_PASSWORD`

## 配置边界

- `backend.config.prod.yaml`：后端应用的非敏感配置，例如端口、监控开关、允许来源、MySQL/Redis 客户端连接池参数
- `.env`：敏感值与环境覆盖项，例如 JWT 密钥、MySQL DSN、Redis 密码
- `mysql.prod.cnf`：MySQL 服务端实例参数
- `redis.prod.conf`：Redis 服务端实例参数

## HTTPS 方案

推荐直接使用 `Let's Encrypt + Nginx`。

1. 先确保 `80` 端口没有被旧服务占用。

2. 使用 `certbot` 签发证书：

```bash
sudo certbot certonly --standalone -d your-domain.example -m your-email@example.com --agree-tos --no-eff-email
```

3. `deploy/nginx.conf` 应为正式 HTTPS 版本：
   - `80` 端口只做跳转到 HTTPS
   - `443` 端口提供正式站点

4. 叠加 TLS 覆盖文件启动：

```bash
docker compose \
  --env-file .env \
  -f docker-compose.prod.yml \
  -f docker-compose.prod.tls.yml \
  up -d --build
```

`deploy/docker-compose.prod.tls.yml` 只负责：

- `443:443`
- 把宿主机 `/etc/letsencrypt` 只读挂进 `gateway` 容器

## 启动顺序

建议按下面顺序启动：

1. 启动核心服务：

```bash
docker compose \
  --env-file .env \
  -f docker-compose.prod.yml \
  -f docker-compose.prod.tls.yml \
  up -d --build
```

2. 验证：
   - `https://你的域名/`
   - `https://你的域名/healthz`
   - 注册 / 登录 / WebSocket / 聊天链路

3. 如果要启动监控，再追加 profile：

```bash
docker compose \
  --env-file .env \
  -f docker-compose.prod.yml \
  -f docker-compose.prod.tls.yml \
  --profile monitoring up -d
```

## 验证

- 首页：`https://你的域名/`
- 后端健康检查：`https://你的域名/healthz`
- Prometheus：通过 SSH 隧道访问 `http://127.0.0.1:9091`
- Grafana：通过 SSH 隧道访问 `http://127.0.0.1:3000`

Grafana 账号密码从 `.env` 读取。
建议为 `mysqld_exporter` 单独准备一个只读监控账号，并把它写到 `.env` 的：
- `MYSQL_EXPORTER_ADDRESS`
- `MYSQL_EXPORTER_USER`
- `MYSQL_EXPORTER_PASSWORD`

SSH 隧道示例：

```bash
ssh -L 3000:127.0.0.1:3000 -L 9091:127.0.0.1:9091 your-user@your-server
```

## 升级流程

```bash
git pull
cd deploy
docker compose \
  --env-file .env \
  -f docker-compose.prod.yml \
  -f docker-compose.prod.tls.yml \
  up -d --build
```

## 回滚建议

- 保留上一版镜像标签，不要只用 `latest`
- 升级前备份 MySQL
- 如果只是前后端代码回滚，优先回滚 `gateway` 和 `backend`

## 监控建议

- 线上建议只让 `prometheus`、`grafana` 监听 `127.0.0.1`
- `debug/hub` 建议默认关闭，需要排查 WebSocket Hub 状态时再临时打开
- `pprof` 默认关闭，避免直接暴露性能分析接口
- 如果 `gateway` 业务已通但容器状态显示 `unhealthy`，优先检查 healthcheck 是否和 HTTPS / 证书校验策略匹配
- 通过现有仪表盘重点看：
  - HTTP QPS
  - HTTP P95 延迟
  - WebSocket 连接数
  - Hub 队列积压
  - DB 连接池状态
  - MySQL QPS、slow queries、threads connected/running、buffer pool read pressure、row lock waits
  - Redis 内存、连接数、吞吐、keyspace hit/miss、拒绝连接
  - Redis 应用内业务操作速率、错误率和 P95 耗时
- Prometheus 规则文件已经包含基础告警：
  - `IMBackendDown`
  - `RedisExporterDown`
  - `MySQLExporterDown`
  - `MySQLDown`
  - `HighHTTPP95Latency`
  - `HubQueuePressure`
  - `DBConnectionPoolWaits`
  - `RedisRejectedConnections`
  - `MySQLTooManyConnections`
  - `MySQLSlowQueriesIncreasing`
  - `MySQLRowLockWaits`
  - `RedisAppErrors`

## 这套方案的边界

这是一套适合当前阶段的单机部署方案，优点是快、稳、维护成本低。后续如果在线人数明显上涨，再演进到：

- 前置负载均衡
- 多实例后端
- Redis 独立高可用
- MySQL 主从或托管数据库
- 对象存储接管头像资源
