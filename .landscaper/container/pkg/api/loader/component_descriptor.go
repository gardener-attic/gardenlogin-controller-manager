// Copyright (c) 2021 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file
// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package loader

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	cdv2 "github.com/gardener/component-spec/bindings-go/apis/v2"
	"gopkg.in/yaml.v2"
)

// ComponentDescriptorToFile writes a component descriptor to a file.
func ComponentDescriptorToFile(componentDescriptor *cdv2.ComponentDescriptorList, path string) error {
	b, err := yaml.Marshal(componentDescriptor)
	if err != nil {
		return err
	}

	folderPath := filepath.Dir(path)
	if _, err := os.Stat(folderPath); os.IsNotExist(err) {
		if err := os.MkdirAll(folderPath, 0700); err != nil {
			return err
		}
	}

	return ioutil.WriteFile(path, b, os.ModePerm)
}

// ComponentDescriptorFromFile reads a component descriptor from a file.
func ComponentDescriptorFromFile(componentDescriptorPath string) (*cdv2.ComponentDescriptor, error) {
	data, err := ioutil.ReadFile(componentDescriptorPath)
	if err != nil {
		return nil, err
	}

	cdList := &cdv2.ComponentDescriptorList{}
	if err := yaml.Unmarshal(data, cdList); err != nil {
		return nil, err
	}

	if len(cdList.Components) != 1 {
		return nil, fmt.Errorf("Component descriptor list does not contain a unique entry")
	}

	return &cdList.Components[0], nil
}
