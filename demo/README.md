# Local Testing Environment

Kafka 4.x (KRaft) + backup-daemon + AKHQ + monitoring + lag-exporter + Robot Framework test runner.

All services run with `read_only: true` and `tmpfs: [/tmp]` to verify that the
security-hardened images start cleanly under a read-only root filesystem.

## Services

| Service | Port | Purpose |
|---|---|---|
| kafka | 9092 | Kafka broker (KRaft mode) |
| backup-daemon | 8080 | Topic / ACL backup REST API |
| akhq | 8081 | Kafka UI |
| monitoring | — | Telegraf + JMX Prometheus scraper |
| lag-exporter | 9308 | Kafka consumer-lag Prometheus exporter |
| integration-tests | 8090 | Robot Framework web terminal |

## Start

```bash
cd demo
docker-compose up -d
```

To build a service from source instead of pulling the published image:

```bash
docker-compose -f docker-compose.yml -f docker-compose.build.yml up -d --build
```

## Verify all services started

```bash
docker-compose ps
# All services should show status "healthy" or "running".
# lag-exporter metrics:  http://localhost:9308/metrics
# AKHQ UI:               http://localhost:8081
# Backup daemon health:  http://localhost:8080/health/prometheus
```

## Run Tests

```bash
# Smoke tests
docker-compose exec integration-tests robot -i kafka_crud ./tests

# Backup tests
docker-compose exec integration-tests robot -i backup ./tests

# Full suite
docker-compose exec integration-tests robot ./tests
```

## Results

HTML report and XML log are written to `demo/output/` on the host.

## Stop

```bash
docker-compose down -v
```
