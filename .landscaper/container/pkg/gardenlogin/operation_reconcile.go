// Copyright (c) 2021 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file
// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package gardenlogin

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"github.com/gardener/gardener/pkg/utils"
	"io"
	"io/ioutil"
	"k8s.io/apimachinery/pkg/api/equality"
	kErros "k8s.io/apimachinery/pkg/api/errors"
	"os/exec"
	"regexp"
	"time"

	secretsutil "github.com/gardener/gardener/pkg/utils/secrets"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/tools/cache"
	clientcmdv1 "k8s.io/client-go/tools/clientcmd/api/v1"
	watchtools "k8s.io/client-go/tools/watch"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/yaml"
)

// Reconcile runs the reconcile operation.
func (o *operation) Reconcile(ctx context.Context) error {
	if err := o.setTlsCertificate(ctx); err != nil {
		return err
	}

	if err := o.setImages(); err != nil {
		return err
	}

	if err := setNamespace([]string{
		o.contents.VirtualGardenOverlayPath,
		o.contents.RuntimeOverlayPath,
		o.contents.SingleClusterPath,
	}, o.imports.Namespace); err != nil {
		return err
	}

	if err := setNamePrefix([]string{
		o.contents.VirtualGardenOverlayPath,
		o.contents.RuntimeOverlayPath,
		o.contents.SingleClusterPath,
	}, o.imports.NamePrefix); err != nil {
		return err
	}

	if !o.imports.MultiClusterDeploymentScenario {
		// single cluster deployment
		if err := buildAndApplyOverlay(o.contents.SingleClusterPath, o.singleCluster.kubeconfig); err != nil {
			return fmt.Errorf("failed to apply or delete overlay for single cluster deployment: %w", err)
		}
	} else {
		if err := buildAndApplyOverlay(o.contents.VirtualGardenOverlayPath, o.multiCluster.applicationCluster.kubeconfig); err != nil {
			return fmt.Errorf("failed to apply or delete overlay for application cluster: %w", err)
		}

		if err := o.setGardenloginKubeconfig(ctx); err != nil {
			return err
		}

		if err := buildAndApplyOverlay(o.contents.RuntimeOverlayPath, o.multiCluster.runtimeCluster.kubeconfig); err != nil {
			return fmt.Errorf("failed to apply or delete overlay for runtime cluster: %w", err)
		}
	}

	return nil
}

// buildAndApplyOverlay builds the given overlay using kustomize and applies the result using kubectl depending on the given deleteOverlay parameter
func buildAndApplyOverlay(overlayPath string, kubeconfigPath string) error {
	kustomizeCmd := exec.Command("kustomize", "build", overlayPath)
	kustomizeStdOut, err := kustomizeCmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to get stdout pipe of kustomize command: %w", err)
	}
	kustomizeStdErr, err := kustomizeCmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("failed to get stdout pipe of kustomize command: %w", err)
	}

	kubectlCmd := exec.Command("kubectl", "--kubeconfig", kubeconfigPath, "apply", "-f", "-")

	capturedKustomizeStdOut := &bytes.Buffer{}

	// pipe stdout of kustomize to stdin of kubectl and also capture what is read from the pipe
	kubectlCmd.Stdin = io.TeeReader(kustomizeStdOut, capturedKustomizeStdOut)

	outRc, err := kubectlCmd.StdoutPipe()
	if err != nil {
		return err
	}
	errRc, err := kubectlCmd.StderrPipe()
	if err != nil {
		return err
	}

	if err := kubectlCmd.Start(); err != nil {
		return fmt.Errorf("failed to start applying kustomize build result using kubectl: %w", err)
	}

	if err := kustomizeCmd.Start(); err != nil {
		return fmt.Errorf("failed to start kustomzie build for overlay %s: %w", overlayPath, err)
	}

	stdErr, err := ioutil.ReadAll(kustomizeStdErr)
	if err != nil {
		return fmt.Errorf("failed to read stderr of kustomize command: %w", err)
	}

	if err := kustomizeCmd.Wait(); err != nil {
		stdOut := capturedKustomizeStdOut.String()
		return fmt.Errorf("failed to wait for kustomzie build to exit for overlay %s\nStdout: %s: StdErr: %s:  %w", overlayPath, stdOut, stdErr, err)
	}

	stdOut, err := ioutil.ReadAll(outRc)
	if err != nil {
		return fmt.Errorf("failed to read stdout of kubectl command")
	}
	stdErr, err = ioutil.ReadAll(errRc)
	if err != nil {
		return fmt.Errorf("failed to read stderr of kubectl command")
	}
	if err := kubectlCmd.Wait(); err != nil {
		return fmt.Errorf("failed to  wait for the kubectl command to exit for overlay %s:\nStdout: %s: StdErr: %s:  %w", overlayPath, stdOut, stdErr, err)
	}

	return nil
}

