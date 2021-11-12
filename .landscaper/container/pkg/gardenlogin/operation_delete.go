// Copyright (c) 2021 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file
// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package gardenlogin

import (
	"context"
	"fmt"

	admissionregistrationv1 "k8s.io/api/admissionregistration/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Delete runs the delete operation. It clears all resources that were previously generated by the deploy container.
func (o *operation) Delete(ctx context.Context) error {
	err := o.deleteRuntimeResources(ctx)
	if err != nil {
		return fmt.Errorf("failed to delete runtime resources %w", err)
	}

	err = o.deleteApplicationResources(ctx)
	if err != nil {
		return fmt.Errorf("failed to delete application resources %w", err)
	}

	return nil
}

// deleteRuntimeResources deletes the runtime cluster specific resources of the gardenlogin-controller-manager if not already deleted
func (o *operation) deleteRuntimeResources(ctx context.Context) error {
	rtClient := o.runtimeCluster().client

	nsKey := client.ObjectKey{Name: o.imports.Namespace}
	ns := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: nsKey.Name}}

	if err := ensureDeleted(ctx, rtClient, nsKey, ns); err != nil {
		return err
	}

	crbKey := client.ObjectKey{Name: fmt.Sprintf("%sproxy-rolebinding", o.imports.NamePrefix)}
	crb := &rbacv1.ClusterRoleBinding{ObjectMeta: metav1.ObjectMeta{Name: crbKey.Name}}

	if err := ensureDeleted(ctx, rtClient, crbKey, crb); err != nil {
		return err
	}

	crKey := client.ObjectKey{Name: fmt.Sprintf("%sproxy-role", o.imports.NamePrefix)}
	cr := &rbacv1.ClusterRole{ObjectMeta: metav1.ObjectMeta{Name: crKey.Name}}

	if err := ensureDeleted(ctx, rtClient, crKey, cr); err != nil {
		return err
	}

	return nil
}

// deleteApplicationResources deletes the application cluster specific resources of the gardenlogin-controller-manager if not already deleted
func (o *operation) deleteApplicationResources(ctx context.Context) error {
	appClient := o.applicationCluster().client

	nsKey := client.ObjectKey{Name: o.imports.Namespace}
	ns := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: nsKey.Name}}

	if err := ensureDeleted(ctx, appClient, nsKey, ns); err != nil {
		return err
	}

	crbKey := client.ObjectKey{Name: fmt.Sprintf("%smanager-rolebinding", o.imports.NamePrefix)}
	crb := &rbacv1.ClusterRoleBinding{ObjectMeta: metav1.ObjectMeta{Name: crbKey.Name}}

	if err := ensureDeleted(ctx, appClient, crbKey, crb); err != nil {
		return err
	}

	crKey := client.ObjectKey{Name: fmt.Sprintf("%smanager-role", o.imports.NamePrefix)}
	cr := &rbacv1.ClusterRole{ObjectMeta: metav1.ObjectMeta{Name: crKey.Name}}

	if err := ensureDeleted(ctx, appClient, crKey, cr); err != nil {
		return err
	}

	vwcKey := client.ObjectKey{Name: fmt.Sprintf("%svalidating-webhook-configuration", o.imports.NamePrefix)}
	vwc := &admissionregistrationv1.ValidatingWebhookConfiguration{ObjectMeta: metav1.ObjectMeta{Name: vwcKey.Name}}

	if err := ensureDeleted(ctx, appClient, vwcKey, vwc); err != nil {
		return err
	}

	return nil
}

func ensureDeleted(ctx context.Context, c client.Client, objectKey client.ObjectKey, obj client.Object) error {
	if err := c.Get(ctx, objectKey, obj); err != nil {
		if apierrors.IsNotFound(err) {
			return nil // already deleted
		}

		return err
	}

	if !obj.GetDeletionTimestamp().IsZero() {
		return nil // The object is being deleted
	}

	return client.IgnoreNotFound(c.Delete(ctx, obj))
}