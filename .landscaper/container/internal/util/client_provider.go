/*
SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors

SPDX-License-Identifier: Apache-2.0
*/

package util

import (
	"fmt"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// ControllerRuntimeClientProvider is able to take a kubeconfig either directly or
// from a file and return a controller-runtime client for it.
type ControllerRuntimeClientProvider interface {
	// FromBytes reads YAML directly and returns a controller-runtime kubernetes client.
	FromBytes(kubeconfig []byte) (client.Client, error)
}

// ClientGoClientProvider is able to take a kubeconfig either directly or
// from a file and return a client-go client for it.
type ClientGoClientProvider interface {
	// FromBytes reads YAML directly and returns a client-go kubernetes client.
	FromBytes(kubeconfig []byte) (kubernetes.Interface, error)
}

type crClientProvider struct{}

var _ ControllerRuntimeClientProvider = &crClientProvider{}

// NewControllerRuntimeClientProvider returns a new ClientProvider.
func NewControllerRuntimeClientProvider() ControllerRuntimeClientProvider {
	return &crClientProvider{}
}

// FromBytes reads YAML directly and returns a controller-runtime kubernetes client.
func (p *crClientProvider) FromBytes(kubeconfig []byte) (client.Client, error) {
	config, err := clientcmd.RESTConfigFromKubeConfig(kubeconfig)
	if err != nil {
		return nil, fmt.Errorf("failed to load kubeconfig: %w", err)
	}

	return client.New(config, client.Options{})
}

type cgClientProvider struct{}

var _ ClientGoClientProvider = &cgClientProvider{}

// NewClientGoClientProvider returns a new ClientProvider.
func NewClientGoClientProvider() ClientGoClientProvider {
	return &cgClientProvider{}
}

// FromBytes reads YAML directly and returns a client-go kubernetes client.
func (p *cgClientProvider) FromBytes(kubeconfig []byte) (kubernetes.Interface, error) {
	config, err := clientcmd.RESTConfigFromKubeConfig(kubeconfig)
	if err != nil {
		return nil, fmt.Errorf("failed to load kubeconfig: %w", err)
	}

	return kubernetes.NewForConfig(config)
}
