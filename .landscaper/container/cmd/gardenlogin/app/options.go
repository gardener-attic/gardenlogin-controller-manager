// Copyright (c) 2021 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file
// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package app

import (
	"fmt"
	"os"

	"github.com/spf13/pflag"
)

// Options has all the context and parameters needed to run the virtual garden deployer.
type Options struct {
	// OperationType is the operation to be executed.
	OperationType OperationType
	// ImportsPath is the path to the imports file.
	ImportsPath string
	// ExportsPath is the path to the exports file. The parent directory exists; the export file itself must be created.
	// The format of the exports file must be json or yaml.
	ExportsPath string
	// StatePath is the path to the state directory.
	StatePath string
	// ComponentDescriptorPath is the path to the component descriptor file.
	ComponentDescriptorPath string
}

// NewOptions returns a new options structure.
func NewOptions() *Options {
	return &Options{OperationType: OperationTypeReconcile}
}

// AddFlags adds flags for a specific Scheduler to the specified FlagSet.
func (o *Options) AddFlags(fs *pflag.FlagSet) {
}

// InitializeFromEnvironment initializes the options from the found environment variables.
func (o *Options) InitializeFromEnvironment() {
	if op := os.Getenv("OPERATION"); len(op) > 0 {
		o.OperationType = OperationType(op)
	}
	o.ImportsPath = os.Getenv("IMPORTS_PATH")
	o.ExportsPath = os.Getenv("EXPORTS_PATH")
	o.StatePath = os.Getenv("STATE_PATH")

	o.ComponentDescriptorPath = os.Getenv("COMPONENT_DESCRIPTOR_PATH")
}

// validate validates all the required options.
func (o *Options) validate(args []string) error {
	if o.OperationType != OperationTypeReconcile && o.OperationType != OperationTypeDelete {
		return fmt.Errorf("operation must be %q or %q", OperationTypeReconcile, OperationTypeDelete)
	}

	if len(o.ImportsPath) == 0 {
		return fmt.Errorf("missing path for imports file")
	}

	if len(o.ExportsPath) == 0 {
		return fmt.Errorf("missing path for exports file")
	}

	if len(o.ComponentDescriptorPath) == 0 {
		return fmt.Errorf("missing path for component descriptor file")
	}

	return nil
}
