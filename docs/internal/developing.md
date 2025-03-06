# Developer Guide

## Dev-Kit

### Operator Framework Terminal

The common developer case is changing CRD structure. After the changes, the all related Golang files should be
re-generated. To simplify work with Golang Operator Framework, the dev-kit has specific terminal.

To run it, need to go to `kafka-service-operator/dev-kit/` directory and run `terminal.sh` file.

The `terminal.sh` file runs docker-compose with operator-framework image and mount to the project.

In the running terminal the developer can go to `kafka-service-operator` directory and work with it.

For example, to re-generate code and CRDs need to run the following commands:

```sh
make generate
make manifests
```

More information can be found in [Operator guide](/docs/internal/operator-guide.md).