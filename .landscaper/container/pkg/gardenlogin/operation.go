// Copyright (c) 2021 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file
// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package gardenlogin

import (
	"context"

	"github.com/gardener/gardenlogin-controller-manager/.landscaper/container/pkg/api"

	"github.com/sirupsen/logrus"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Interface is an interface for the operation.
type Interface interface {
	// Reconcile performs a reconcile operation.
	Reconcile(context.Context) (*api.Exports, error)
	// Delete performs a delete operation.
	Delete(context.Context) error
}

// Prefix is the prefix for resource names related to the gardenlogin-controller-manager.
const Prefix = "gardenlogin"

// operation contains the configuration for a operation.
type operation struct {
	// multiCluster holds the data for the multi-cluster deployment scenario with which the runtime part and application part is deployed into separate clusters
	multiCluster multiCluster

	// singleCluster holds the data for the single-cluster deployment scenario with which the resources are deployed into a single cluster
	singleCluster cluster

	// multiClusterDeploymentScenario is true when the runtime part and application part is deployed into separate clusters. It is false when only a single cluster is used
	// if true, the multiCluster should be used. If false, singleCluster should be used.
	multiClusterDeploymentScenario bool

	// namePrefix is the name prefix of the resources build by kustomize. Defaults to gardenlogin-
	namePrefix string

	// namespace is the namespace into which the resources shall be installed.
	namespace string

	// log is a logger.
	log logrus.FieldLogger

	// clock provides the current time
	clock Clock

	// imports contains the imports configuration.
	imports *api.Imports

	exports api.Exports

	// imageRefs contains the image references from the component descriptor that are needed for the Deployments.
	imageRefs api.ImageRefs

	contents api.Contents

	state api.State
}

type multiCluster struct {
	// runtimeCluster holds the data for the runtime cluster.
	runtimeCluster cluster

	// applicationCluster holds the data for the application cluster.
	applicationCluster cluster
}

type cluster struct {
	//clientSet holds the client set for the cluster
	*clientSet

	// kubeconfig holds the path to the kubeconfig of the cluster.
	kubeconfig string
}

type clientSet struct {
	// client is the Kubernetes client for the cluster.
	client client.Client
	// kubernetes is the kubernetes client set for the cluster.
	kubernetes kubernetes.Interface
}

// NewOperation returns a new operation structure that implements Interface.
func NewOperation(
	runtimeClient client.Client,
	applicationClient client.Client,
	log *logrus.Logger,
	clock Clock,
	namespace string,
	imports *api.Imports,
	imageRefs *api.ImageRefs,
	contents api.Contents,
	state api.State,
) Interface {
	return &operation{
		multiCluster: multiCluster{
			runtimeCluster: cluster{
				clientSet: &clientSet{
					client:     runtimeClient,
					kubernetes: nil, // TODO
				},
				//kubeconfig: "",
			},
			applicationCluster: cluster{
				clientSet: &clientSet{
					client:     runtimeClient,
					kubernetes: nil, // TODO
				},
				//kubeconfig: "",
			},
		},
		log:   log,
		clock: clock,

		namePrefix: "gardenlogin-", // TODO
		namespace:  namespace,
		imports:    imports,
		imageRefs:  *imageRefs,
		contents:   contents,
		state:      state,
	}
}
