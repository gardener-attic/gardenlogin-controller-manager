// Copyright (c) 2021 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file
// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package validation

import (
	"errors"
	"fmt"
	"os"

	"github.com/gardener/gardenlogin-controller-manager/.landscaper/container/pkg/api"
)

// ValidateContents validates an Contents object.
func ValidateContents(obj api.Contents) error {
	if err := validatePathExists(obj.DefaultPath); err != nil {
		return fmt.Errorf("validation failed for default path: %w", err)
	}
	if err := validatePathExists(obj.ManagerPath); err != nil {
		return fmt.Errorf("validation failed for manager path: %w", err)
	}
	if err := validatePathExists(obj.GardenloginTlsPath); err != nil {
		return fmt.Errorf("validation failed for tls path: %w", err)
	}
	if err := validatePathExists(obj.RuntimeManagerPath); err != nil {
		return fmt.Errorf("validation failed for runtime manager path: %w", err)
	}
	if err := validatePathExists(obj.VirtualGardenOverlayPath); err != nil {
		return fmt.Errorf("validation failed for virtual garden overlay path: %w", err)
	}
	if err := validatePathExists(obj.RuntimeOverlayPath); err != nil {
		return fmt.Errorf("validation failed for runtime overlay path: %w", err)
	}
	if err := validatePathExists(obj.SingleClusterPath); err != nil {
		return fmt.Errorf("validation failed for single cluster overlay path: %w", err)
	}

	return nil
}

// validatePathExists validates that the given path exists.
func validatePathExists(path string) error {
	if len(path) == 0 {
		return errors.New("path not set")
	}

	if _, err := os.Stat(path); os.IsNotExist(err) {
		return fmt.Errorf("path %s does not exist", path)
	} else {
		return err
	}

	return nil
}
