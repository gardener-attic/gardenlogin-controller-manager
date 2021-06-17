/*
SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors

SPDX-License-Identifier: Apache-2.0
*/

package v1alpha1

// ExecPluginConfig contains a reference to the garden and shoot cluster
type ExecPluginConfig struct {
	// ShootRef references the shoot cluster
	ShootRef ShootRef `json:"shootRef"`
	// GardenClusterIdentity is the cluster identifier of the garden cluster.
	// See cluster-identity ConfigMap in kube-system namespace of the garden cluster
	GardenClusterIdentity string `json:"gardenClusterIdentity"`
}

// ShootRef references the shoot cluster by namespace and name
type ShootRef struct {
	// Namespace is the namespace of the shoot cluster
	Namespace string `json:"namespace"`
	// Name is the name of the shoot cluster
	Name string `json:"name"`
}
