#!/bin/bash
#
# Copyright (c) 2020 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file
#
# SPDX-License-Identifier: Apache-2.0

set -e

SOURCE_PATH="$(dirname $0)/.."
TMP_DIR="$(mktemp -d)"
INSTALLATION_PATH="${TMP_DIR}/installation.yaml"
CONFIGMAP_PATH="${TMP_DIR}/configmap.yaml"

IMAGE_REGISTRY="${IMAGE_REGISTRY:-eu.gcr.io/gardener-project/development/images}"
CD_REGISTRY="${CD_REGISTRY:-eu.gcr.io/gardener-project/development}"

cat << EOF > ${INSTALLATION_PATH}
apiVersion: landscaper.gardener.cloud/v1alpha1
kind: Installation
metadata:
  name: gardenlogin-container-deployer
spec:
  componentDescriptor:
    ref:
      repositoryContext:
        type: ociRegistry
        baseUrl: ${CD_REGISTRY}
      componentName: github.com/gardener/gardenlogin-controller-manager/.landscaper/container
      version: ${EFFECTIVE_VERSION}

  blueprint:
    ref:
      resourceName: blueprint

  imports:
    targets:
    - name: applicationClusterTarget
      target: "#application-cluster"
    - name: runtimeClusterTarget
      target: "#runtime-cluster"
    data:
    - name: applicationClusterEndpoint
      configMapRef:
        key: applicationClusterEndpoint
        name: gardenlogin-container-deployer
#    - name: multiClusterDeploymentScenario
#      configMapRef:
#        key: multiClusterDeploymentScenario
#        name: gardenlogin-container-deployer
    - name: namePrefix
      configMapRef:
        key: namePrefix
        name: gardenlogin-container-deployer
    - name: namespace
      configMapRef:
        key: namespace
        name: gardenlogin-container-deployer

  exports: {}

EOF

endpointData="${APPLICATION_CLUSTER_ENDPOINT}"
multiClusterData=$(echo "true" | base64 -w0)
namePrefixData="gardenlogin-"
namespaceData="gardenlogin-system"

cat << EOF > ${CONFIGMAP_PATH}
apiVersion: v1
kind: ConfigMap
metadata:
  name: gardenlogin-container-deployer
data:
  applicationClusterEndpoint: ${endpointData}
  namePrefix: ${namePrefixData}
  namespace: ${namespaceData}
binaryData:
  multiClusterDeploymentScenario: ${multiClusterData}

EOF

echo "Resources stored under ${TMP_DIR}"
