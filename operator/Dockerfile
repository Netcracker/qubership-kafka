# Build the manager binary
FROM --platform=$BUILDPLATFORM golang:1.24-alpine3.22 as builder
ARG BUILDPLATFORM
ARG TARGETOS
ARG TARGETARCH

WORKDIR /workspace
# Copy the Go Modules manifests
COPY go.mod go.mod
COPY go.sum go.sum
# cache deps before building and copying source so that we don't need to re-download as much
# and so that source changes don't invalidate our downloaded layer
RUN go mod download

# Copy the go source
COPY main.go main.go
COPY api api/
COPY controllers controllers/
COPY util util/
COPY cfg cfg/
COPY workers workers/

# Tests
RUN CGO_ENABLED=0 go test -v ./...
# Build
RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} GO111MODULE=on go build -a -o manager main.go

# Use distroless as minimal base image to package the manager binary
# Refer to https://github.com/GoogleContainerTools/distroless for more details
FROM alpine:3.22

ENV USER_UID=1001 \
    USER_NAME=kafka-service-operator \
    GROUP_NAME=kafka-service-operator

WORKDIR /
COPY --from=builder --chown=${USER_UID} /workspace/manager .

RUN set -x \
    && apk add --upgrade --no-cache \
        curl

# Avoiding vulnerabilities
RUN set -x \
    && apk add --upgrade --no-cache apk-tools

# Upgrade all tools to avoid vulnerabilities
RUN set -x && apk upgrade --no-cache --available

RUN addgroup ${GROUP_NAME} && adduser -D -G ${GROUP_NAME} -u ${USER_UID} ${USER_NAME}
USER ${USER_UID}

ENTRYPOINT ["/manager"]