// loadOrGenerateTlsCertificate loads or generates the gardenlogin tls certificate.
// It tries to restore the tls certificate from a secret.
// It generates a new ca and tls certificate in case none was restored, it is invalid or not within the validity threshold
func (o *operation) loadOrGenerateTlsCertificate(ctx context.Context) (*secretsutil.Certificate, error) {
	var rtClient client.Client
	if o.imports.MultiClusterDeploymentScenario {
		rtClient = o.multiCluster.runtimeCluster.client
	} else {
		rtClient = o.singleCluster.client
	}

	secret := &corev1.Secret{}
	if err := rtClient.Get(ctx, client.ObjectKey{Namespace: o.imports.Namespace, Name: o.imports.NamePrefix + TlsSecretSuffix}, secret); err != nil {
		if !kErros.IsNotFound(err) {
			return nil, err
		}
		secret = nil // not found
	}

	if secret != nil {
		certificatePEM := secret.Data[corev1.TLSCertKey]

		certificate, err := utils.DecodeCertificate(certificatePEM)
		if err != nil {
			o.log.Infof("failed to parse tls certificate: %w", err)
		} else {
			needsGeneration := certificateNeedsRenewal(certificate, o.clock.Now(), 0.8)
			if !needsGeneration {
				privateKey := secret.Data[corev1.TLSPrivateKeyKey]

				return secretsutil.LoadCertificate("", privateKey, certificatePEM)
			}
		}
	}

	o.log.Info("generating new ca and tls certificate")

	caCertConfig := &secretsutil.CertificateSecretConfig{
		CertType:   secretsutil.CACert,
		CommonName: fmt.Sprintf("%s:ca", o.imports.NamePrefix),
	}

	caCert, err := caCertConfig.GenerateCertificate()
	if err != nil {
		return nil, fmt.Errorf("failed to generate ca certificate: %w", err)
	}

	certConfig := &secretsutil.CertificateSecretConfig{
		CertType:   secretsutil.ServerClientCert,
		SigningCA:  caCert,
		CommonName: fmt.Sprintf("%s-webhook-service.%s.svc.cluster.local", o.imports.NamePrefix, o.imports.Namespace),
		DNSNames: []string{
			fmt.Sprintf("%s-webhook-service", o.imports.NamePrefix),
			fmt.Sprintf("%s-webhook-service.%s", o.imports.NamePrefix, o.imports.Namespace),
			fmt.Sprintf("%s-webhook-service.%s.svc", o.imports.NamePrefix, o.imports.Namespace),
			fmt.Sprintf("%s-webhook-service.%s.svc.cluster", o.imports.NamePrefix, o.imports.Namespace),
			fmt.Sprintf("%s-webhook-service.%s.svc.cluster.local", o.imports.NamePrefix, o.imports.Namespace),
		},
	}

	cert, err := certConfig.GenerateCertificate()
	if err != nil {
		return nil, fmt.Errorf("failed to generate certificate for webhook service: %w", err)
	}

	return o.createOrUpdateTlsSecret(ctx, rtClient, cert)
}

// createOrUpdateTlsSecret creates or updates the tls secret for the webhook-server.
func (o *operation) createOrUpdateTlsSecret(ctx context.Context, c client.Client, cert *secretsutil.Certificate) (*secretsutil.Certificate, error) {
	o.log.Info("creating or updating tls certificate secret")

	secret := &corev1.Secret{}
	objKey := client.ObjectKey{Namespace: o.imports.Namespace, Name: o.imports.NamePrefix + TlsSecretSuffix}
	if err := c.Get(ctx, objKey, secret); err != nil {
		if !kErros.IsNotFound(err) {
			return nil, err
		}

		secret.Namespace = objKey.Namespace
		secret.Name = objKey.Name
		secret.Type = corev1.SecretTypeTLS
		secret.Data = map[string][]byte{
			corev1.TLSCertKey:       cert.CertificatePEM,
			corev1.TLSPrivateKeyKey: cert.PrivateKeyPEM,
		}

		if err := c.Create(ctx, secret); err != nil {
			return nil, fmt.Errorf("failed to store tls cert secret: %w", err)
		}
		return cert, nil
	}

	existing := secret.DeepCopyObject()
	secret.Type = corev1.SecretTypeTLS
	secret.Data[corev1.TLSCertKey] = cert.CertificatePEM
	secret.Data[corev1.TLSPrivateKeyKey] = cert.PrivateKeyPEM

	if equality.Semantic.DeepEqual(existing, secret) {
		return cert, nil
	}

	if err := c.Update(ctx, secret); err != nil {
		return nil, fmt.Errorf("failed to update tls cert secret: %w", err)
	}

	return cert, nil
}

