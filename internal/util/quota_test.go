/*
SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors

SPDX-License-Identifier: Apache-2.0
*/

package util_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"

	"github.com/gardener/gardenlogin-controller-manager/internal/util"
)

var _ = Describe("Quota", func() {

	DescribeTable("x less than y",
		func(x corev1.ResourceList, y corev1.ResourceList, expected bool) {
			Expect(util.LessThan(x, y)).To(Equal(expected))
		},
		Entry("x > y", corev1.ResourceList{"foo/bar": resource.MustParse("2")}, corev1.ResourceList{"foo/bar": resource.MustParse("1")}, false),
		Entry("x == y", corev1.ResourceList{"foo/bar": resource.MustParse("1")}, corev1.ResourceList{"foo/bar": resource.MustParse("1")}, false),
		Entry("x < y", corev1.ResourceList{"foo/bar": resource.MustParse("1")}, corev1.ResourceList{"foo/bar": resource.MustParse("2")}, true),
		Entry("x has different resource name than y", corev1.ResourceList{"foo/bar": resource.MustParse("1")}, corev1.ResourceList{"bar/baz": resource.MustParse("2")}, true),
	)
})
