#!/usr/bin/env bash

# SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors
#
# SPDX-License-Identifier: Apache-2.0

set -o errexit
set -o pipefail

# For the check step concourse will set the following environment variables:
# SOURCE_PATH - path to component repository root directory.
if [[ -z "${SOURCE_PATH}" ]]; then
  export SOURCE_PATH="$(readlink -f "$(dirname ${0})/../../../")"
else
  export SOURCE_PATH="$(readlink -f ${SOURCE_PATH})"
fi

KUSTOMIZE_VERSION=${KUSTOMIZE_VERSION:-"v4.3.0"}
GO_TEST_ADDITIONAL_FLAGS=${GO_TEST_ADDITIONAL_FLAGS:-""}
OS=${OS:-$(go env GOOS)}
ARCH=${ARCH:-$(go env GOARCH)}

bin_dir="${SOURCE_PATH}/.landscaper/container/bin"
mkdir -p "${bin_dir}"

pushd "${bin_dir}"

echo "> Get kustomize"
wget -qO kustomize.tar.gz "https://github.com/kubernetes-sigs/kustomize/releases/download/kustomize%2F${KUSTOMIZE_VERSION}/kustomize_${KUSTOMIZE_VERSION}_${OS}_${ARCH}.tar.gz"
tar -zxf kustomize.tar.gz && rm kustomize.tar.gz
chmod +x kustomize

export PATH="${bin_dir}:$PATH"

popd

source "${SOURCE_PATH}/hack/test-common.sh"

run_test gardenlogin-container-deployer "${SOURCE_PATH}/.landscaper/container" "${GO_TEST_ADDITIONAL_FLAGS}"
