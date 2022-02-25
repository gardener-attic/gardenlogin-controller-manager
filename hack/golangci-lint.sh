#!/usr/bin/env bash

# SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors
#
# SPDX-License-Identifier: Apache-2.0

set -o errexit
set -o pipefail

# For the check step concourse will set the following environment variables:
# SOURCE_PATH - path to component repository root directory.

if [[ -z "${SOURCE_PATH}" ]]; then
  export SOURCE_PATH="$(readlink -f "$(dirname ${0})/..")"
else
  export SOURCE_PATH="$(readlink -f ${SOURCE_PATH})"
fi

# Install golangci-lint (linting tool)
curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(go env GOPATH)/bin v1.44.1

function run_lint {
  local component=$1
  local target_dir=$2
  echo "> Lint $component"

  pushd "$target_dir"

  golangci-lint run ./... -E whitespace,wsl --skip-files "zz_generated.*"  --verbose --timeout 2m

  popd
}

run_lint gardenlogin-controller-manager "${SOURCE_PATH}"
# submodules are currently ignored by golangci-lint, hence we have to scan it separately (https://github.com/golangci/golangci-lint/issues/828)
run_lint gardenlogin-container-deployer "${SOURCE_PATH}/.landscaper/container"
