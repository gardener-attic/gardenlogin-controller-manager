/*
SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors

SPDX-License-Identifier: Apache-2.0
*/

package constants

const (
	// GardenerOperationsRole is a constant for a label that describes a role.
	GardenerOperationsRole = "operations.gardener.cloud/role"
	// GardenerOperationsKubeconfig is the value of the GardenerOperationsRole key indicating type 'kubeconfig'.
	GardenerOperationsKubeconfig = "kubeconfig"

	// DataKeyKubeconfig is the key in a configmap data holding the kubeconfig.
	DataKeyKubeconfig = "kubeconfig"
)
