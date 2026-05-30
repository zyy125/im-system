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
3. 开放公网端口：
   - `80`：站点入口
   - `443`：如果你后面接 TLS 终止
4. 不建议对公网暴露：
   - `3306`
   - `6379`
   - `9090`
   - `9091`
   - `3000`

## 首次部署

1. 复制环境变量模板：

```bash
cd deploy
cp .env.example .env
```

2. 复制后端生产配置：

```bash
cp backend.config.prod.yaml.example backend.config.prod.yaml
```

3. 修改 `backend.config.prod.yaml` 里的关键配置：
   - `jwt.secret`
   - `http.allowed_origins`
   - `ws.allowed_origins`
   - `mysql.dsn`

4. 启动核心服务：

```bash
docker compose --env-file .env -f docker-compose.prod.yml up -d --build
```

5. 如果要启动监控：

```bash
docker compose --env-file .env -f docker-compose.prod.yml --profile monitoring up -d
```

## 验证

- 首页：`http://你的域名/`
- 后端健康检查：`http://你的域名/healthz`
- Prometheus：`http://127.0.0.1:9091`
- Grafana：`http://127.0.0.1:3000`

Grafana 账号密码从 `.env` 读取。
建议为 `mysqld_exporter` 单独准备一个只读监控账号，并把它写到 `.env` 的：
- `MYSQL_EXPORTER_ADDRESS`
- `MYSQL_EXPORTER_USER`
- `MYSQL_EXPORTER_PASSWORD`

## 直接接管 80/443

如果服务器上已经停掉旧站点，想让 IM 直接占用 `80/443`，推荐这样做：

1. 停掉旧项目并确认端口已释放：

```bash
cd /ubuntu/projects/my-blog
docker compose down
docker ps
ss -ltnp | grep -E ':80|:443'
```

2. 把项目放到新目录，例如：

```bash
mkdir -p /ubuntu/projects/im-system
```

3. 在 `deploy/` 目录准备生产配置：

```bash
cp .env.example .env
cp backend.config.prod.yaml.example backend.config.prod.yaml
```

4. 修改 `backend.config.prod.yaml`：
   - `jwt.secret` 改成强随机值
   - `http.allowed_origins` 改成你的正式域名
   - `ws.allowed_origins` 改成你的正式域名
   - `mysql.dsn` 中的数据库密码与 `.env` 保持一致

5. 如果你暂时只想先跑通 HTTP，直接启动：

```bash
docker compose --env-file .env -f docker-compose.prod.yml up -d --build
```

6. 如果你要直接复用容器内 Nginx 做 HTTPS：
   - 把 [nginx.tls.conf.example](/home/zhuyin/im-system/deploy/nginx.tls.conf.example:1) 复制为 `deploy/nginx.conf`
   - 把里面的域名和证书路径改成你自己的
   - 把证书目录挂到 `deploy/certs`
   - 再叠加 TLS override 启动：

```bash
docker compose \
  --env-file .env \
  -f docker-compose.prod.yml \
  -f docker-compose.prod.tls.yml.example \
  up -d --build
```

7. 启动后检查：

```bash
docker compose -f docker-compose.prod.yml ps
curl -I http://127.0.0.1/healthz
```

## 升级流程

```bash
git pull
cd deploy
docker compose --env-file .env -f docker-compose.prod.yml up -d --build
```

## 回滚建议

- 保留上一版镜像标签，不要只用 `latest`
- 升级前备份 MySQL
- 如果只是前后端代码回滚，优先回滚 `gateway` 和 `backend`

## 监控建议

- 线上建议只让 `prometheus`、`grafana` 监听 `127.0.0.1`
- `debug/hub` 建议默认关闭，需要排查 WebSocket Hub 状态时再临时打开
- `pprof` 默认关闭，避免直接暴露性能分析接口
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
