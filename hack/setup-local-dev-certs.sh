#!/bin/bash -e
#
# SPDX-FileCopyrightText: 2019 SAP SE or an SAP affiliate company and Gardener contributors
#
# SPDX-License-Identifier: Apache-2.0

cd "${0%/*}"
echo "Generating certificates.."
./gen-certs.sh

path_tls="../.landscaper/blueprint/config/secret/tls"

if [ -z "${CERTS_DIR}" ]; then
    export CERTS_DIR=/tmp/k8s-webhook-server/serving-certs
fi

echo "Creating temporary directory for serving certs under ${CERTS_DIR}"
mkdir -p $CERTS_DIR

echo "Copying certificates to ${CERTS_DIR}"
cp "${path_tls}/gardenlogin-controller-manager-tls.pem" ${CERTS_DIR}/tls.crt
cp "${path_tls}/gardenlogin-controller-manager-tls-key.pem" ${CERTS_DIR}/tls.key
