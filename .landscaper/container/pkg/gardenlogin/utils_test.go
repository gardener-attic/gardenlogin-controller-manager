// Copyright (c) 2021 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file
// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package gardenlogin

import (
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Utils", func() {
	Describe("#certificateNeedsRenewal", func() {
		var (
			notBefore time.Time
			notAfter  time.Time
			now       time.Time

			validityPercentage float64
		)
		BeforeEach(func() {
			notBefore = getTime("2017-01-01T00:00:00.000Z")
			notAfter = getTime("2017-01-10T23:59:59.000Z")
			now = getTime("2017-01-08T00:00:01.000Z")
			validityPercentage = 0.8 // when 80% of the validity is elapsed the certificate should be renewed
		})

		It("should require certificate renewal", func() {
			notBefore = getTime("2017-01-01T00:00:00.000Z")
			notAfter = getTime("2017-01-01T00:00:10.000Z")

			now = getTime("2017-01-01T00:00:07.000Z")
			Expect(certificateNeedsRenewal(notBefore, notAfter, now, validityPercentage)).To(BeFalse())

			now = getTime("2017-01-01T00:00:08.000Z")
			Expect(certificateNeedsRenewal(notBefore, notAfter, now, validityPercentage)).To(BeTrue())

			validityPercentage = 1 // complete validity range is used - 100%
			Expect(certificateNeedsRenewal(notBefore, notAfter, now, validityPercentage)).To(BeFalse())
		})

		It("should require certificate renewal for expired certificate", func() {
			now = getTime("2017-01-11T00:00:00.000Z")
			Expect(certificateNeedsRenewal(notBefore, notAfter, now, validityPercentage)).To(BeTrue())
		})

		It("should require certificate renewal for not yet valid certificate", func() {
			now = getTime("2016-12-31T00:00:00.000Z")
			Expect(certificateNeedsRenewal(notBefore, notAfter, now, validityPercentage)).To(BeTrue())
		})
	})
})

func getTime(s string) time.Time {
	t, err := time.Parse(time.RFC3339, s)
	Expect(err).ToNot(HaveOccurred())
	return t
}
