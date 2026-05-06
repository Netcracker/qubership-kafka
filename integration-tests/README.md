This section describes using integration tests based on [Robot Framework](http://robotframework.org).

Base docker image for Integration Tests: https://github.com/Netcracker/qubership-docker-integration-tests

# Run Tests Locally

For the full local setup (Docker Compose stack, build instructions, and test commands) see [How to run demo](../demo/README.md).

# Test Suites

| Suite | Tags | What it covers |
|---|---|---|
| `kafka/crud/topic_tests.robot` | `kafka_crud` | Topic creation, partition change, deletion |
| `kafka/cp_tests/consumer_producer_tests.robot` | `kafka_consumer_producer` | Produce and consume a message end-to-end |
| `kafka/backup/backup.robot` | `backup` | Full/granular backup, restore, eviction, unauthorized access |
| `kafka/acl_backup/acl_backup.robot` | `backup` | ACL backup and restore |
| `kafka/acl/acl_tests.robot` | — | ACL CRUD operations |
| `kafka/ha/ha_tests.robot` | — | High-availability scenarios |

# Mandatory Test Coverage

Every change to the repository **must** be tested with at minimum:

- **Smoke tests** — topic CRUD and consumer/producer (`-i kafka_crud -i kafka_consumer_producer`)
- **Backup tests** — full backup/restore cycle (`-i backup`)

Run from the `demo/` directory:

```bash
# Smoke tests
docker-compose exec integration-tests robot -i kafka_crud -i kafka_consumer_producer ./tests

# Backup tests
docker-compose exec integration-tests robot -i backup ./tests

# Full suite
docker-compose exec integration-tests robot ./tests
```
