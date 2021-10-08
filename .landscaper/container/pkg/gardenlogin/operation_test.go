// Copyright (c) 2021 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file
// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package gardenlogin

import (
	"io/ioutil"

	"github.com/gardener/gardenlogin-controller-manager/.landscaper/container/internal/fake"
	"github.com/gardener/gardenlogin-controller-manager/.landscaper/container/pkg/api"
	"github.com/gardener/gardenlogin-controller-manager/.landscaper/container/pkg/test"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/sirupsen/logrus"
	clientgofake "k8s.io/client-go/kubernetes/fake"
	fakeclient "sigs.k8s.io/controller-runtime/pkg/client/fake"
)

var _ = Describe("Operation", func() {
	var (
		f *fake.Factory

		log *logrus.Logger

		imports   *api.Imports
		imageRefs *api.ImageRefs
		contents  *api.Contents
	)

	BeforeEach(func() {
		var err error

		f, err = fake.NewFakeFactory(nil, nil, nil)
		Expect(err).ToNot(HaveOccurred())

		log = logrus.New()

		imports = &api.Imports{}
		imageRefs = &api.ImageRefs{}
		contents = &api.Contents{}
	})

	Describe("#NewOperation", func() {
		It("should return the correct operation object for multicluster deployment", func() {
			runtimeClient := fakeclient.NewClientBuilder().Build()
			applicationClient := fakeclient.NewClientBuilder().Build()

			runtimeKubeClient := clientgofake.NewSimpleClientset()
			applicationKubeClient := clientgofake.NewSimpleClientset()

			runtimeKubeconfig := "runtime"
			applicationKubeconfig := "application"

			f.WithControllerRuntimeClient(runtimeKubeconfig, runtimeClient)
			f.WithClientGoClient(runtimeKubeconfig, runtimeKubeClient)

			f.WithControllerRuntimeClient(applicationKubeconfig, applicationClient)
			f.WithClientGoClient(applicationKubeconfig, applicationKubeClient)

			imports.MultiClusterDeploymentScenario = true
			target, err := test.NewKubernetesClusterTarget(&runtimeKubeconfig, nil)
			Expect(err).ToNot(HaveOccurred())
			imports.RuntimeClusterTarget = *target

			target, err = test.NewKubernetesClusterTarget(&applicationKubeconfig, nil)
			Expect(err).ToNot(HaveOccurred())
			imports.ApplicationClusterTarget = *target

			operationInterface, err := NewOperation(f, log, imports, imageRefs, contents)
			Expect(err).NotTo(HaveOccurred())

			op, ok := operationInterface.(*operation)
			Expect(ok).To(BeTrue())

			Expect(op.multiCluster.applicationCluster.client).To(Equal(applicationClient))
			Expect(op.multiCluster.runtimeCluster.client).To(Equal(runtimeClient))

			Expect(ioutil.ReadFile(op.multiCluster.runtimeCluster.kubeconfig)).To(Equal([]byte(runtimeKubeconfig)))
			Expect(ioutil.ReadFile(op.multiCluster.applicationCluster.kubeconfig)).To(Equal([]byte(applicationKubeconfig)))

			Expect(op.singleCluster).To(BeNil())
			Expect(op.log).To(Equal(log))
			Expect(op.clock).To(Equal(f.Clock()))
			Expect(op.imports).To(Equal(imports))
			Expect(op.contents).To(Equal(contents))
		})

		It("should return the correct operation object for single-cluster deployment", func() {
			client := fakeclient.NewClientBuilder().Build()

			kubeClient := clientgofake.NewSimpleClientset()

			kubeconfig := "single-cluster"

			f.WithControllerRuntimeClient(kubeconfig, client)
			f.WithClientGoClient(kubeconfig, kubeClient)

			imports.MultiClusterDeploymentScenario = false
			target, err := test.NewKubernetesClusterTarget(&kubeconfig, nil)
			Expect(err).ToNot(HaveOccurred())
			imports.SingleClusterTarget = *target

			operationInterface, err := NewOperation(f, log, imports, imageRefs, contents)
			Expect(err).NotTo(HaveOccurred())

			op, ok := operationInterface.(*operation)
			Expect(ok).To(BeTrue())

			Expect(op.multiCluster).To(BeNil())

			Expect(op.singleCluster.client).To(Equal(client))
			Expect(op.singleCluster.kubernetes).To(Equal(kubeClient))

			Expect(ioutil.ReadFile(op.singleCluster.kubeconfig)).To(Equal([]byte(kubeconfig)))
			Expect(op.log).To(Equal(log))
			Expect(op.clock).To(Equal(f.Clock()))
			Expect(op.imports).To(Equal(imports))
			Expect(op.contents).To(Equal(contents))
		})
	})
})
