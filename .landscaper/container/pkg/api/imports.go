// Copyright (c) 2021 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file
// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package api

import (
	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
)

// Imports defines the structure for the required configuration values from other components.
type Imports struct {
	// ApplicationClusterTarget is the kubeconfig of the application cluster into which the application resources
	// like ValidatingWebhookConfigurations for the ConfigMaps are installed into (which is usually the virtual-garden}.
	ApplicationClusterTarget lsv1alpha1.Target `json:"applicationClusterTarget" yaml:"applicationClusterTarget"`

	// RuntimeClusterTarget is the kubeconfig of the hosting cluster into which the virtual garden shall be installed.
	RuntimeClusterTarget lsv1alpha1.Target `json:"runtimeClusterTarget" yaml:"runtimeClusterTarget"`
	// RuntimeCluster contains settings for the hosting cluster that runs the gardenlogin-controller-manager.
	RuntimeCluster RuntimeCluster `json:"runtimeCluster" yaml:"runtimeCluster"`

	// Gardenlogin contains configuration for the gardenlogin-controller-manager.
	Gardenlogin Gardenlogin `json:"gardenlogin" yaml:"gardenlogin"`

	// ApplicationClusterEndpoint holds the endpoint of the application cluster
	ApplicationClusterEndpoint string
}

// RuntimeCluster contains settings for the hosting cluster that runs the gardenlogin-controller-manager.
type RuntimeCluster struct {
}

// Gardenlogin contains configuration for the gardenlogin-controller-manager.
type Gardenlogin struct {
}
