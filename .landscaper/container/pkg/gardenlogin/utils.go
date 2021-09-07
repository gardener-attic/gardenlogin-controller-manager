// Copyright (c) 2021 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file
// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package gardenlogin

import (
	"crypto/x509"
	"time"
)

// certificateNeedsRenewal returns true in case the certificate is not (yet) valid or in case the given validityThresholdPercentage is exceeded.
// A validityThresholdPercentage lower than 100% should be given in case the certificate should be renewed well in advance before the certificate expires.
func certificateNeedsRenewal(certificate *x509.Certificate, now time.Time, validityThresholdPercentage float64) bool {
	notBefore := certificate.NotBefore.UTC()
	notAfter := certificate.NotAfter.UTC()

	validNotBefore := now.After(notBefore) || now.Equal(notBefore)
	validNotAfter := now.Before(notAfter) || now.Equal(notAfter)

	isValid := validNotBefore && validNotAfter
	if !isValid {
		return true
	}

	validityTimespan := notAfter.Sub(notBefore).Seconds()
	elapsedValidity := now.Sub(notBefore).Seconds()

	validityThreshold := validityTimespan * validityThresholdPercentage

	return elapsedValidity > validityThreshold
}
