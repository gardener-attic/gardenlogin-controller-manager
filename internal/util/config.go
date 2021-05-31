/*
SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors

SPDX-License-Identifier: Apache-2.0
*/

package util

import (
	"os"

	"gopkg.in/yaml.v2"
	"k8s.io/apimachinery/pkg/util/validation/field"
)

// ControllerManagerConfiguration defines the configuration for the Gardener controller manager.
type ControllerManagerConfiguration struct {
	// +optional
	Kind string `yaml:"kind"`
	// +optional
	APIVersion string `yaml:"apiVersion"`

	// Controllers defines the configuration of the controllers.
	Controllers ControllerManagerControllerConfiguration `yaml:"controllers"`
	// Webhooks defines the configuration of the admission webhooks.
	Webhooks ControllerManagerWebhookConfiguration `yaml:"webhooks"`
}

// ControllerManagerControllerConfiguration defines the configuration of the controllers.
type ControllerManagerControllerConfiguration struct {
	// ShootState defines the configuration of the ShootState controller.
	ShootState ShootStateControllerConfiguration `yaml:"shootState"`
}

// ShootStateControllerConfiguration defines the configuration of the ShootState controller.
type ShootStateControllerConfiguration struct {
	// MaxConcurrentReconciles is the maximum number of concurrent Reconciles which can be run. Defaults to 15.
	MaxConcurrentReconciles int `yaml:"maxConcurrentReconciles"`

	// MaxConcurrentReconciles is the maximum number of concurrent Reconciles which can be run per Namespace (independent of the user who created the ShootState resource). Defaults to 3.
	MaxConcurrentReconcilesPerNamespace int `yaml:"maxConcurrentReconcilesPerNamespace"`
}

// ControllerManagerWebhookConfiguration defines the configuration of the admission webhooks.
type ControllerManagerWebhookConfiguration struct {
	// ConfigMapValidation defines the configuration of the validating webhook.
	ConfigMapValidation ConfigMapValidatingWebhookConfiguration `yaml:"configmapValidation"`
}

// ConfigMapValidatingWebhookConfiguration defines the configuration of the validating webhook.
type ConfigMapValidatingWebhookConfiguration struct {
	// MaxObjectSize is the maximum size of a configmap resource in bytes. Defaults to 10240.
	MaxObjectSize int `yaml:"maxObjectSize"`
}

func ReadControllerManagerConfiguration(configFile string) (*ControllerManagerConfiguration, error) {
	// Default configuration
	cfg := ControllerManagerConfiguration{
		Controllers: ControllerManagerControllerConfiguration{
			ShootState: ShootStateControllerConfiguration{
				MaxConcurrentReconciles:             50,
				MaxConcurrentReconcilesPerNamespace: 3,
			},
		},
	}

	if err := readFile(configFile, &cfg); err != nil {
		return nil, err
	}

	if err := validateConfig(&cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}

func readFile(configFile string, cfg *ControllerManagerConfiguration) error {
	f, err := os.Open(configFile)
	if err != nil {
		return err
	}
	defer f.Close()

	decoder := yaml.NewDecoder(f)

	return decoder.Decode(cfg)
}

func validateConfig(cfg *ControllerManagerConfiguration) error {
	if cfg.Controllers.ShootState.MaxConcurrentReconciles < 1 {
		fldPath := field.NewPath("controllers", "shootState", "maxConcurrentReconciles")
		return field.Invalid(fldPath, cfg.Controllers.ShootState.MaxConcurrentReconciles, "must be 1 or greater")
	}

	if cfg.Controllers.ShootState.MaxConcurrentReconcilesPerNamespace > cfg.Controllers.ShootState.MaxConcurrentReconciles {
		fldPath := field.NewPath("controllers", "shootState", "maxConcurrentReconcilesPerNamespace")
		return field.Invalid(fldPath, cfg.Controllers.ShootState.MaxConcurrentReconcilesPerNamespace, "must not be greater than maxConcurrentReconciles")
	}

	return nil
}
