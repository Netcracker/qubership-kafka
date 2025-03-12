- [Introduction](#introduction)
- [Custom Resource Definition Versioning](#custom-resource-definition-versioning)
- [Useful Operator Commands](#useful-operator-commands)
- [Useful Helm commands](#useful-helm-commands)
- [Manual Linter Run](#manual-linter-run)

# Introduction

This guide provides the necessary information for the Kafka cluster development.

# Custom Resource Definition Versioning

Custom resource definition versioning allows to have different incompatible CRD versions of the Kafka cluster in several
namespaces of OpenShift/Kubernetes. Each Kafka cluster is placed in separate namespace and controlled by corresponding
operator. Each operator reconciles the only version of the application which it is compatible with.

## Build Custom Docker Image for Operator SDK

To make it easier to prepare Operator SDK environment, you can create Docker image that extends
`quay.io/operator-framework/operator-sdk:v1.16` image by installing `gcc` and `git` utilities:

```dockerfile
FROM quay.io/operator-framework/operator-sdk:v1.16

RUN microdnf install --nodocs \
    gcc \
    git \
    && microdnf clean all

ENTRYPOINT ["/bin/bash"]
```

You can use this script to build the extended image:

```sh
#!/usr/bin/env bash

set -e

DOCKER_FILE=Dockerfile
IMAGE_NAME=operator-sdk-ext:v1.16

docker build \
  --pull \
  --file=${DOCKER_FILE} \
  -t ${IMAGE_NAME} \
  --no-cache \
  .

docker inspect ${IMAGE_NAME}
```

To run the environment you can use the following command:

```sh
docker run -it -v <path_to_operator>:/kafka-service-operator operator-sdk-ext:v1.16
```

where `<path_to_operator>` is the path to your `kafka-service-operator` directory.

## Create New Operator Version with API Changes

There are times when incompatible changes do not affect deployment, but most often such changes cause problems. The changes
which can be incompatible and should be monitored are described in 
[Backward compatibility gotchas](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api_changes.md#backward-compatibility-gotchas)
.The most important of these are as following:

* Adding a new representation, since clients that only understood the old representation would not be aware of the new
  representation nor its semantics.
* Adding a new option to the set of fields, since it may not follow the appropriate conventions of the original object.
* Changing any validation rules, since it changes the assumptions about part of the API.

If API model has incompatible changes, new application version should be released. There are several steps to make it:

1. Add a new API by running the following command in the `kafka-service-operator` directory:

    ```sh
    operator-sdk create api --version <new_version> --kind KafkaService --resource
    ```

   where `<new_version>` is the name of version that should appear. For example, `v3`.

   After the operation is completed, the package `kafka-service-operator/api/<new_version>` has to be created.

2. Copy types structure from last active API version to `kafka-service-operator/api/<new_version>/kafkaservice_types.go` file.

3. Add the marker `//+kubebuilder:storageversion` to the `kafkaservice_types.go` of new version and remove it for previous.

4. Change dependencies from the previous version to the new one in code of operator:
   `kafka-service-operator/controllers` and `kafka-service-operator/controllers/provider` folders. 
   So, in the new version reconciliation of old model is not supported.

5. Make necessary changes in API.

6. Update generated code using the following commands:

    ```sh
    make generate
    make manifests
    ```

   Make sure generated CRD `kafka-service-operator/config/crd/bases/qubership.org_kafkaservices.yaml` contains new version.

7. Copy content of new generated CRD to `kafka-service-operator/charts/helm/kafka-service/crds/crd.yaml` file.

8. Change `apiVersion` from old to new one in `kafka-service-operator/charts/helm/kafka-service/templates/cr.yaml` file.

# Useful Operator Commands

_operator-sdk_:

* `operator-sdk create api --group qubership.org --version <new_version> --kind=KafkaService --resource`
  is the command to add a new custom resource definition API called KafkaService, with APIVersion
  `qubership.org/<new_version>` and Kind `KafkaService`.
* `operator-sdk create api --group qubership.org --version <new_version> --kind=KafkaService --controller`
  is the command to add a new controller to the project that will watch and reconcile the KafkaService resource.
* `make generate` is the command to update the generated code for the KafkaService resource.
  You should run this command every time when you change `kafkaservice_types.go`.
* `make manifests` is the command to update the OpenAPI validation section in the custom resource definition.
  You should run this command every time when you change `kafkaservice_types.go`.

_minikube_:

* `minikube start --kubernetes-version=v1.11.10` is the command to start minikube with specific Kubernetes version.
* `minikube dashboard` is the command to run Kubernetes dashboard.

_kubectl_:

* `kubectl config set-context $(kubectl config current-context) --namespace=${NAMESPACE}` is the command to permanently save
  the namespace for all subsequent kubectl commands in that context.
* `kubectl get crd` is the command to get custom resource definitions.

# Useful Helm Commands

* `helm template kafka-service/` is the command that renders templates.
* `helm install --dry-run your-release-name kafka-service/` is the command that runs template rendering with connecting
  to Kubernetes/OpenShift, but does not deploy anything.
* `helm install --debug your-release-name kafka-service/` is the command that prints additional logs which can help
  investigate issues with charts.
* `helm test kafka-service/` is the command to run test suite which is stored in directory `kafka-service/tests`.

# Manual Linter Run

To run `golangci-lint` linter locally, you need to install it on your computer with the following command:

```sh
go install github.com/golangci/golangci-lint/cmd/golangci-lint@<version>
```

where `<version>` is `golangci-lint` version. For example, `v1.50.1`.

Make changes to the file with `golangci-lint` configuration (`.golangci.yml`) if it is necessary.
Then go to the folder that contains main module and run linter:

```sh
cd kafka-service-operator
```

```sh
golangci-lint run ./... -v
```

More information about `golangci-lint` linter you can find in [official documentation](https://golangci-lint.run/).
