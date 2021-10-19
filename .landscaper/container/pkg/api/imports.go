// Copyright (c) 2021 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file
// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package api

import (
	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	v1 "k8s.io/api/core/v1"
)

// Imports defines the structure for the required configuration values from other components.
type Imports struct {
	// ApplicationClusterEndpoint holds the endpoint of the application cluster
	ApplicationClusterEndpoint string `json:"applicationClusterEndpoint" yaml:"applicationClusterEndpoint"`

	// ApplicationClusterTarget is the kubeconfig of the application cluster into which the application resources
	// like ValidatingWebhookConfigurations for the ConfigMaps are installed into (which is usually the virtual-garden}.
	// Must not be set when MultiClusterDeploymentScenario is false.
	ApplicationClusterTarget lsv1alpha1.Target `json:"applicationClusterTarget" yaml:"applicationClusterTarget"`

	// RuntimeClusterTarget is the kubeconfig of the hosting cluster into which the gardenlogin-controller-manager shall be installed.
	// Must not be set when MultiClusterDeploymentScenario is false.
	RuntimeClusterTarget lsv1alpha1.Target `json:"runtimeClusterTarget" yaml:"runtimeClusterTarget"`

	// SingleClusterTarget is the kubeconfig of cluster into which the resources shall be installed.
	// Must not be set when MultiClusterDeploymentScenario is true.
	SingleClusterTarget lsv1alpha1.Target `json:"singleClusterTarget" yaml:"singleClusterTarget"`

	// MultiClusterDeploymentScenario is true when the runtime part and application part is deployed into separate clusters. It is false when only a single cluster is used
	// if true, the multiCluster should be used. If false, singleCluster should be used.
	MultiClusterDeploymentScenario bool `json:"multiClusterDeploymentScenario" yaml:"multiClusterDeploymentScenario"`

	// NamePrefix is the name prefix of the resources build by kustomize.
	NamePrefix string `json:"namePrefix" yaml:"namePrefix"`

	// Namespace is the namespace into which the resources shall be installed. The namespace must not be shared with other components as the namespace will be deleted by the deploy container on DELETE operation.
	// It must not start with garden- to prevent name clashes with project namespaces and it must not be the garden namespace.
	Namespace string `json:"namespace" yaml:"namespace"`

	// Gardenlogin contains configuration for the gardenlogin-controller-manager.
	Gardenlogin Gardenlogin `json:"gardenlogin" yaml:"gardenlogin"`
}

// Gardenlogin contains configuration for the gardenlogin-controller-manager.
type Gardenlogin struct {
	// ManagerResources define the resource requirements by the "manager" container.
	ManagerResources v1.ResourceRequirements `json:"managerResources" yaml:"managerResources"`

	// KubeRBACProxyResources define the resource requirements by the "kube-rbac-proxy" container.
	KubeRBACProxyResources v1.ResourceRequirements `json:"kubeRbacProxyResources" yaml:"kubeRbacProxyResources"`
}
