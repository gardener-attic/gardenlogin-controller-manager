# SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors
#
# SPDX-License-Identifier: Apache-2.0

# Image URL to use all building/pushing image targets
IMG ?= eu.gcr.io/gardener-project/development/gardenlogin-container-deployer

REPO_ROOT           := $(shell git rev-parse --show-toplevel)
VERSION             := $(shell cat "$(REPO_ROOT)/VERSION")
EFFECTIVE_VERSION   := $(VERSION)-$(shell git rev-parse HEAD)

# Get the currently used golang install path (in GOPATH/bin, unless GOBIN is set)
ifeq (,$(shell go env GOBIN))
GOBIN=$(shell go env GOPATH)/bin
else
GOBIN=$(shell go env GOBIN)
endif

# Setting SHELL to bash allows bash commands to be executed by recipes.
# This is a requirement for 'setup-envtest.sh' in the test target.
# Options are set to exit when a recipe line exits non-zero or a piped command fails.
SHELL = /usr/bin/env bash -o pipefail
.SHELLFLAGS = -ec

all: build

##@ General

# The help target prints out all targets with their descriptions organized
# beneath their categories. The categories are represented by '##@' and the
# target descriptions by '##'. The awk commands is responsible for reading the
# entire set of makefiles included in this invocation, looking for lines of the
# file as xyz: ## something, and then pretty-format the target and help. Then,
# if there's a line with ##@ something, that gets pretty-printed as a category.
# More info on the usage of ANSI control characters for terminal formatting:
# https://en.wikipedia.org/wiki/ANSI_escape_code#SGR_parameters
# More info on the awk command:
# http://linuxcommand.org/lc3_adv_awk.php

help: ## Display this help.
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_0-9-]+:.*?##/ { printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

##@ Development

fmt: ## Run go fmt against code.
	go fmt ./...

lint: $(GOPATH)/bin/golangci-lint ## Run golangci-lint against code.
	golangci-lint run ./... -E golint,whitespace,wsl --skip-files "zz_generated.*"

ENVTEST_ASSETS_DIR=$(shell pwd)/testbin
test: fmt lint ## Run tests.
	mkdir -p "${ENVTEST_ASSETS_DIR}"
	test -f "${ENVTEST_ASSETS_DIR}"/setup-envtest.sh || curl -sSLo "${ENVTEST_ASSETS_DIR}"/setup-envtest.sh https://raw.githubusercontent.com/kubernetes-sigs/controller-runtime/${CR_VERSION}/hack/setup-envtest.sh
	source "${ENVTEST_ASSETS_DIR}"/setup-envtest.sh; fetch_envtest_tools "${ENVTEST_ASSETS_DIR}"; setup_envtest_env "${ENVTEST_ASSETS_DIR}"; go test ./... -coverprofile cover.out

##@ Build

build: fmt lint ## Build container-deployer binary.
	go build -o bin/container-deployer main.go

run: generate fmt lint ## Run a controller from your host.
	go run ./main.go

docker-build:  ## Build docker image with the container-deployer.
	docker build -t $(IMG):$(EFFECTIVE_VERSION) .

docker-push: ## Push docker image with the container-deployer.
	@docker push $(IMG):$(EFFECTIVE_VERSION)

cnudie-generate-cd: ## Generate component descriptor.
	@EFFECTIVE_VERSION=$(EFFECTIVE_VERSION) ./hack/generate-cd.sh

cnudie-create-installation: ## Create landscaper installation
	@EFFECTIVE_VERSION=$(EFFECTIVE_VERSION) ./hack/create-installation.sh

$(GOPATH)/bin/golangci-lint:
	curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(go env GOPATH)/bin v1.40.1

# go-get-tool will 'go get' any package $2 and install it to $1.
PROJECT_DIR := $(shell dirname $(abspath $(lastword $(MAKEFILE_LIST))))
define go-get-tool
@[ -f $(1) ] || { \
set -e ;\
TMP_DIR=$$(mktemp -d) ;\
cd $$TMP_DIR ;\
go mod init tmp ;\
echo "Downloading $(2)" ;\
GOBIN=$(PROJECT_DIR)/bin go get $(2) ;\
rm -rf $$TMP_DIR ;\
}
endef