// setTlsCertificate loads the tls certificate for the gardenlogin-controller-manager from a secret or generates a new certificate.
// The tls key and tls pem file is written to the respective directory of the kustomize config
func (o *operation) setTlsCertificate(ctx context.Context) error {
	tlsCert, err := o.loadOrGenerateTlsCertificate(ctx)
	if err != nil {
		return fmt.Errorf("could not load or generate gardenlogin tls certificate: %w", err)
	}

	err = ioutil.WriteFile(o.contents.GardenloginTlsKeyPemFile, tlsCert.PrivateKeyPEM, 0600)
	if err != nil {
		return fmt.Errorf("failed to write tls key pem file to path %s: %w", o.contents.GardenloginTlsKeyPemFile, err)
	}

	err = ioutil.WriteFile(o.contents.GardenloginTlsPemFile, tlsCert.CertificatePEM, 0600)
	if err != nil {
		return fmt.Errorf("failed to write tls pem file to path %s: %w", o.contents.GardenloginTlsPemFile, err)
	}

	return nil
}

// setImages uses kustomize cli to set the image for the controller (gardenlogin) and kube-rbac-proxy
func (o *operation) setImages() error {
	cmd := exec.Command("kustomize", "edit", "set", "image", fmt.Sprintf("controller=%s", o.imageRefs.GardenloginImage))
	cmd.Dir = o.contents.ManagerPath
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to set controller image %s, Output %s: %w", o.imageRefs.GardenloginImage, out, err)
	}

	cmd = exec.Command("kustomize", "edit", "set", "image", fmt.Sprintf("gcr.io/kubebuilder/kube-rbac-proxy=%s", o.imageRefs.KubeRbacProxyImage))
	cmd.Dir = o.contents.ManagerPath
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to set kube-rbac-proxy image %s, Output %s: %w", o.imageRefs.KubeRbacProxyImage, out, err)
	}

	return nil
}

// setNamespace uses kustomize cli to set the namespace field in the kustomization file
func setNamespace(overlayPaths []string, namespace string) error {
	for _, overlayPath := range overlayPaths {
		cmd := exec.Command("kustomize", "edit", "set", "namespace", namespace)
		cmd.Dir = overlayPath
		if out, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("failed to set namespace %s for overlay path %s, Output: %s: %w", namespace, out, overlayPath, err)
		}
	}
	return nil
}

// setNamespace uses kustomize cli to set the namePrefix field in the kustomization file
func setNamePrefix(overlayPaths []string, namePrefix string) error {
	for _, overlayPath := range overlayPaths {
		cmd := exec.Command("kustomize", "edit", "set", "nameprefix", namePrefix)
		cmd.Dir = overlayPath
		if out, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("failed to set nameprefix %s for overlay path %s, Output: %s: %w", namePrefix, overlayPath, out, err)
		}
	}

	return nil
}

// setGardenloginKubeconfig generates a kubeconfig for the gardenlogin-controller-manager and adds it to the overlay using kustomize cli. It reads the token of from the controller-manager service account
func (o *operation) setGardenloginKubeconfig(ctx context.Context) error {
	serviceAccount := &corev1.ServiceAccount{}
	serviceAccountName := fmt.Sprintf("%scontroller-manager", o.imports.NamePrefix)
	if err := o.multiCluster.applicationCluster.client.Get(ctx, client.ObjectKey{Namespace: o.imports.Namespace, Name: serviceAccountName}, serviceAccount); err != nil {
		return err
	}

	childCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	secret, err := waitUntilTokenAvailable(childCtx, o.multiCluster.applicationCluster.clientSet, serviceAccount)
	if err != nil {
		return fmt.Errorf("failed to wait until token is available: %w", err)
	}

	kubeconfig, err := generateKubeconfigFromTokenSecret(o.imports.ApplicationClusterEndpoint, secret)
	if err != nil {
		return fmt.Errorf("failed to generate kubeconfig for gardenlogin-controller-manager: %w", err)
	}

	if err := ioutil.WriteFile(o.contents.GardenloginKubeconfigPath, kubeconfig, 0600); err != nil {
		return fmt.Errorf("could not write kubeconfig for gardenlogin-controller-manager to %s: %w", o.contents.GardenloginKubeconfigPath, err)
	}

	cmd := exec.Command("kustomize", "edit", "add", "secret", "kubeconfig", fmt.Sprintf("--from-file=kubeconfig=%s", o.contents.GardenloginKubeconfigPath))
	cmd.Dir = o.contents.RuntimeManagerPath
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to add kubeconfig secret %s using kustomize, Output: %s: %w", o.contents.GardenloginKubeconfigPath, out, err)
	}

	return nil
}

