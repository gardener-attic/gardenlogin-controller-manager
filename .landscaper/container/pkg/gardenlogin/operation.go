// Copyright (c) 2021 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file
// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package gardenlogin

import (
	"context"
	"github.com/gardener/gardenlogin-controller-manager/.landscaper/container/pkg/api"
	//"github.com/gardener/gardenlogin-controller-manager/.landscaper/container/pkg/api/helper"
	//"github.com/gardener/gardenlogin-controller-manager/.landscaper/container/pkg/provider"
	"github.com/sirupsen/logrus"
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
	// runtimeClient is the Kubernetes client for the runtime cluster.
	runtimeClient client.Client

	// applicationClient is the Kubernetes client for the application cluster.
	applicationClient client.Client

	// log is a logger.
	log logrus.FieldLogger

	// imports contains the imports configuration.
	imports *api.Imports

	exports api.Exports

	// imageRefs contains the image references from the component descriptor that are needed for the Deployments
	imageRefs api.ImageRefs

	state api.State
}

// NewOperation returns a new operation structure that implements Interface.
func NewOperation(
	runtimeClient client.Client,
	applicationClient client.Client,
	log *logrus.Logger,
	imports *api.Imports,
	imageRefs *api.ImageRefs,
	state api.State,
) Interface {
	return &operation{
		runtimeClient:     runtimeClient,
		applicationClient: applicationClient,
		log:               log,
		imports:           imports,
		imageRefs:         *imageRefs,
		state:             state,
	}
}
