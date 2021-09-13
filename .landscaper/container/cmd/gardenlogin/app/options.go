// Copyright (c) 2021 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file
// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package app

import (
	"fmt"
	"os"

	"github.com/gardener/landscaper/apis/deployer/container"
)

// Options has all the context and parameters needed to run the gardenlogin deployer.
type Options struct {
	// OperationType is the operation to be executed.
	OperationType container.OperationType
	// ImportsPath is the path to the imports file.
	ImportsPath string
	// ContentPath is the path to the content directory containing a copy of all blueprint data.
	ContentPath string
	// ComponentDescriptorPath is the path to the component descriptor file.
	ComponentDescriptorPath string
}

// NewOptions returns a new options structure.
func NewOptions() *Options {
	return &Options{OperationType: container.OperationReconcile}
}

// InitializeFromEnvironment initializes the options from the found environment variables.
func (o *Options) InitializeFromEnvironment() {
	if op := os.Getenv("OPERATION"); len(op) > 0 {
		o.OperationType = container.OperationType(op)
	}

	o.ImportsPath = os.Getenv(container.ImportsPathName)
	o.ContentPath = os.Getenv(container.ContentPathName)

	o.ComponentDescriptorPath = os.Getenv(container.ComponentDescriptorPathName)
}

// validate validates all the required options.
func (o *Options) validate(args []string) error {
	if o.OperationType != container.OperationReconcile && o.OperationType != container.OperationDelete {
		return fmt.Errorf("operation must be %q or %q", container.OperationReconcile, container.OperationDelete)
	}

	if len(o.ImportsPath) == 0 {
		return fmt.Errorf("missing path for imports file")
	}

	if len(o.ContentPath) == 0 {
		return fmt.Errorf("missing path for content data")
	}

	if len(o.ComponentDescriptorPath) == 0 {
		return fmt.Errorf("missing path for component descriptor file")
	}

	return nil
}
