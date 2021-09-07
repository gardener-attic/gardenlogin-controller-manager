// Copyright (c) 2021 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file
// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package app

import (
	"context"
	"fmt"
	"os"

	"github.com/gardener/gardenlogin-controller-manager/.landscaper/container/pkg/api"
	"github.com/gardener/gardenlogin-controller-manager/.landscaper/container/pkg/api/loader"
	"github.com/gardener/gardenlogin-controller-manager/.landscaper/container/pkg/api/validation"
	"github.com/gardener/gardenlogin-controller-manager/.landscaper/container/pkg/gardenlogin"

	"github.com/gardener/landscaper/apis/deployer/container"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/component-base/version/verflag"
)

// NewCommandVirtualGarden creates a *cobra.Command object with default parameters.
func NewCommandVirtualGarden() *cobra.Command {
	opts := NewOptions()

	cmd := &cobra.Command{
		Use:   "gardenlogin-controller-manager",
		Short: "Launch the gardenlogin-controller-manager deployer",
		Long:  `The virtual garden deployer deploys a virtual garden cluster into a hosting cluster.`,
		Run: func(cmd *cobra.Command, args []string) {
			verflag.PrintAndExitIfRequested()

			opts.InitializeFromEnvironment()
			utilruntime.Must(opts.validate(args))

			log := &logrus.Logger{
				Out:   os.Stderr,
				Level: logrus.InfoLevel,
				Formatter: &logrus.TextFormatter{
					DisableColors: true,
				},
			}

			cmd.Flags().VisitAll(func(flag *pflag.Flag) {
				log.Infof("FLAG: --%s=%s", flag.Name, flag.Value)
			})

			clock := gardenlogin.RealClock{}

			if err := run(cmd.Context(), log, clock, opts); err != nil {
				panic(err)
			}

			log.Infof("Execution finished successfully.")
		},
	}

	verflag.AddFlags(cmd.Flags())
	opts.AddFlags(cmd.Flags())

	return cmd
}

// run runs the gardenlogin deployer.
func run(ctx context.Context, log *logrus.Logger, clock gardenlogin.Clock, opts *Options) error {
	log.Infof("Reading imports file from %s", opts.ImportsPath)

	imports, err := loader.ImportsFromFile(opts.ImportsPath)
	if err != nil {
		return err
	}

	log.Infof("Reading component descriptor file from %s", opts.ComponentDescriptorPath)

	cd, err := loader.ComponentDescriptorFromFile(opts.ComponentDescriptorPath)
	if err != nil {
		return err
	}

	imageRefs, err := api.NewImageRefsFromComponentDescriptor(cd)
	if err != nil {
		return err
	}

	log.Infof("Validating imports file")

	if errList := validation.ValidateImports(imports); len(errList) > 0 {
		return errList.ToAggregate()
	}

	log.Infof("Validating content path")

	contents := api.NewContentsFromPath(opts.ContentPath)
	if err := validation.ValidateContents(contents); err != nil {
		return fmt.Errorf("failed to validate contents: %w", err)
	}

	operation, err := gardenlogin.NewOperation(
		log,
		clock,
		imports,
		imageRefs,
		contents,
	)
	if err != nil {
		return err
	}

	if opts.OperationType == container.OperationReconcile {
		return operation.Reconcile(ctx)
	} else if opts.OperationType == container.OperationDelete {
		return operation.Delete(ctx)
	}

	return fmt.Errorf("unknown operation type: %q", opts.OperationType)
}
