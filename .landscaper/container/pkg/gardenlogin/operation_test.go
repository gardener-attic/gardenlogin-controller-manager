// Copyright (c) 2021 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file
// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package gardenlogin

import (
	"github.com/gardener/gardenlogin-controller-manager/.landscaper/container/pkg/api"
	mockclient "github.com/gardener/gardenlogin-controller-manager/.landscaper/container/pkg/mock/controller-runtime/client"
	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/sirupsen/logrus"
)

var _ = Describe("Operation", func() {
	Describe("#NewOperation", func() {
		It("should return the correct operation object", func() {
			var (
				runtimeClient     = mockclient.NewMockClient(gomock.NewController(GinkgoT()))
				applicationClient = mockclient.NewMockClient(gomock.NewController(GinkgoT()))
				log               = logrus.New()
				imports           = &api.Imports{}
				imageRefs         = &api.ImageRefs{
					GardenloginImage: "",
					AuthProxyImage:   "",
				}
				state = api.State{}
			)

			operationInterface := NewOperation(runtimeClient, applicationClient, log, imports, imageRefs, state)

			op, ok := operationInterface.(*operation)
			Expect(ok).To(BeTrue())
			Expect(op.runtimeClient).To(Equal(runtimeClient))
			Expect(op.applicationClient).To(Equal(applicationClient))
			Expect(op.log).To(Equal(log))
			Expect(op.imports).To(Equal(imports))
		})
	})
})
