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

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	"github.com/ghodss/yaml"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	apiextensionsv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	kubernetesscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/component-base/version/verflag"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// OperationType is a string alias.
type OperationType string

const (
	// OperationTypeReconcile is a  constant for the RECONCILE operation type.
	OperationTypeReconcile OperationType = "RECONCILE"
	// OperationTypeDelete is a constant for the DELETE operation type.
	OperationTypeDelete OperationType = "DELETE"
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

			if err := run(cmd.Context(), log, opts); err != nil {
				panic(err)
			}

			log.Infof("Execution finished successfully.")
		},
	}

	verflag.AddFlags(cmd.Flags())
	opts.AddFlags(cmd.Flags())
	return cmd
}

// run runs the virtual garden deployer.
func run(ctx context.Context, log *logrus.Logger, opts *Options) error {
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

	state := api.NewStateFromPath(opts.StatePath)

	log.Infof("Creating REST config and Kubernetes client based on given kubeconfig for the runtime cluster")
	runtimeClient, err := NewClientFromTarget(imports.RuntimeClusterTarget)
	if err != nil {
		return err
	}

	log.Infof("Creating REST config and Kubernetes client based on given kubeconfig for the application cluster")
	applicationClient, err := NewClientFromTarget(imports.ApplicationClusterTarget)
	if err != nil {
		return err
	}

	operation := gardenlogin.NewOperation(runtimeClient, applicationClient, log, imports, imageRefs, contents, state)

	if opts.OperationType == OperationTypeReconcile {
		exports, err := operation.Reconcile(ctx)
		if err != nil {
			return err
		}

		log.Infof("Writing exports file to EXPORTS_PATH(%s)", opts.ExportsPath)
		err = loader.ExportsToFile(exports, opts.ExportsPath)
		if err != nil {
			return err
		}

		return nil
	} else if opts.OperationType == OperationTypeDelete {
		return operation.Delete(ctx)
	}
	return fmt.Errorf("unknown operation type: %q", opts.OperationType)
}

// NewClientFromTarget creates a new Kubernetes client for the kubeconfig in the given target.
func NewClientFromTarget(target lsv1alpha1.Target) (client.Client, error) {
	targetConfig := target.Spec.Configuration.RawMessage
	targetConfigMap := make(map[string]string)

	err := yaml.Unmarshal(targetConfig, &targetConfigMap)
	if err != nil {
		return nil, err
	}

	kubeconfig, ok := targetConfigMap["kubeconfig"]
	if !ok {
		return nil, fmt.Errorf("Imported target does not contain a kubeconfig")
	}

	return NewClientFromKubeconfig([]byte(kubeconfig))
}

// NewClientFromKubeconfig creates a new Kubernetes client for the given kubeconfig.
func NewClientFromKubeconfig(kubeconfig []byte) (client.Client, error) {
	clientConfig, err := clientcmd.NewClientConfigFromBytes(kubeconfig)
	if err != nil {
		return nil, err
	}

	restConfig, err := clientConfig.ClientConfig()
	if err != nil {
		return nil, err
	}

	scheme := runtime.NewScheme()
	utilruntime.Must(kubernetesscheme.AddToScheme(scheme))
	utilruntime.Must(apiextensionsv1beta1.AddToScheme(scheme))

	return client.New(restConfig, client.Options{Scheme: scheme})
}
