// Copyright (c) 2021 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file
// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package gardenlogin

import (
	"context"
	"fmt"
	"os"

	"github.com/gardener/gardenlogin-controller-manager/.landscaper/container/pkg/api"

	secretsutil "github.com/gardener/gardener/pkg/utils/secrets"
)

// Reconcile runs the reconcile operation.
func (o *operation) Reconcile(ctx context.Context) (*api.Exports, error) {
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

	// TODO certResult
	_, err = loadOrGenerateCertificate(o.state.GardenloginTlsKeyPemPath, o.state.GardenloginTlsKeyPemPath, certConfig, o.clock)
	if err != nil {
		return nil, fmt.Errorf("failed to load or generate certificate for webhook service: %w", err)
	}

	//o.contents.GardenloginTlsKeyPemFile

	return &o.exports, nil
}
