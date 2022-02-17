/*
SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors

SPDX-License-Identifier: Apache-2.0
*/

package fake

import (
	"context"
	"time"

	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/gardener/gardenlogin-controller-manager/.landscaper/container/internal/util"
)

// Factory implements util.Factory interface
type Factory struct {
	// contextImpl is the root context any command should use.
	contextImpl context.Context

	// clockImpl can be used to override the clock implementation. Will use a real clock if not set.
	clockImpl *Clock

	// controllerRuntimeClientProviderImpl holds the controller-runtime client provider implementation.
	controllerRuntimeClientProviderImpl *ControllerRuntimeClientProvider
	// clientGoClientProviderImpl holds the client-go client provider implementation.
	clientGoClientProviderImpl *ClientGoClientProvider
}

var _ util.Factory = &Factory{}

// NewFakeFactory returns a new fake factory. Nil parameters will be defaulted
func NewFakeFactory(clock *Clock, cr *ControllerRuntimeClientProvider, cg *ClientGoClientProvider) (*Factory, error) {
	if clock == nil {
		var err error

		clock, err = NewClock()
		if err != nil {
			return nil, err
		}
	}

	if cr == nil {
		cr = NewFakeControllerRuntimeClientProvider()
	}

	if cg == nil {
		cg = NewFakeClientGoClientProvider()
	}

	return &Factory{
		clockImpl:                           clock,
		controllerRuntimeClientProviderImpl: cr,
		clientGoClientProviderImpl:          cg,
	}, nil
}

// Context returns the root context any command should use.
func (f *Factory) Context() context.Context {
	if f.contextImpl != nil {
		return f.contextImpl
	}

	return context.Background()
}

// Clock returns a clock that provides access to the current time.
func (f *Factory) Clock() util.Clock {
	return f.clockImpl
}

// ControllerRuntimeClientProvider returns a controller-runtime client provider to get controller-runtime clients for a given kubeconfig
func (f *Factory) ControllerRuntimeClientProvider() util.ControllerRuntimeClientProvider {
	return f.controllerRuntimeClientProviderImpl
}

// ClientGoClientProvider returns a client-go client provider to get client-go clients for a given kubeconfig
func (f *Factory) ClientGoClientProvider() util.ClientGoClientProvider {
	return f.clientGoClientProviderImpl
}

// WithControllerRuntimeClient adds an additional controller runtime client to the provider, which it will
// return whenever a consumer requests a controller-runtime client with the same key.
func (f *Factory) WithControllerRuntimeClient(key string, c client.Client) util.ControllerRuntimeClientProvider {
	return f.controllerRuntimeClientProviderImpl.WithClient(key, c)
}

// WithClientGoClient adds an client-go client to the provider, which it will
// return whenever a consumer requests a client-go client with the same key.
func (f *Factory) WithClientGoClient(key string, c kubernetes.Interface) util.ClientGoClientProvider {
	return f.clientGoClientProviderImpl.WithClient(key, c)
}

// WithTime sets the provided time on the fake clock
func (f *Factory) WithTime(t time.Time) *Clock {
	f.clockImpl.FakeTime = t
	return f.clockImpl
}
