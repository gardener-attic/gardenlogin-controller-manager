// Copyright (c) 2021 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file
// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"encoding/json"
	"fmt"

	cdv2 "github.com/gardener/component-spec/bindings-go/apis/v2"
)

// ImageRefs defines the structure for the used images.
type ImageRefs struct {
	// GardenloginImage holds the image of the gardenlogin-controller-manager
	GardenloginImage string
	// KubeRbacProxyImage holds the image of the brancz/kube-rbac-proxy image
	KubeRbacProxyImage string
}

// NewImageRefsFromComponentDescriptor extracts the relevant images from the component descriptor.
func NewImageRefsFromComponentDescriptor(cd *cdv2.ComponentDescriptor) (*ImageRefs, error) {
	const (
		resourceNameGardenlogin   = "gardenlogin-controller-manager"
		resourceNameKubeRbacProxy = "kube-rbac-proxy"
	)

	imageRefs := ImageRefs{}

	var err error

	imageRefs.GardenloginImage, err = getImageRef(resourceNameGardenlogin, cd)
	if err != nil {
		return nil, err
	}

	imageRefs.KubeRbacProxyImage, err = getImageRef(resourceNameKubeRbacProxy, cd)
	if err != nil {
		return nil, err
	}

	return &imageRefs, nil
}

func getImageRef(resourceName string, cd *cdv2.ComponentDescriptor) (string, error) {
	for i := range cd.Resources {
		resource := &cd.Resources[i]

		if resource.Name == resourceName {
			access := cdv2.OCIRegistryAccess{}
			if err := json.Unmarshal(resource.Access.Raw, &access); err != nil {
				return "", err
			}

			return access.ImageReference, nil
		}
	}

	return "", fmt.Errorf("No resource with name %s found in component descriptor", resourceName)
}
