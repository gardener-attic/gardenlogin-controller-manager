// Copyright (c) 2021 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file
// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package gardenlogin

import (
	"github.com/gardener/gardenlogin-controller-manager/.landscaper/container/internal/fake"
	"github.com/gardener/gardenlogin-controller-manager/.landscaper/container/pkg/api"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/sirupsen/logrus"
)

var _ = Describe("Operation", func() {
	Describe("#NewOperation", func() {
		It("should return the correct operation object", func() {
			fakeClock, err := fake.NewFakeClock()
			Expect(err).ToNot(HaveOccurred())
			var (
				log       = logrus.New()
				clock     = fakeClock
				imports   = &api.Imports{} // TODO targets
				imageRefs = &api.ImageRefs{
					GardenloginImage:   "",
					KubeRbacProxyImage: "",
				}
				contents = api.Contents{}
			)

			operationInterface, err := NewOperation(log, clock, imports, imageRefs, contents)
			Expect(err).NotTo(HaveOccurred())

			op, ok := operationInterface.(*operation)
			Expect(ok).To(BeTrue())
			Expect(op.multiCluster).To(BeNil())
			Expect(op.singleCluster).To(BeNil())
			Expect(op.log).To(Equal(log))
			Expect(op.clock).To(Equal(clock))
			Expect(op.imports).To(Equal(imports))
			Expect(op.contents).To(Equal(contents))
		})
	})
})
