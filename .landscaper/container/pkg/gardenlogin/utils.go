// Copyright (c) 2021 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file
// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package gardenlogin

import (
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"os"
	"time"

	secretsutil "github.com/gardener/gardener/pkg/utils/secrets"
)

type CertificateResult struct {
	// certificate holds the certificate
	certificate *secretsutil.Certificate
	// loaded is true in case the certificate could be loaded from the state
	loaded bool
}

// loadOrGenerateCertificate tries to load the private key pem and tls pem from the given paths (usually from the state directory).
// If
// - either one is not found
// - or the certificate is expired
// - or 80% of the validity threshold is exceeded,
// a new certificate is generated using the given certificateConfig.
// The generated private key pem and tls pem is written to the given path (which is usually within the state directory).
func loadOrGenerateCertificate(tlsKeyPemPath string, tlsPemPath string, certificateConfig *secretsutil.CertificateSecretConfig, clock Clock) (*CertificateResult, error) {
	tlsKeyPemExists, err := checkFileExists(tlsKeyPemPath)
	if err != nil {
		return nil, fmt.Errorf("failed to check if tls key pem file exists: %w", err)
	}
	tlsPemExists, err := checkFileExists(tlsPemPath)
	if err != nil {
		return nil, fmt.Errorf("failed to check if tls pem file exists: %w", err)
	}

	needsGeneration := true
	if tlsKeyPemExists && tlsPemExists {
		// check if certificate is expired

		tlsPem, err := ioutil.ReadFile(tlsPemPath)
		if err != nil {
			return nil, fmt.Errorf("could not read tls pem to verify certificate validity: %w", err)
		}

		certificate, err := x509.ParseCertificate(tlsPem)
		if err != nil {
			return nil, fmt.Errorf("failed to parse certificate: %w", err)
		}

		needsGeneration = certificateNeedsRenewal(certificate, clock.Now(), 0.8)
	}

	if needsGeneration {
		certificate, err := certificateConfig.GenerateCertificate()
		if err != nil {
			return nil, err
		}

		err = ioutil.WriteFile(tlsKeyPemPath, certificate.PrivateKeyPEM, 0600)
		if err != nil {
			return nil, fmt.Errorf("failed write private key pem to path %s: %w", tlsKeyPemPath, err)
		}

		err = ioutil.WriteFile(tlsPemPath, certificate.CertificatePEM, 0600)
		if err != nil {
			return nil, fmt.Errorf("failed write certificate pem to path %s: %w", tlsPemPath, err)
		}

		return &CertificateResult{
			certificate: certificate,
			loaded:      true,
		}, nil
	}

	tlsKeyPem, err := ioutil.ReadFile(tlsKeyPemPath)
	if err != nil {
		return nil, err
	}

	tlsPem, err := ioutil.ReadFile(tlsPemPath)
	if err != nil {
		return nil, err
	}

	certificate, err := secretsutil.LoadCertificate(certificateConfig.Name, tlsKeyPem, tlsPem)
	if err != nil {
		return nil, err
	}
	certificate.CA = certificateConfig.SigningCA

	return &CertificateResult{
		certificate: certificate,
		loaded:      false,
	}, nil
}

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

func checkFileExists(path string) (bool, error) {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return false, nil
	} else {
		return false, err
	}

	return true, nil
}
