#!/bin/bash
#
# Copyright (c) 2020 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file
#
# SPDX-License-Identifier: Apache-2.0

set -e

SOURCE_PATH="$(dirname $0)/.."
REPO_CTX="eu.gcr.io/gardener-project/development"
CA_PATH="$(mktemp -d)"
BASE_DEFINITION_PATH="${CA_PATH}/component-descriptor.yaml"

if ! which component-cli 1>/dev/null; then
  echo -n "component-cli is required to generate the component descriptors"
  echo -n "Trying to installing it..."
  go get github.com/gardener/component-cli/cmd/component-cli

  if ! which component-cli 1>/dev/null; then
    echo -n "component-cli was successfully installed but the binary cannot be found"
    echo -n "Try adding the \$GOPATH/bin to your \$PATH..."
    exit 1
  fi
fi
if ! which jq 1>/dev/null; then
  echo -n "jq canot be found"
  exit 1
fi

echo "> Generate Component Descriptor ${EFFECTIVE_VERSION}"
echo "> Creating base definition"
component-cli ca create "${CA_PATH}" \
    --component-name=github.com/gardener/gardenlogin-controller-manager/.landscaper/container \
    --component-version=${EFFECTIVE_VERSION} \
    --repo-ctx=${REPO_CTX}

echo "> Extending resources.yaml: adding image of gardenlogin-container-deployer"
RESOURCES_BASE_PATH="$(mktemp -d)"
cp -R ".landscaper/" "${RESOURCES_BASE_PATH}"

RESOURCES_FILE_PATH="${RESOURCES_BASE_PATH}/resources.yaml"
cat << EOF >> ${RESOURCES_FILE_PATH}
---
type: ociImage
name: gardenlogin-container-deployer
relation: local
access:
  type: ociRegistry
  imageReference: eu.gcr.io/gardener-project/development/gardenlogin-container-deployer:${EFFECTIVE_VERSION}
...
EOF

echo "> Creating ctf"
CTF_PATH=./gen/ctf.tar
mkdir -p ./gen
[ -e $CTF_PATH ] && rm ${CTF_PATH}
CTF_PATH=${CTF_PATH} BASE_DEFINITION_PATH=${BASE_DEFINITION_PATH} CURRENT_COMPONENT_REPOSITORY=${REPO_CTX} RESOURCES_FILE_PATH=${RESOURCES_FILE_PATH} bash $SOURCE_PATH/.ci/component_descriptor

component-cli ctf push --repo-ctx=${REPO_CTX} "${CTF_PATH}"
