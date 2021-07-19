// Copyright (c) 2021 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file
// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package loader

import (
	"io/ioutil"

	"sigs.k8s.io/yaml"

	"github.com/gardener/gardenlogin-controller-manager/.landscaper/container/pkg/api"
)

// ImportsFromFile will read the file from the given path and try to unmarshal it into an api.Imports structure.
func ImportsFromFile(path string) (*api.Imports, error) {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}

	imports := &api.Imports{}
	if err := yaml.Unmarshal(data, imports); err != nil {
		return nil, err
	}

	return imports, nil
}
