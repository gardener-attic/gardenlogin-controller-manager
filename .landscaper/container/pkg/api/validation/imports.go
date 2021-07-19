// Copyright (c) 2021 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file
// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package validation

import (
	"github.com/gardener/gardenlogin-controller-manager/.landscaper/container/pkg/api"
	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"

	"k8s.io/apimachinery/pkg/util/validation/field"
)

// ValidateImports validates an Imports object.
func ValidateImports(obj *api.Imports) field.ErrorList {
	allErrs := field.ErrorList{}

	allErrs = append(allErrs, ValidateCluster(&obj.RuntimeClusterTarget, field.NewPath("runtimeClusterTarget"))...)
	allErrs = append(allErrs, ValidateCluster(&obj.ApplicationClusterTarget, field.NewPath("applicationClusterTarget"))...)

	return allErrs
}

// ValidateCluster validates the cluster.
func ValidateCluster(obj *lsv1alpha1.Target, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}

	if obj == nil {
		allErrs = append(allErrs, field.Required(fldPath, "target is required"))
	} else if len(obj.Spec.Configuration.RawMessage) == 0 {
		allErrs = append(allErrs, field.Required(fldPath, "kubeconfig is required"))
	}

	return allErrs
}
