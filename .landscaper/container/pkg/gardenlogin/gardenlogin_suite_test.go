// Copyright (c) 2021 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file
// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package gardenlogin_test

import (
	"testing"

	"github.com/gardener/gardenlogin-controller-manager/.landscaper/container/pkg/test"

	gardenerkubernetes "github.com/gardener/gardener/pkg/client/kubernetes"
	gardenenvtest "github.com/gardener/gardener/pkg/envtest"
	"github.com/gardener/gardener/test/framework"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func TestGardenlogin(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Gardenlogin Integration Test Suite")
}

var (
	testEnv    *gardenenvtest.GardenerTestEnvironment
	restConfig *rest.Config

	testClient     client.Client
	testKubeClient kubernetes.Interface

	kubeconfig []byte
)

var _ = BeforeSuite(func() {
	By("starting test environment")
	testEnv = &gardenenvtest.GardenerTestEnvironment{
		GardenerAPIServer: &gardenenvtest.GardenerAPIServer{
			Args: []string{"--disable-admission-plugins=ResourceReferenceManager,ExtensionValidator,ShootQuotaValidator,ShootValidator,ShootTolerationRestriction"},
		},
	}
	var err error
	restConfig, err = testEnv.Start()
	Expect(err).ToNot(HaveOccurred())

	testClient, err = client.New(restConfig, client.Options{Scheme: gardenerkubernetes.GardenScheme})
	Expect(err).ToNot(HaveOccurred())
	testKubeClient, err = kubernetes.NewForConfig(restConfig)
	Expect(err).ToNot(HaveOccurred())

	kubeconfig, err = test.KubeconfigFromRestConfig(restConfig)
	Expect(err).ToNot(HaveOccurred())
})

var _ = AfterSuite(func() {
	By("running cleanup actions")
	framework.RunCleanupActions()

	By("stopping test environment")
	Expect(testEnv.Stop()).To(Succeed())
})
