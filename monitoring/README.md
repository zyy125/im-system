# Monitoring Stack

This directory contains a local monitoring stack for load testing the IM backend.

## What It Runs

- Prometheus on `http://localhost:9091`
- Grafana on `http://localhost:3000`
- Redis exporter on `http://localhost:9121/metrics`
- MySQL exporter on `http://localhost:9104/metrics`

Grafana credentials:

- Username: `admin`
- Password: `admin123`

## Important Backend Setting

Prometheus runs in Docker, so your backend metrics endpoint must be reachable from containers.

The backend default monitoring address is `127.0.0.1:9090`, which is only reachable from the host itself.
Before starting the backend for this stack, make the monitoring server listen on all interfaces:

```bash
cd backend
IM_MONITOR_ADDR=:9090 go run ./cmd
```

If you prefer config-file changes, set:

```yaml
monitor:
  addr: ":9090"
  enable_metrics: true
  enable_debug_hub: false
  enable_pprof: false
```

Only `/metrics` is required for this local Prometheus stack. Keep `/debug/hub` and `pprof` disabled unless you are actively diagnosing Hub state or Go runtime performance.

## Start

From the repo root:

```bash
docker compose -f monitoring/docker-compose.yml up -d
```

If your Docker installation still uses the standalone Compose binary, use:

```bash
docker-compose -f monitoring/docker-compose.yml up -d
```

## Verify

1. Open `http://localhost:9091/targets` and confirm `im-backend` is `UP`.
2. Confirm `redis-exporter` is also `UP`.
3. Confirm `mysql-exporter` is also `UP`.
4. Open `http://localhost:3000`.
5. Open the provisioned dashboards:
   - `IM System / IM System Overview`
   - `IM System / IM Redis Overview`
   - `IM System / IM MySQL Overview`
   - `IM System / IM Redis Business Overview`

## Stop

```bash
docker compose -f monitoring/docker-compose.yml down
```

Or:

```bash
docker-compose -f monitoring/docker-compose.yml down
```

To remove local monitoring data too:

```bash
docker compose -f monitoring/docker-compose.yml down -v
```

Or:

```bash
docker-compose -f monitoring/docker-compose.yml down -v
```

## What To Watch During Load Tests

- `HTTP Request Rate`
- `HTTP P95 Latency`
- `Hub Connections`
- `Hub Queue Lengths`
- `Hub Pressure Signals`
- `DB Connection Pool`
- `Redis Up`
- `Redis Memory`
- `Redis Throughput`
- `Keyspace Hits vs Misses`
- `Redis Pressure Signals`
- `MySQL Up`
- `MySQL Throughput`
- `Threads and Connections`
- `InnoDB Buffer Pool Read Pressure`
- `Redis Ops by Module`
- `Redis P95 Duration by Module/Op`

## MySQL Exporter Notes

The local stack now configures `mysqld_exporter` with:

- `MYSQL_EXPORTER_ADDRESS`
- `MYSQL_EXPORTER_USER`
- `MYSQL_EXPORTER_PASSWORD`

If your local MySQL is not `root:123456@127.0.0.1:3306`, override them when starting the stack:

```bash
MYSQL_EXPORTER_ADDRESS=host.docker.internal:3306 \
MYSQL_EXPORTER_USER=root \
MYSQL_EXPORTER_PASSWORD=your-password \
docker compose -f monitoring/docker-compose.yml up -d
```
