// Copyright (c) 2021 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file
// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package gardenlogin_test

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path"

	"github.com/gardener/gardenlogin-controller-manager/.landscaper/container/internal/fake"
	"github.com/gardener/gardenlogin-controller-manager/.landscaper/container/pkg/api"
	"github.com/gardener/gardenlogin-controller-manager/.landscaper/container/pkg/api/validation"
	"github.com/gardener/gardenlogin-controller-manager/.landscaper/container/pkg/gardenlogin"
	"github.com/gardener/gardenlogin-controller-manager/.landscaper/container/pkg/test"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/sirupsen/logrus"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/utils/pointer"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/kind/pkg/fs"
)

var _ = Describe("Operation Reconcile", func() {
	var (
		err error
		f   *fake.Factory
		ctx context.Context

		log *logrus.Logger

		imports   *api.Imports
		imageRefs *api.ImageRefs
		contents  *api.Contents

		op gardenlogin.Interface

		tmpDir string
	)

	BeforeEach(func() {
		f, err = fake.NewFakeFactory(nil, nil, nil)
		Expect(err).ToNot(HaveOccurred())

		ctx = f.Context()
		log = logrus.New()

		suffix := test.StringWithCharset(5, nil)

		imports = &api.Imports{
			Namespace:  fmt.Sprintf("test-namespace-%s", suffix),
			NamePrefix: "prefix-",
		}

		imageRefs = &api.ImageRefs{
			GardenloginImage:   "eu.gcr.io/gardener-project/gardener/gardenlogin-controller-manager:latest",
			KubeRbacProxyImage: "quay.io/brancz/kube-rbac-proxy:v0.8.0",
		}

		By("copying config folder for test")
		tmpDir, err = ioutil.TempDir("", "reconcile-tests")
		Expect(err).NotTo(HaveOccurred())
		tmpConfigDir := path.Join(tmpDir, "config")
		Expect(os.Mkdir(tmpConfigDir, 0700)).To(Succeed())
		Expect(fs.Copy(path.Join("..", "..", "..", "blueprint", "config"), tmpConfigDir)).To(Succeed())
		contents = api.NewContentsFromPath(tmpDir)
		Expect(validation.ValidateContents(contents)).To(Succeed())

		f.WithControllerRuntimeClient(string(kubeconfig), testClient)
		f.WithClientGoClient(string(kubeconfig), testKubeClient)
	})

	AfterEach(func() {
		Expect(os.RemoveAll(tmpDir)).To(Succeed())

		Expect(op.Delete(ctx)).To(Succeed())
	})

	Describe("#Single-Cluster reconcile", func() {
		BeforeEach(func() {
			singleClusterTarget, err := test.NewKubernetesClusterTarget(pointer.StringPtr(string(kubeconfig)), nil)
			Expect(err).ToNot(HaveOccurred())

			imports.MultiClusterDeploymentScenario = false
			imports.SingleClusterTarget = *singleClusterTarget
		})

		It("should create and delete gardenlogin-controller-manager resources", func() {
			op, err = gardenlogin.NewOperation(f, log, imports, imageRefs, contents)
			Expect(err).NotTo(HaveOccurred())

			By("running reconcile op")
			Expect(op.Reconcile(ctx)).NotTo(HaveOccurred())

			By("verifying that resources were deleted")

			namespaceKey := client.ObjectKey{Name: imports.Namespace}
			namespace := &corev1.Namespace{}
			Expect(testClient.Get(ctx, namespaceKey, namespace)).To(Succeed())

			deploymentKey := client.ObjectKey{Namespace: imports.Namespace, Name: imports.NamePrefix + "controller-manager"}
			deployment := &appsv1.Deployment{}
			Expect(testClient.Get(ctx, deploymentKey, deployment)).To(Succeed())

			crbKey := client.ObjectKey{Name: fmt.Sprintf("%sproxy-rolebinding", imports.NamePrefix)}
			crb := &rbacv1.ClusterRoleBinding{}
			Expect(testClient.Get(ctx, crbKey, crb)).To(Succeed())

			crKey := client.ObjectKey{Name: fmt.Sprintf("%sproxy-role", imports.NamePrefix)}
			cr := &rbacv1.ClusterRole{}
			Expect(testClient.Get(ctx, crKey, cr)).To(Succeed())

			By("running delete op")
			Expect(op.Delete(ctx)).NotTo(HaveOccurred())

			By("verifying that resources were deleted")
			Eventually(func() bool {
				err = testClient.Get(ctx, namespaceKey, namespace)
				return errors.IsNotFound(err) || !namespace.GetDeletionTimestamp().IsZero()
			}).Should(BeTrue())
			Eventually(func() bool {
				err = testClient.Get(ctx, crbKey, cr)
				return errors.IsNotFound(err) || !cr.GetDeletionTimestamp().IsZero()
			}).Should(BeTrue())
			Eventually(func() bool {
				err = testClient.Get(ctx, crKey, cr)
				return errors.IsNotFound(err) || !cr.GetDeletionTimestamp().IsZero()
			}).Should(BeTrue())
		})

		It("should not error when running delete operation multiple times", func() {
			By("running delete operation")
			op, err = gardenlogin.NewOperation(f, log, imports, imageRefs, contents)
			Expect(err).NotTo(HaveOccurred())

			Expect(op.Delete(ctx)).To(Succeed())
			Expect(op.Delete(ctx)).To(Succeed())
		})

		It("should renew the webhook server certificate", func() {
			op, err = gardenlogin.NewOperation(f, log, imports, imageRefs, contents)
			Expect(err).NotTo(HaveOccurred())

			Expect(op.Reconcile(ctx)).NotTo(HaveOccurred())

			tlsSecretKey := client.ObjectKey{Namespace: imports.Namespace, Name: imports.NamePrefix + gardenlogin.TLSSecretSuffix}

			secretBefore := &corev1.Secret{}
			Expect(testClient.Get(ctx, tlsSecretKey, secretBefore)).To(Succeed())

			By("forwarding time 10 years")
			f.WithTime(f.Clock().Now().AddDate(10, 0, 0))

			By("reconciling again")
			Expect(op.Reconcile(ctx)).NotTo(HaveOccurred())

			secretAfter := &corev1.Secret{}
			Expect(testClient.Get(ctx, tlsSecretKey, secretAfter)).To(Succeed())

			Expect(secretBefore.Data[corev1.TLSCertKey]).NotTo(Equal(secretAfter.Data[corev1.TLSCertKey]))
			Expect(secretBefore.Data[corev1.TLSPrivateKeyKey]).NotTo(Equal(secretAfter.Data[corev1.TLSPrivateKeyKey]))
		})
	})
})
