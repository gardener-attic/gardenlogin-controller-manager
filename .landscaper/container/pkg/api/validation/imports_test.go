// Copyright (c) 2021 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file
// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package validation_test

import (
	"github.com/gardener/gardenlogin-controller-manager/.landscaper/container/pkg/api"
	. "github.com/gardener/gardenlogin-controller-manager/.landscaper/container/pkg/api/validation"
	"github.com/gardener/gardenlogin-controller-manager/.landscaper/container/pkg/test"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gstruct"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"k8s.io/utils/pointer"
)

var _ = Describe("Imports", func() {
	Describe("#ValidateImports - multiClusterDeployment", func() {
		var (
			obj *api.Imports
		)

		BeforeEach(func() {
			runtimeClusterTarget, err := test.NewKubernetesClusterTarget(pointer.StringPtr("foo1"), nil)
			Expect(err).ToNot(HaveOccurred())

			applicationClusterTarget, err := test.NewKubernetesClusterTarget(pointer.StringPtr("foo2"), nil)
			Expect(err).ToNot(HaveOccurred())

			obj = &api.Imports{
				RuntimeClusterTarget:           *runtimeClusterTarget,
				ApplicationClusterTarget:       *applicationClusterTarget,
				MultiClusterDeploymentScenario: true,
				Namespace:                      "foo",
				NamePrefix:                     "bar",
			}
		})

		It("should pass for a valid configuration", func() {
			Expect(ValidateImports(obj)).To(BeEmpty())
		})

		It("should require runtimeCluster and applicationCluster target to be set", func() {
			obj.RuntimeClusterTarget = lsv1alpha1.Target{}
			obj.ApplicationClusterTarget = lsv1alpha1.Target{}

			Expect(ValidateImports(obj)).To(ConsistOf(
				PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeRequired),
					"Field": Equal("runtimeClusterTarget.spec.configuration"),
				})),
				PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeRequired),
					"Field": Equal("applicationClusterTarget.spec.configuration"),
				})),
				PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeRequired),
					"Field": Equal("runtimeClusterTarget.spec.type"),
				})),
				PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeRequired),
					"Field": Equal("applicationClusterTarget.spec.type"),
				})),
			))
		})

		It("should fail for missing single cluster configuration", func() {
			obj.MultiClusterDeploymentScenario = false

			Expect(ValidateImports(obj)).To(ConsistOf(
				PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeRequired),
					"Field": Equal("singleClusterTarget.spec.configuration"),
				})),
				PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeRequired),
					"Field": Equal("singleClusterTarget.spec.type"),
				})),
				PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeForbidden),
					"Field": Equal("runtimeClusterTarget.spec.configuration"),
				})),
				PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeForbidden),
					"Field": Equal("applicationClusterTarget.spec.configuration"),
				})),
			))
		})

		It("should fail for garden namespace", func() {
			obj.Namespace = "garden"

			Expect(ValidateImports(obj)).To(ConsistOf(
				PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeForbidden),
					"Field": Equal("namespace"),
				})),
			))
		})

		It("should fail for namespace with garden- prefix", func() {
			obj.Namespace = "garden-foo"

			Expect(ValidateImports(obj)).To(ConsistOf(
				PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeForbidden),
					"Field": Equal("namespace"),
				})),
			))
		})

		It("should fail for wrong target type", func() {
			obj.ApplicationClusterTarget.Spec.Type = "foo"

			Expect(ValidateImports(obj)).To(ConsistOf(
				PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeForbidden),
					"Field": Equal("applicationClusterTarget.spec.type"),
				})),
			))
		})

		It("should fail if target secretRef is set", func() {
			target, err := test.NewKubernetesClusterTarget(nil, &lsv1alpha1.SecretReference{
				ObjectReference: lsv1alpha1.ObjectReference{
					Name:      "foo",
					Namespace: "bar",
				},
				Key: "foo",
			})
			Expect(err).ToNot(HaveOccurred())
			obj.ApplicationClusterTarget = *target

			Expect(ValidateImports(obj)).To(ConsistOf(
				PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeRequired),
					"Field": Equal("applicationClusterTarget.spec.configuration.kubeconfig"),
				})),
				PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeForbidden),
					"Field": Equal("applicationClusterTarget.spec.configuration.kubeconfig.secretRef"),
				})),
			))
		})

		It("should fail if target configuration is invalid", func() {
			obj.ApplicationClusterTarget.Spec.Configuration = lsv1alpha1.NewAnyJSON([]byte("invalid-config"))

			Expect(ValidateImports(obj)).To(ConsistOf(
				PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeInvalid),
					"Field": Equal("applicationClusterTarget.spec.configuration"),
				})),
			))
		})
	})
})
