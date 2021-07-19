// Copyright (c) 2021 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file
// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package gardenlogin

import (
	"context"

	"github.com/gardener/gardenlogin-controller-manager/.landscaper/container/pkg/api"
)

// Reconcile runs the reconcile operation.
func (o *operation) Reconcile(ctx context.Context) (*api.Exports, error) {
	return &o.exports, nil
}
