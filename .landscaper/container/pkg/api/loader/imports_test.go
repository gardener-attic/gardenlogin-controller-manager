// Copyright (c) 2021 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file
// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package loader_test

import (
	"io/ioutil"
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	. "github.com/gardener/gardenlogin-controller-manager/.landscaper/container/pkg/api/loader"
)

var _ = Describe("Imports", func() {
	Describe("#ImportsFromFile", func() {
		It("should fail because the path does not exist", func() {
			_, err := ImportsFromFile("does-not-exist")
			Expect(err).To(HaveOccurred())
			Expect(err).To(BeAssignableToTypeOf(&os.PathError{}))
		})

		Context("successful read", func() {
			var (
				dir string
				err error
			)

			BeforeEach(func() {
				dir, err = ioutil.TempDir("", "test-imports")
				Expect(err).NotTo(HaveOccurred())
			})

			AfterEach(func() {
				Expect(os.RemoveAll(dir)).To(Succeed())
			})

			It("should succeed reading but fail parsing the file", func() {
				path := filepath.Join(dir, "imports.yaml")
				Expect(ioutil.WriteFile(path, []byte("foo"), 0644)).To(Succeed())

				imports, err := ImportsFromFile(path)
				Expect(err).To(HaveOccurred())
				Expect(imports).To(BeNil())
			})
		})
	})
})
