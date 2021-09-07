// Copyright (c) 2021 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file
// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package validation

import (
	"strings"

	"github.com/gardener/gardenlogin-controller-manager/.landscaper/container/pkg/api"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	"k8s.io/apimachinery/pkg/util/validation/field"
)

// ValidateImports validates an Imports object.
func ValidateImports(obj *api.Imports) field.ErrorList {
	allErrs := field.ErrorList{}

	if obj.MultiClusterDeploymentScenario {
		allErrs = append(allErrs, validateTarget(obj.RuntimeClusterTarget, field.NewPath("runtimeClusterTarget"))...)
		allErrs = append(allErrs, validateTarget(obj.ApplicationClusterTarget, field.NewPath("applicationClusterTarget"))...)

		allErrs = append(allErrs, validateTargetNotSet(obj.SingleClusterTarget, field.NewPath("singleClusterTarget"))...)
	} else {
		allErrs = append(allErrs, validateTarget(obj.SingleClusterTarget, field.NewPath("singleClusterTarget"))...)

		allErrs = append(allErrs, validateTargetNotSet(obj.RuntimeClusterTarget, field.NewPath("runtimeClusterTarget"))...)
		allErrs = append(allErrs, validateTargetNotSet(obj.ApplicationClusterTarget, field.NewPath("applicationClusterTarget"))...)
	}

	fldValidations := fieldValidations(obj)
	allErrs = append(allErrs, validateRequiredFields(fldValidations)...)

	if strings.HasPrefix(obj.Namespace, "garden-") {
		allErrs = append(allErrs, field.Forbidden(field.NewPath("namespace"), "must not be prefixed with garden-"))
	}

	return allErrs
}

// validateTarget validates the that a target has a kubeconfig set.
func validateTarget(obj lsv1alpha1.Target, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}

	if len(obj.Spec.Configuration.RawMessage) == 0 {
		allErrs = append(allErrs, field.Required(fldPath, "kubeconfig is required"))
	}

	return allErrs
}

// validateTargetNotSet validates that the target is not initialized.
func validateTargetNotSet(obj lsv1alpha1.Target, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}

	if len(obj.Spec.Configuration.RawMessage) != 0 {
		allErrs = append(allErrs, field.Forbidden(fldPath, "target (kubeconfig) must not be set"))
	}

	return allErrs
}

func fieldValidations(obj *api.Imports) *[]fldValidation {
	fldValidations := &[]fldValidation{
		{
			value:   &obj.NamePrefix,
			fldPath: field.NewPath("namePrefix"),
		},
		{
			value:   &obj.Namespace,
			fldPath: field.NewPath("namespace"),
		},
	}

	return fldValidations
}

type fldValidation struct {
	value   *string
	fldPath *field.Path
}

func validateRequiredFields(fldValidations *[]fldValidation) field.ErrorList {
	allErrs := field.ErrorList{}

	for _, fldValidation := range *fldValidations {
		if err := validateRequiredField(fldValidation.value, fldValidation.fldPath); err != nil {
			allErrs = append(allErrs, err)
		}
	}

	return allErrs
}

func validateRequiredField(val *string, fldPath *field.Path) *field.Error {
	if val == nil || len(*val) == 0 {
		return field.Required(fldPath, "field is required")
	}

	return nil
}
