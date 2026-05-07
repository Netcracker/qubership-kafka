# Local Testing Environment

Kafka 4.x (KRaft) + backup-daemon + Robot Framework test runner.

## Start

```bash
cd demo
docker-compose up -d
```

To build a service from source instead of pulling the published image:

```bash
docker-compose -f docker-compose.yml -f docker-compose.build.yml up -d --build
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