// waitUntilTokenAvailable waits until the secret that is referenced in the service account exists and returns it.
func waitUntilTokenAvailable(ctx context.Context, cs *clientSet, serviceAccount *corev1.ServiceAccount) (*corev1.Secret, error) {
	fieldSelector := fields.SelectorFromSet(map[string]string{
		"metadata.name": serviceAccount.Name,
	}).String()

	lw := &cache.ListWatch{
		ListFunc: func(options metav1.ListOptions) (runtime.Object, error) {
			options.FieldSelector = fieldSelector
			return cs.kubernetes.CoreV1().ServiceAccounts(serviceAccount.Namespace).List(ctx, options)
		},
		WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
			options.FieldSelector = fieldSelector
			return cs.kubernetes.CoreV1().ServiceAccounts(serviceAccount.Namespace).Watch(ctx, options)
		},
	}

	event, err := watchtools.UntilWithSync(ctx, lw, &corev1.ServiceAccount{}, nil,
		func(event watch.Event) (bool, error) {
			switch event.Type {
			case watch.Deleted:
				return false, nil
			case watch.Error:
				return false, fmt.Errorf("error watching")

			case watch.Added, watch.Modified:
				watchedSa, ok := event.Object.(*corev1.ServiceAccount)
				if !ok {
					return false, fmt.Errorf("unexpected object type: %T", event.Object)
				}
				if len(watchedSa.Secrets) == 0 {
					return false, nil
				}
				return true, nil

			default:
				return false, fmt.Errorf("unexpected event type: %v", event.Type)
			}
		})

	if err != nil {
		return nil, fmt.Errorf("unable to read secret from service account: %v", err)
	}

	watchedSa, _ := event.Object.(*corev1.ServiceAccount)
	secretRef := watchedSa.Secrets[0]

	secret := &corev1.Secret{}

	return secret, cs.client.Get(ctx, client.ObjectKey{Namespace: serviceAccount.Namespace, Name: secretRef.Name}, secret)
}

// generateKubeconfigFromTokenSecret generates a kubeconfig using the bearer token from the provided secret to authenticate against the provided server.
// If the server points to localhost, the kubernetes default service is used instead as server.
func generateKubeconfigFromTokenSecret(server string, secret *corev1.Secret) ([]byte, error) {
	if server == "" {
		return nil, errors.New("api server host is required")
	}

	matched, _ := regexp.MatchString(`^https:\/\/localhost:\d{1,5}$`, server)
	if matched {
		server = "https://kubernetes.default.svc.cluster.local"
	}

	token, ok := secret.Data[corev1.ServiceAccountTokenKey]
	if !ok {
		return nil, fmt.Errorf("no %s data key found on secret", corev1.ServiceAccountTokenKey)
	}

	name := "gardenlogin"
	kubeconfig := &clientcmdv1.Config{
		APIVersion: "v1",
		Kind:       "Config",
		Preferences: clientcmdv1.Preferences{
			Colors: false,
		},
		Clusters: []clientcmdv1.NamedCluster{
			{
				Name: name,
				Cluster: clientcmdv1.Cluster{
					Server:                   server,
					InsecureSkipTLSVerify:    false,
					CertificateAuthorityData: secret.Data[corev1.ServiceAccountRootCAKey],
				},
			},
		},
		AuthInfos: []clientcmdv1.NamedAuthInfo{
			{
				Name: name,
				AuthInfo: clientcmdv1.AuthInfo{
					Token: string(token),
				},
			},
		},
		Contexts: []clientcmdv1.NamedContext{
			{
				Name: name,
				Context: clientcmdv1.Context{
					Cluster:  name,
					AuthInfo: name,
				},
			},
		},
		CurrentContext: name,
	}

	return yaml.Marshal(kubeconfig)
}
