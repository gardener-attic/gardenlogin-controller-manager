// Copyright (c) 2021 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file
// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package api

import "path/filepath"

// Contents defines the structure for the used content data of the landscaper component.
type Contents struct {
	// DefaultPath holds the path of the "default" folder in which the kube-rbac-proxy image is defined
	DefaultPath string
	// ManagerPath holds the path of the "manager" folder in which the gardenlogin-controller-manager image is defined
	ManagerPath string

	// GardenloginTlsPath holds the path of the "tls" folder in which the tls certificate files are placed for kustomize's secretGenerator to pick them up
	GardenloginTlsPath string
	// GardenloginTlsPemFile holds the file path of the gardenlogin-controller-manager-tls.pem file for the webhook server
	GardenloginTlsPemFile string
	// GardenloginTlsKeyPemFile holds the file path of the gardenlogin-controller-manager-tls-key.pem file for the webhook server
	GardenloginTlsKeyPemFile string

	// RuntimeManagerPath is the path to the manager directory of the runtime overlay
	RuntimeManagerPath string
	// GardenloginKubeconfigPath holds the file path of the kubeconfig for the gardenlogin-controller-manager
	GardenloginKubeconfigPath string

	// Kustomize Overlay Paths

	// VirtualGardenOverlayPath holds the path of the virtual garden kustomize overlay
	VirtualGardenOverlayPath string
	// RuntimeOverlayPath holds the path of the runtime-cluster kustomize overlay
	RuntimeOverlayPath string
	// SingleClusterPath holds the path of the single-cluster kustomize overlay
	SingleClusterPath string
}

// TODO
func NewContentsFromPath(contentPath string) Contents {
	contents := Contents{
		DefaultPath: filepath.Join(contentPath, "config", "default"),
		ManagerPath: filepath.Join(contentPath, "config", "manager"),

		GardenloginTlsPath:       filepath.Join(contentPath, "config", "secret", "tls"),
		GardenloginTlsPemFile:    filepath.Join(contentPath, "config", "secret", "tls", "gardenlogin-controller-manager-tls.pem"),
		GardenloginTlsKeyPemFile: filepath.Join(contentPath, "config", "secret", "tls", "gardenlogin-controller-manager-tls-key.pem"),

		RuntimeManagerPath:        filepath.Join(contentPath, "config", "overlay", "multi-cluster", "runtime", "manager"),
		GardenloginKubeconfigPath: filepath.Join(contentPath, "config", "overlay", "multi-cluster", "runtime", "manager", "kubeconfig.yaml"),

		VirtualGardenOverlayPath: filepath.Join(contentPath, "config", "overlay", "multi-cluster", "virtual-garden"),
		RuntimeOverlayPath:       filepath.Join(contentPath, "config", "overlay", "multi-cluster", "runtime"),
		SingleClusterPath:        filepath.Join(contentPath, "config", "overlay", "single-cluster"),
	}

	return contents
}
