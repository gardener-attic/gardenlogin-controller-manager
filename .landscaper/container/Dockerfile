# SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors
#
# SPDX-License-Identifier: Apache-2.0

# Get the required binaries
FROM golang:1.17.3 as binaries
ARG kustomize_version=v4.3.0

WORKDIR /workspace
# Get kustomize
RUN wget -qO kustomize.tar.gz https://github.com/kubernetes-sigs/kustomize/releases/download/kustomize%2F${kustomize_version}/kustomize_${kustomize_version}_linux_amd64.tar.gz && \
    tar -zxf kustomize.tar.gz && \
    chmod +x kustomize

# Build the container-deployer binary
FROM golang:1.17.3 as builder

WORKDIR /workspace
# Copy the Go Modules manifests
COPY go.mod go.mod
COPY go.sum go.sum
# cache deps before building and copying source so that we don't need to re-download as much
# and so that source changes don't invalidate our downloaded layer
RUN go mod download

# Copy the go source
COPY cmd/ cmd/
COPY internal/ internal/
COPY pkg/ pkg/

# Build
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -o container-deployer cmd/gardenlogin/main.go

# Use distroless as minimal base image to package the container-deployer binary
# Refer to https://github.com/GoogleContainerTools/distroless for more details
FROM gcr.io/distroless/static:nonroot
WORKDIR /
COPY --from=builder /workspace/container-deployer .
COPY --from=binaries /workspace/kustomize /usr/bin/kustomize

# nonroot user https://github.com/GoogleContainerTools/distroless/blob/18b2d2c5ebfa58fe3e0e4ee3ffe0e2651ec0f7f6/base/base.bzl#L8
USER 65532:65532

ENTRYPOINT ["/container-deployer"]
