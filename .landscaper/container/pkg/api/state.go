// Copyright (c) 2021 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file
// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"path/filepath"
)

// State defines the structure for the used state of the landscaper component.
type State struct {
	// GardenloginTlsPemPath holds the path of the gardenlogin-controller-manager tls pem file under the state directory
	GardenloginTlsPemPath string
	// GardenloginTlsKeyPemPath holds the path of the gardenlogin-controller-manager tls pem key file under the state directory
	GardenloginTlsKeyPemPath string
}

// NewStateFromPath extracts the relevant state files from the state path that were written in a previous run of the gardenlogin-controller-manager landscaper component.
func NewStateFromPath(statePath string) State {
	return State{
		GardenloginTlsPemPath:    filepath.Join(statePath, "tls.pem"),
		GardenloginTlsKeyPemPath: filepath.Join(statePath, "tls-key.pem"),
	}
}
