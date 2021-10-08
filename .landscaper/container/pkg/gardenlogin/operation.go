// Copyright (c) 2021 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file
// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package gardenlogin

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"

	"github.com/gardener/gardenlogin-controller-manager/.landscaper/container/internal/util"
	"github.com/gardener/gardenlogin-controller-manager/.landscaper/container/pkg/api"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	"github.com/sirupsen/logrus"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Interface is an interface for the operation.
type Interface interface {
	// Reconcile performs a reconcile operation.
	Reconcile(context.Context) error
	// Delete performs a delete operation.
	Delete(context.Context) error
}

// TLSSecretSuffix is the suffix for the secret that holds the tls certificate for the webhook-service.
const TLSSecretSuffix = "-tls"

// operation contains the configuration for a operation.
type operation struct {
	// multiCluster holds the data for the multi-cluster deployment scenario with which the runtime part and application part is deployed into separate clusters.
	multiCluster *multiCluster

	// singleCluster holds the data for the single-cluster deployment scenario with which the resources are deployed into a single cluster
	singleCluster *cluster

	// log is a logger.
	log logrus.FieldLogger

	// clock provides the current time
	clock util.Clock

	// imports contains the imports configuration.
	imports *api.Imports

	// imageRefs contains the image references from the component descriptor that are needed for the Deployments.
	imageRefs *api.ImageRefs

	// contents holds the content data of the landscaper component.
	contents *api.Contents
}

type multiCluster struct {
	// runtimeCluster holds the data for the runtime cluster.
	runtimeCluster *cluster

	// applicationCluster holds the data for the application cluster.
	applicationCluster *cluster
}

type cluster struct {
	//clientSet holds the client set for the cluster
	*clientSet

	// kubeconfig holds the path to the kubeconfig of the cluster.
	kubeconfig string
}

type clientSet struct {
	// client is the controller-runtime kubernetes client for the cluster.
	client client.Client
	// kubernetes is the client-go kubernetes client set for the cluster.
	kubernetes kubernetes.Interface
}

// runtimeClusterClient returns the cluster struct for the runtime cluster depending on the MultiClusterDeploymentScenario import flag
// application cluster and runtime cluster is the same in case of single cluster deployment
func (o *operation) runtimeCluster() *cluster {
	if !o.imports.MultiClusterDeploymentScenario {
		return o.singleCluster
	}

	return o.multiCluster.runtimeCluster
}

// applicationClusterClient returns the cluster struct for the application cluster depending on the MultiClusterDeploymentScenario import flag
// application cluster and runtime cluster is the same in case of single cluster deployment
func (o *operation) applicationCluster() *cluster {
	if !o.imports.MultiClusterDeploymentScenario {
		return o.singleCluster
	}

	return o.multiCluster.applicationCluster
}

// NewOperation returns a new operation structure that implements Interface.
func NewOperation(
	f util.Factory,
	log *logrus.Logger,
	imports *api.Imports,
	imageRefs *api.ImageRefs,
	contents *api.Contents,
) (Interface, error) {
	var (
		mc *multiCluster
		sc *cluster
	)

	if imports.MultiClusterDeploymentScenario {
		runtimeCluster, err := newClusterFromTarget(f, imports.RuntimeClusterTarget)
		if err != nil {
			return nil, fmt.Errorf("could not create runtime cluster from target: %w", err)
		}

		applicationCluster, err := newClusterFromTarget(f, imports.ApplicationClusterTarget)
		if err != nil {
			return nil, fmt.Errorf("could not create application cluster from target: %w", err)
		}

		mc = &multiCluster{
			runtimeCluster:     runtimeCluster,
			applicationCluster: applicationCluster,
		}
	} else {
		var err error

		sc, err = newClusterFromTarget(f, imports.SingleClusterTarget)
		if err != nil {
			return nil, fmt.Errorf("could not create cluster from target: %w", err)
		}
	}

	return &operation{
		multiCluster:  mc,
		singleCluster: sc,

		log:   log,
		clock: f.Clock(),

		imports:   imports,
		imageRefs: imageRefs,
		contents:  contents,
	}, nil
}

// kubeconfigFromTarget returns the kubeconfig from the given target.
func kubeconfigFromTarget(target lsv1alpha1.Target) ([]byte, error) {
	targetConfig := &lsv1alpha1.KubernetesClusterTargetConfig{}
	if err := json.Unmarshal(target.Spec.Configuration.RawMessage, targetConfig); err != nil {
		return nil, fmt.Errorf("unable to parse target confíguration: %w", err)
	}

	if targetConfig.Kubeconfig.StrVal == nil {
		return nil, errors.New("kubeconfig is not set in target")
	}

	return []byte(*targetConfig.Kubeconfig.StrVal), nil
}

// newClusterFromTarget returns a cluster struct for the given target and writes the kubeconfig of the target to a temporary file
func newClusterFromTarget(f util.Factory, target lsv1alpha1.Target) (*cluster, error) {
	kubeconfig, err := kubeconfigFromTarget(target)
	if err != nil {
		return nil, fmt.Errorf("could not get kubeconfig from target: %w", err)
	}

	kubeconfigFile, err := ioutil.TempFile("", "kubeconfig-*.yaml")
	if err != nil {
		return nil, err
	}

	err = ioutil.WriteFile(kubeconfigFile.Name(), kubeconfig, 0600)
	if err != nil {
		return nil, err
	}

	kube, err := f.ClientGoClientProvider().FromBytes(kubeconfig)
	if err != nil {
		return nil, fmt.Errorf("could not create kubenernetes clientset from config: %w", err)
	}

	client, err := f.ControllerRuntimeClientProvider().FromBytes(kubeconfig)
	if err != nil {
		return nil, fmt.Errorf("could not create client from config: %w", err)
	}

	return &cluster{
		clientSet: &clientSet{
			client:     client,
			kubernetes: kube,
		},
		kubeconfig: kubeconfigFile.Name(),
	}, nil
}
