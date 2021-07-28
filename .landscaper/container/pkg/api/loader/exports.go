// Copyright (c) 2021 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file
// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package loader

import (
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/gardener/gardenlogin-controller-manager/.landscaper/container/pkg/api"

	"gopkg.in/yaml.v2"
)

// ExportsToFile writes export data to a file.
func ExportsToFile(exports *api.Exports, path string) error {
	b, err := yaml.Marshal(exports)
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

// ExportsFromFile reads export data from a file.
func ExportsFromFile(path string) (*api.Exports, error) {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}

	exports := &api.Exports{}
	if err := yaml.Unmarshal(data, exports); err != nil {
		return nil, err
	}

	return exports, nil
}
