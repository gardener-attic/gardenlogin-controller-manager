// Copyright (c) 2021 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file
// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package gardenlogin_test

import (
	"context"
	"fmt"
	"io/ioutil"
	"k8s.io/apimachinery/pkg/api/resource"
	"os"
	"path"

	"github.com/gardener/gardenlogin-controller-manager/.landscaper/container/internal/fake"
	"github.com/gardener/gardenlogin-controller-manager/.landscaper/container/pkg/api"
	"github.com/gardener/gardenlogin-controller-manager/.landscaper/container/pkg/api/validation"
	"github.com/gardener/gardenlogin-controller-manager/.landscaper/container/pkg/gardenlogin"
	"github.com/gardener/gardenlogin-controller-manager/.landscaper/container/pkg/test"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gstruct"
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
			KubeRBACProxyImage: "quay.io/brancz/kube-rbac-proxy:v0.8.0",
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
		var (
			defaultManagerResources   corev1.ResourceRequirements
			defaultRbacProxyResources corev1.ResourceRequirements
		)
		BeforeEach(func() {
			singleClusterTarget, err := test.NewKubernetesClusterTarget(pointer.StringPtr(string(kubeconfig)), nil)
			Expect(err).ToNot(HaveOccurred())

			imports.MultiClusterDeploymentScenario = false
			imports.SingleClusterTarget = *singleClusterTarget

			defaultManagerResources = corev1.ResourceRequirements{
				Limits: corev1.ResourceList{
					corev1.ResourceCPU:    resource.MustParse("200m"),
					corev1.ResourceMemory: resource.MustParse("300Mi"),
				},
				Requests: corev1.ResourceList{
					corev1.ResourceCPU:    resource.MustParse("100m"),
					corev1.ResourceMemory: resource.MustParse("100Mi"),
				},
			}

			defaultRbacProxyResources = corev1.ResourceRequirements{
				Limits: corev1.ResourceList{
					corev1.ResourceCPU:    resource.MustParse("100m"),
					corev1.ResourceMemory: resource.MustParse("30Mi"),
				},
				Requests: corev1.ResourceList{
					corev1.ResourceCPU:    resource.MustParse("100m"),
					corev1.ResourceMemory: resource.MustParse("20Mi"),
				},
			}
		})

		It("should create and delete gardenlogin-controller-manager resources", func() {
			op, err = gardenlogin.NewOperation(f, log, imports, imageRefs, contents)
			Expect(err).NotTo(HaveOccurred())

			By("running reconcile op")
			Expect(op.Reconcile(ctx)).NotTo(HaveOccurred())

			By("verifying that resources were created")

			namespaceKey := client.ObjectKey{Name: imports.Namespace}
			namespace := &corev1.Namespace{}
			Expect(testClient.Get(ctx, namespaceKey, namespace)).To(Succeed())

			deploymentKey := client.ObjectKey{Namespace: imports.Namespace, Name: imports.NamePrefix + "controller-manager"}
			deployment := &appsv1.Deployment{}
			Expect(testClient.Get(ctx, deploymentKey, deployment)).To(Succeed())

			containerIdentifier := func(element interface{}) string {
				container, ok := element.(corev1.Container)
				Expect(ok).To(BeTrue())
				return container.Name
			}

			By("verifying default resource requirements")
			Expect(deployment.Spec.Template.Spec.Containers).To(MatchAllElements(containerIdentifier, Elements{
				"manager": MatchFields(IgnoreExtras, Fields{
					"Resources": Equal(defaultManagerResources),
				}),
				"kube-rbac-proxy": MatchFields(IgnoreExtras, Fields{
					"Resources": Equal(defaultRbacProxyResources),
				}),
			}))

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

		It("should patch resource requirements", func() {
			patchedManagerResources := corev1.ResourceRequirements{
				Limits: corev1.ResourceList{
					corev1.ResourceCPU:    resource.MustParse("250m"),
					corev1.ResourceMemory: resource.MustParse("350Mi"),
				},
				Requests: corev1.ResourceList{
					corev1.ResourceCPU:    resource.MustParse("150m"),
					corev1.ResourceMemory: resource.MustParse("150Mi"),
				},
			}
			patchedRbacProxyResources := corev1.ResourceRequirements{
				Limits: corev1.ResourceList{
					corev1.ResourceCPU:    resource.MustParse("150m"),
					corev1.ResourceMemory: resource.MustParse("35Mi"),
				},
				Requests: corev1.ResourceList{
					corev1.ResourceCPU:    resource.MustParse("150m"),
					corev1.ResourceMemory: resource.MustParse("25Mi"),
				},
			}

			imports.Gardenlogin.ManagerResources = patchedManagerResources
			imports.Gardenlogin.KubeRBACProxyResources = patchedRbacProxyResources

			op, err = gardenlogin.NewOperation(f, log, imports, imageRefs, contents)
			Expect(err).NotTo(HaveOccurred())

			By("running reconcile op")
			Expect(op.Reconcile(ctx)).NotTo(HaveOccurred())

			By("verifying that resources were created")

			namespaceKey := client.ObjectKey{Name: imports.Namespace}
			namespace := &corev1.Namespace{}
			Expect(testClient.Get(ctx, namespaceKey, namespace)).To(Succeed())

			deploymentKey := client.ObjectKey{Namespace: imports.Namespace, Name: imports.NamePrefix + "controller-manager"}
			deployment := &appsv1.Deployment{}
			Expect(testClient.Get(ctx, deploymentKey, deployment)).To(Succeed())

			containerIdentifier := func(element interface{}) string {
				container, ok := element.(corev1.Container)
				Expect(ok).To(BeTrue())
				return container.Name
			}

			By("verifying patched resource requirements")
			Expect(deployment.Spec.Template.Spec.Containers).To(MatchAllElements(containerIdentifier, Elements{
				"manager": MatchFields(IgnoreExtras, Fields{
					"Resources": Equal(patchedManagerResources),
				}),
				"kube-rbac-proxy": MatchFields(IgnoreExtras, Fields{
					"Resources": Equal(patchedRbacProxyResources),
				}),
			}))
		})
	})
})
