# Monitoring

## 1. 文档目的

本文档描述 `IM System` 的监控范围、指标来源、告警对象和当前监控边界。

## 2. 适用范围

本文档覆盖以下内容：

- 当前监控栈
- 监控端口与暴露方式
- 指标分类
- Dashboard 组织方式
- 告警规则范围

本文档不覆盖：

- 每个 Dashboard 的图表细节
- 告警通知集成方式

## 3. 监控栈

当前监控组件包括：

- `Prometheus`
  采集应用和 exporter 指标。
- `Grafana`
  展示 Dashboard。
- `mysqld-exporter`
  采集 MySQL 指标。
- `redis-exporter`
  采集 Redis 指标。

相关资源分为两组：

- `monitoring/`
  面向本地开发、演示或单独拉起监控栈时使用。
- `deploy/*.prod.yml`
  面向生产部署，与 `deploy/docker-compose.prod.yml` 配套使用。

## 4. 监控暴露方式

后端当前使用两个端口：

- `:8080`
  业务 HTTP / WebSocket
- `127.0.0.1:9090`
  监控和调试端口

监控 / 调试端口可暴露以下能力：

- `/metrics`
- `/debug/hub`
- `/debug/pprof/*`

该分离方式用于隔离公网业务入口与调试能力。

## 5. 指标分类

### 5.1 HTTP 指标

应用内 HTTP 指标包括：

- `im_http_requests_in_flight`
- `im_http_requests_total`
- `im_http_request_duration_seconds`

该组指标用于观测：

- 并发请求量
- 路由状态码分布
- 延迟分位数变化

### 5.2 Redis 业务指标

应用额外记录业务侧 Redis 指标：

- `im_redis_operations_total`
- `im_redis_operation_duration_seconds`

该组指标用于区分：

- Redis 实例是否可用
- 应用内部 Redis 操作是否变慢或报错

### 5.3 Hub 指标

Hub 指标包括：

- 在线用户数
- 当前连接数
- register / unregister / forward 队列长度与容量
- `forward_queue_full_total`
- `pending_queue_full_total`
- `send_queue_full_total`
- `sync_required_emitted_total`
- `bootstrap_failed_total`

该组指标用于观测实时链路背压和补洞降级情况。

### 5.4 数据库与实例指标

MySQL 相关监控分为两层：

- 应用进程内连接池状态
- MySQL 实例运行指标

Redis / MySQL 实例层重点指标包括：

- Redis 内存、吞吐、拒绝连接、hit/miss
- MySQL QPS、slow queries、threads connected、row lock waits

## 6. Dashboard 组织

当前 Grafana Dashboard 按主题拆分：

- `im-overview.json`
- `im-mysql-overview.json`
- `im-redis-overview.json`
- `im-redis-business-overview.json`

该拆分方式用于区分：

- 系统总览
- MySQL 专项观测
- Redis 实例观测
- Redis 业务侧观测

## 7. 告警范围

Prometheus 告警规则位于 `monitoring/prometheus/alerts.yml`，当前覆盖的关键对象包括：

- 后端服务可用性
- Redis exporter 可用性
- MySQL exporter 可用性
- MySQL 实例可用性
- HTTP P95 延迟
- Hub 队列压力
- DB 连接池等待
- Redis 拒绝连接
- MySQL 连接占用
- MySQL 慢查询增长
- MySQL 行锁等待
- Redis 业务操作错误

## 8. 监控边界

当前监控体系的边界如下：

- 未集成日志聚合系统
- 未集成分布式 tracing
- 告警规则主要面向单机部署
- 尚未定义业务级 SLO

## 9. 相关文档

- [architecture.md](./architecture.md)
- [performance-test.md](./performance-test.md)
- [../deploy/README.md](../deploy/README.md)
