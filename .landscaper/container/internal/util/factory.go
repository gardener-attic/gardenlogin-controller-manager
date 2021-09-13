/*
SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors

SPDX-License-Identifier: Apache-2.0
*/

package util

import (
	"context"

	"sigs.k8s.io/controller-runtime/pkg/manager/signals"
)

// Factory provides abstractions for the context, clock and client providers that should be used by the commmand
type Factory interface {
	// Context returns the root context any command should use.
	Context() context.Context
	// Clock returns a clock that provides access to the current time.
	Clock() Clock
	// ControllerRuntimeClientProvider returns a controller-runtime client provider to get controller-runtime clients for a given kubeconfig
	ControllerRuntimeClientProvider() ControllerRuntimeClientProvider
	// ClientGoClientProvider returns a client-go client provider to get client-go clients for a given kubeconfig
	ClientGoClientProvider() ClientGoClientProvider
}

// FactoryImpl implements util.Factory interface
type FactoryImpl struct{}

var _ Factory = &FactoryImpl{}

// Context returns the root context any command should use.
func (f *FactoryImpl) Context() context.Context {
	return signals.SetupSignalHandler()
}

// Clock returns a clock that provides access to the current time.
func (f *FactoryImpl) Clock() Clock {
	return &RealClock{}
}

// ControllerRuntimeClientProvider returns a controller-runtime client provider to get controller-runtime clients for a given kubeconfig
func (f *FactoryImpl) ControllerRuntimeClientProvider() ControllerRuntimeClientProvider {
	return NewControllerRuntimeClientProvider()
}

// ClientGoClientProvider returns a client-go client provider to get client-go clients for a given kubeconfig
func (f *FactoryImpl) ClientGoClientProvider() ClientGoClientProvider {
	return NewClientGoClientProvider()
}
