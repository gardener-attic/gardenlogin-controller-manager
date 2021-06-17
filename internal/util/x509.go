/*
SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors

SPDX-License-Identifier: Apache-2.0
*/

package util

import (
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
)

// ValidateCertificate takes a byte slice, decodes it from the PEM format, ensures it's type is Certificate,
// and tries to parse it as x509.Certificate. In case an error occurs, it returns the error.
func ValidateCertificate(bytes []byte) error {
	block, _ := pem.Decode(bytes)
	if block == nil || block.Type != "CERTIFICATE" {
		return errors.New("PEM block type must be CERTIFICATE")
	}

	_, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return fmt.Errorf("failed to parse certificate: %w", err)
	}

	return nil
}
