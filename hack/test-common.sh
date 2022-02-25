# SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors
#
# SPDX-License-Identifier: Apache-2.0

ENVTEST_K8S_VERSION=${ENVTEST_K8S_VERSION:-"1.23"}

function run_test {
  local component=$1
  local target_dir=$2
  local go_test_additional_flags=$3
  echo "> Test $component"

  pushd "${target_dir}" || exit

  make envtest

  # --use-env allows overwriting the envtest tools path via the KUBEBUILDER_ASSETS env var just like it was before
  KUBEBUILDER_ASSETS=$(bin/setup-envtest use --use-env -p path "${ENVTEST_K8S_VERSION}")
  export KUBEBUILDER_ASSETS

  echo "using envtest tools installed at '$KUBEBUILDER_ASSETS'"

  GO111MODULE=on go test ./... -coverprofile cover.out "${go_test_additional_flags}"

  popd || exit
}
