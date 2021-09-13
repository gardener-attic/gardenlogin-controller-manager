/*
SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors

SPDX-License-Identifier: Apache-2.0
*/

package fake

import (
	"fmt"

	"github.com/gardener/gardenlogin-controller-manager/.landscaper/container/internal/util"

	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// ControllerRuntimeClientProvider holds the fake controller-runtime clients
type ControllerRuntimeClientProvider struct {
	fakeClients map[string]client.Client
}

var _ util.ControllerRuntimeClientProvider = &ControllerRuntimeClientProvider{}

// NewFakeControllerRuntimeClientProvider returns a new ControllerRuntimeClientProvider that returns a static
// client for a given kubeconfig / kubeconfig file.
func NewFakeControllerRuntimeClientProvider() *ControllerRuntimeClientProvider {
	return &ControllerRuntimeClientProvider{
		fakeClients: map[string]client.Client{},
	}
}

// WithClient adds an additional client to the provider, which it will
// return whenever a consumer requests a client with the same key.
func (p *ControllerRuntimeClientProvider) WithClient(key string, c client.Client) *ControllerRuntimeClientProvider {
	p.fakeClients[key] = c
	return p
}

func (p *ControllerRuntimeClientProvider) tryGetClient(key string) (client.Client, error) {
	if c, ok := p.fakeClients[key]; ok {
		return c, nil
	}

	return nil, fmt.Errorf("no fake client configured for %q", key)
}

// FromBytes returns a controller-runtime client for the provided kubeconfig
// that was added beforehand using the WithClient function.
func (p *ControllerRuntimeClientProvider) FromBytes(kubeconfig []byte) (client.Client, error) {
	return p.tryGetClient(string(kubeconfig))
}

// ClientGoClientProvider holds the fake client-go clients
type ClientGoClientProvider struct {
	fakeClients map[string]kubernetes.Interface
}

var _ util.ClientGoClientProvider = &ClientGoClientProvider{}

// NewFakeClientGoClientProvider returns a new ClientGoClientProvider that returns a static
// client for a given kubeconfig / kubeconfig file.
func NewFakeClientGoClientProvider() *ClientGoClientProvider {
	return &ClientGoClientProvider{
		fakeClients: map[string]kubernetes.Interface{},
	}
}

// WithClient adds an additional client to the provider, which it will
// return whenever a consumer requests a client with the same key.
func (p *ClientGoClientProvider) WithClient(key string, c kubernetes.Interface) *ClientGoClientProvider {
	p.fakeClients[key] = c
	return p
}

func (p *ClientGoClientProvider) tryGetClient(key string) (kubernetes.Interface, error) {
	if c, ok := p.fakeClients[key]; ok {
		return c, nil
	}

	return nil, fmt.Errorf("no fake client configured for %q", key)
}

// FromBytes returns a client go kubernetes client for the provided kubeconfig
// that was added beforehand using the WithClient function.
func (p *ClientGoClientProvider) FromBytes(kubeconfig []byte) (kubernetes.Interface, error) {
	return p.tryGetClient(string(kubeconfig))
}
