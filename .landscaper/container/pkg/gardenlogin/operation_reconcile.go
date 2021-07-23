// Copyright (c) 2021 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file
// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package gardenlogin

import (
	"context"
	"fmt"
	secretsutil "github.com/gardener/gardener/pkg/utils/secrets"
	"github.com/gardener/gardenlogin-controller-manager/.landscaper/container/pkg/api"
	"io/ioutil"
	"os"
)

// Reconcile runs the reconcile operation.
func (o *operation) Reconcile(ctx context.Context) (*api.Exports, error) {
	tlsCert, err := o.loadOrGenerateGardenloginTlsCertificate()
	if err != nil {
		return nil, fmt.Errorf("could not load or generate gardenlogin tls certificate: %w", err)
	}

	err = ioutil.WriteFile(o.contents.GardenloginTlsKeyPemFile, tlsCert.PrivateKeyPEM, 0600)
	if err != nil {
		fmt.Errorf("failed to write tls key pem file to path %s: %w", o.contents.GardenloginTlsKeyPemFile, err)
	}

	err = ioutil.WriteFile(o.contents.GardenloginTlsPemFile, tlsCert.CertificatePEM, 0600)
	if err != nil {
		fmt.Errorf("failed to write tls pem file to path %s: %w", o.contents.GardenloginTlsPemFile, err)
	}

	return &o.exports, nil
}

// loadOrGenerateGardenloginTlsCertificate loads or generates the gardenlogin tls certificate.
// It tries to restore the ca and tls certificate from the state
// or generates new in case they are not valid or not within the validity threshold
func (o *operation) loadOrGenerateGardenloginTlsCertificate() (*secretsutil.Certificate, error) {
	caCertConfig := &secretsutil.CertificateSecretConfig{
		CertType:   secretsutil.CACert,
		CommonName: Prefix + ":ca",
	}

	caCertResult, err := loadOrGenerateCertificate(o.state.CaKeyPemPath, o.state.CaPemPath, caCertConfig, o.clock)
	if err != nil {
		return nil, fmt.Errorf("failed to load or generate ca certificate: %w", err)
	}

	if !caCertResult.loaded {
		o.log.Info("cleaning up gardenlogin tls certificate from state in order to generate a new certificate")
		err := os.Remove(o.state.GardenloginTlsKeyPemPath)
		if err != nil {
			return nil, fmt.Errorf("failed to cleanup tls key pem file: %w", err)
		}

		err = os.Remove(o.state.GardenloginTlsPemPath)
		if err != nil {
			return nil, fmt.Errorf("failed to cleanup tls pem file: %w", err)
		}
	}

	certConfig := &secretsutil.CertificateSecretConfig{
		CertType:   secretsutil.ServerClientCert,
		SigningCA:  caCertResult.certificate,
		CommonName: fmt.Sprintf("%s-webhook-service.%s.svc.cluster.local", Prefix, o.namespace),
		DNSNames: []string{
			fmt.Sprintf("%s-webhook-service", Prefix),
			fmt.Sprintf("%s-webhook-service.%s", Prefix, o.namespace),
			fmt.Sprintf("%s-webhook-service.%s.svc", Prefix, o.namespace),
			fmt.Sprintf("%s-webhook-service.%s.svc.cluster", Prefix, o.namespace),
			fmt.Sprintf("%s-webhook-service.%s.svc.cluster.local", Prefix, o.namespace),
		},
	}

	certResult, err := loadOrGenerateCertificate(o.state.GardenloginTlsKeyPemPath, o.state.GardenloginTlsKeyPemPath, certConfig, o.clock)
	if err != nil {
		return nil, fmt.Errorf("failed to load or generate certificate for webhook service: %w", err)
	}

	cert := certResult.certificate
	if cert == nil {
		return nil, fmt.Errorf("no certificate returned")
	}

	return cert, nil
}
