// Copyright (c) 2021 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file
// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package util

import (
	"time"

	"github.com/gardener/gardener/pkg/utils/secrets"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Utils", func() {
	Describe("#CertificateNeedsRenewal", func() {
		var (
			notBefore time.Time
			validity  time.Duration

			caCert *secrets.Certificate

			validityPercentage float64
		)
		BeforeEach(func() {
			notBefore = getTime("2017-01-01T00:00:00.000Z")
			validity = 10 * time.Second
			validityPercentage = 0.8 // when 80% of the validity is elapsed the certificate should be renewed

			caCert = generateCaCert()
		})

		Context("within validity threshold", func() {
			It("should not require certificate renewal", func() {
				now := notBefore.Add(7 * time.Second) // within 80% validity threshold
				cert := generateClientCert(caCert, notBefore, validity).Certificate
				Expect(CertificateNeedsRenewal(cert, now, validityPercentage)).To(BeFalse())

				validityPercentage = 1                // complete validity range is used - 100%
				now = notBefore.Add(10 * time.Second) // within 100% validity threshold
				Expect(CertificateNeedsRenewal(cert, now, validityPercentage)).To(BeFalse())
			})
		})

		Context("not within validity threshold", func() {
			It("should require certificate renewal", func() {
				now := notBefore.Add(9 * time.Second) // not within 80% validity threshold
				cert := generateClientCert(caCert, notBefore, validity).Certificate
				Expect(CertificateNeedsRenewal(cert, now, validityPercentage)).To(BeTrue())

			})
		})

		Context("not valid certificate", func() {
			It("should require certificate renewal for expired certificate", func() {
				now := notBefore.Add(validity + 1*time.Second)
				cert := generateClientCert(caCert, notBefore, validity).Certificate
				Expect(CertificateNeedsRenewal(cert, now, validityPercentage)).To(BeTrue())
			})

			It("should require certificate renewal for not yet valid certificate", func() {
				notBefore = getTime("2017-01-01T00:00:00.000Z")
				now := getTime("2016-01-01T00:00:00.000Z")
				cert := generateClientCert(caCert, notBefore, validity).Certificate
				Expect(CertificateNeedsRenewal(cert, now, validityPercentage)).To(BeTrue())
			})
		})
	})
})

func generateClientCert(caCert *secrets.Certificate, notBefore time.Time, validity time.Duration) *secrets.Certificate {
	csc := &secrets.CertificateSecretConfig{
		Name:       "foo",
		CommonName: "foo",
		CertType:   secrets.ClientCert,
		Validity:   &validity,
		SigningCA:  caCert,
		Now: func() time.Time {
			return notBefore
		},
	}
	cert, err := csc.GenerateCertificate()
	Expect(err).ToNot(HaveOccurred())

	return cert
}

func generateCaCert() *secrets.Certificate {
	csc := &secrets.CertificateSecretConfig{
		Name:       "ca-test",
		CommonName: "ca-test",
		CertType:   secrets.CACert,
	}
	caCertificate, err := csc.GenerateCertificate()
	Expect(err).ToNot(HaveOccurred())

	return caCertificate
}

func getTime(s string) time.Time {
	t, err := time.Parse(time.RFC3339, s)
	Expect(err).ToNot(HaveOccurred())

	return t
}
