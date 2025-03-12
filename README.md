[[_TOC_]]

# Kafka Service

## Repository structure

* `./charts` - directory with HELM charts for Kafka and Kafka-Services components.
* `./demo` - directory `docker-compose` to run Kafka with all related services.
* `./docs` - directory with actual documentation for the service.
* `./integration-tests` - directory with Robot Framework test cases.
* `./kafka-service-operator` - directory with operator source code, which is used for running Kafka and Kafka-Services 
  operator.
* `./kafka-service-operator-integration-tests` - directory with HELM Chart for integration-tests and documentation.


## How to start

### Deploy to k8s

#### Pure helm

1. Build operator and integration tests, if you need non-master versions.
2. Prepare kubeconfig on you host machine to work with target cluster.
3. Prepare `sample.yaml` file with deployment parameters, which should contains custom docker images if it is needed.
4. Store `sample.yaml` file in `/charts/helm/kafka` or `/charts/helm/kafka-service` directory.
5. Go to `/charts/helm/kafka` or `/charts/helm/kafka-service` directory.
6. Run the following command if you deploy Kafka only:

     ```sh
     # Run in /charts/helm/kafka directory
     helm install kafke ./ -f sample.yaml -n <TARGET_NAMESPACE>
     ```

7. Run the following command if you deploy Kafka's supplementary services only:

     ```sh
     # Run in /charts/helm/kafka-service directory
     helm install kafka-service ./ -f sample.yaml -n <TARGET_NAMESPACE>
     ```

### Smoke tests

There is no smoke tests.

### How to debug

#### Kafka Operator

To debug Operator in VSCode you can use `Launch Kafka Operator` configuration which is already defined in 
`.vscode/launch.json` file.

The developer should configure environment variables: 

* `KUBECONFIG` - developer should **need to define** `KUBECONFIG` environment variable
  which should contains path to the kube-config file. It can be defined on configuration level
  or on the level of user's environment variables.
* `WATCH_NAMESPACE` - namespace, in which custom resources should be proceeded.
* `OPERATOR_NAMESPACE` - namespace, where Kafka should be proceeded.

#### Kafka-Services Operator

To debug Operator in VSCode you can use `Launch Kafka-Services Operator` configuration which is already defined in 
`.vscode/launch.json` file.

The developer should configure environment variables: 

* `KUBECONFIG` - developer should **need to define** `KUBECONFIG` environment variable
  which should contains path to the kube-config file. It can be defined on configuration level
  or on the level of user's environment variables.
* `WATCH_NAMESPACE` - namespace, in which custom resources should be proceeded.
* `OPERATOR_NAMESPACE` - namespace, where Kafka-Services should be proceeded.

### How to troubleshoot

There are no well-defined rules for troubleshooting, as each task is unique, but there are some tips that can do:

* Deploy parameters.
* Application manifest.
* Logs from all Kafka and Kafka-Services pods: operators, Kafka and others.

Also, developer can take a look on [Troubleshooting guide](/docs/public/troubleshooting.md).

## Evergreen strategy

To keep the component up to date, the following activities should be performed regularly:

* Vulnerabilities fixing.
* Kafka or Kafka supplementary services upgrade.
* Bug-fixing, improvement and feature implementation for operator and other related supplementary services.

## Useful links

* [Installation guide](/docs/public/installation.md).
* [Troubleshooting guide](/docs/public/troubleshooting.md).
* [Architecture Guide](/docs/public/architecture.md).
* [Internal Developer Guide](/docs/internal/developing.md).