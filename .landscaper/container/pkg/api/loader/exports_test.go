// Copyright (c) 2021 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file
// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package loader_test

import (
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/gardener/gardenlogin-controller-manager/.landscaper/container/pkg/api"
	. "github.com/gardener/gardenlogin-controller-manager/.landscaper/container/pkg/api/loader"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Exports", func() {
	Describe("#ExportsToAndFromFile", func() {
		It("should fail because the path does not exist", func() {
			_, err := ExportsFromFile("does-not-exist")
			Expect(err).To(HaveOccurred())
			Expect(err).To(BeAssignableToTypeOf(&os.PathError{}))
		})

		Context("should succeed", func() {
			var (
				dir string
				err error
			)

			BeforeEach(func() {
				dir, err = ioutil.TempDir("", "test-exports")
				Expect(err).NotTo(HaveOccurred())
			})

			AfterEach(func() {
				os.RemoveAll(dir)
			})

			It("should succeed writing and reading", func() {
				path := filepath.Join(dir, "imports.yaml")
				exports := &api.Exports{}

				err := ExportsToFile(exports, path)
				Expect(err).To(BeNil())

				readExports, err := ExportsFromFile(path)
				Expect(err).To(BeNil())
				Expect(readExports).To(BeEquivalentTo(exports))
			})
		})
	})
})
