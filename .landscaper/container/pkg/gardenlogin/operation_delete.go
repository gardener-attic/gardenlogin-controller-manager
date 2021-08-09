// Copyright (c) 2021 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file
// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package gardenlogin

import (
	"context"
	"fmt"

	admissionregistrationv1beta1 "k8s.io/api/admissionregistration/v1beta1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	kErros "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Delete runs the delete operation.
func (o *operation) Delete(ctx context.Context) error {
	var rtClient, appClient client.Client

	if !o.imports.MultiClusterDeploymentScenario {
		appClient = o.singleCluster.clientSet.client
		rtClient = o.singleCluster.clientSet.client
	} else {
		rtClient = o.multiCluster.runtimeCluster.clientSet.client
		appClient = o.multiCluster.applicationCluster.clientSet.client
	}

	deleteRuntimeResources(ctx, rtClient, o.imports.Namespace, o.imports.NamePrefix)
	deleteApplicationResources(ctx, appClient, o.imports.Namespace, o.imports.NamePrefix)
	return nil
}

// deleteRuntimeResources deletes the runtime cluster specific resources if not already deleted
func deleteRuntimeResources(ctx context.Context, client client.Client, namespace string, namePrefix string) error {
	ns := corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: namespace}}
	if err := client.Delete(ctx, &ns); err != nil && !kErros.IsNotFound(err) {
		return err
	}

	crb := rbacv1.ClusterRoleBinding{ObjectMeta: metav1.ObjectMeta{Name: fmt.Sprintf("%sproxy-rolebinding", namePrefix)}}
	if err := client.Delete(ctx, &crb); err != nil && !kErros.IsNotFound(err) {
		return err
	}

	cr := rbacv1.ClusterRole{ObjectMeta: metav1.ObjectMeta{Name: fmt.Sprintf("%sproxy-role", namePrefix)}}
	if err := client.Delete(ctx, &cr); err != nil && !kErros.IsNotFound(err) {
		return err
	}

	return nil
}

// deleteApplicationResources deletes the application cluster specific resources if not already deleted
func deleteApplicationResources(ctx context.Context, client client.Client, namespace string, namePrefix string) error {
	ns := corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: namespace}}
	if err := client.Delete(ctx, &ns); err != nil && !kErros.IsNotFound(err) {
		return err
	}

	crb := rbacv1.ClusterRoleBinding{ObjectMeta: metav1.ObjectMeta{Name: fmt.Sprintf("%smanager-rolebinding", namePrefix)}}
	if err := client.Delete(ctx, &crb); err != nil && !kErros.IsNotFound(err) {
		return err
	}

	cr := rbacv1.ClusterRole{ObjectMeta: metav1.ObjectMeta{Name: fmt.Sprintf("%smanager-role", namePrefix)}}
	if err := client.Delete(ctx, &cr); err != nil && !kErros.IsNotFound(err) {
		return err
	}

	vwc := admissionregistrationv1beta1.ValidatingWebhookConfiguration{ObjectMeta: metav1.ObjectMeta{Name: fmt.Sprintf("%svalidating-webhook-configuration", namePrefix)}}
	if err := client.Delete(ctx, &vwc); err != nil && !kErros.IsNotFound(err) {
		return err
	}

	return nil
}
