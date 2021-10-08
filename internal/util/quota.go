/*
SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors

SPDX-License-Identifier: Apache-2.0
*/

package util

import corev1 "k8s.io/api/core/v1"

// LessThan returns true if a < b for each key in b
func LessThan(a corev1.ResourceList, b corev1.ResourceList) bool {
	result := true

	for key, value := range b {
		if other, found := a[key]; found {
			if other.Cmp(value) >= 0 {
				result = false
			}
		}
	}

	return result
}
