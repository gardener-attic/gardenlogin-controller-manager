// Copyright (c) 2021 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file
// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package gardenlogin

import (
	"bytes"
	"context"
	_ "embed" // The //go:embed directive requires importing "embed", hence using a blank import
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"os/exec"
	"regexp"
	"text/template"
	"time"

	"github.com/gardener/gardenlogin-controller-manager/.landscaper/container/internal/util"

	"github.com/gardener/gardener/pkg/client/kubernetes"
	"github.com/gardener/gardener/pkg/utils"
	secretsutil "github.com/gardener/gardener/pkg/utils/secrets"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
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

var (
	//go:embed templates/deployment_resources_patch.tpl.yaml
	tplResourcesPatch string
	tplResources      *template.Template
)

func init() {
	tplResources = template.Must(
		template.
			New("resources").
			Funcs(map[string]interface{}{
				"mustToJson": mustToJSON,
			}).
			Parse(tplResourcesPatch))
}

func mustToJSON(v interface{}) (string, error) {
	output, err := json.Marshal(v)
	if err != nil {
		return "", err
	}

	return string(output), nil
}

// Reconcile runs the reconcile operation.
func (o *operation) Reconcile(ctx context.Context) error {
	cert, err := o.setTLSCertificate(ctx)
	if err != nil {
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

	if err := o.patchResourceRequirements([]string{
		o.contents.RuntimeOverlayPath,
		o.contents.SingleClusterPath,
	}); err != nil {
		return err
	}

	if err := o.setManagerConfig([]string{
		o.contents.ManagerConfigurationRuntimePath,
		o.contents.ManagerConfigurationSingleClusterPath,
	}); err != nil {
		return err
	}

	if !o.imports.MultiClusterDeploymentScenario {
		// single cluster deployment
		if err := o.singleCluster.buildAndApplyOverlay(ctx, o.contents.SingleClusterPath); err != nil {
			return fmt.Errorf("failed to apply overlay for single cluster deployment: %w", err)
		}
	} else {
		if err := o.multiCluster.applicationCluster.buildAndApplyOverlay(ctx, o.contents.VirtualGardenOverlayPath); err != nil {
			return fmt.Errorf("failed to applyoverlay for application cluster: %w", err)
		}

		if err := o.setGardenloginKubeconfig(ctx); err != nil {
			return err
		}

		if err := o.multiCluster.runtimeCluster.buildAndApplyOverlay(ctx, o.contents.RuntimeOverlayPath); err != nil {
			return fmt.Errorf("failed to apply overlay for runtime cluster: %w", err)
		}
	}

	return o.createOrUpdateTLSSecret(ctx, cert)
}

// buildAndApplyOverlay builds the given overlay using kustomize and applies the manifests to the given cluster
func (cluster *cluster) buildAndApplyOverlay(ctx context.Context, overlayPath string) error {
	cmd := exec.Command("kustomize", "build", overlayPath)

	var errBuff bytes.Buffer
	cmd.Stderr = &errBuff

	out, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("failed to run kustomization: %s, %w", errBuff.String(), err)
	}

	applier := kubernetes.NewApplier(cluster.client, cluster.client.RESTMapper())
	mr := kubernetes.NewManifestReader(out)

	return applier.ApplyManifest(ctx, mr, kubernetes.DefaultMergeFuncs)
}

// loadOrGenerateTLSCertificate loads or generates the gardenlogin tls certificate.
// It tries to restore the tls certificate from a secret.
// It generates a new ca and tls certificate in case none was restored, it is invalid or not within the validity threshold
func (o *operation) loadOrGenerateTLSCertificate(ctx context.Context) (*secretsutil.Certificate, error) {
	rtClient := o.runtimeCluster().client

	secret := &corev1.Secret{}
	if err := rtClient.Get(ctx, client.ObjectKey{Namespace: o.imports.Namespace, Name: o.imports.NamePrefix + TLSSecretSuffix}, secret); err != nil {
		if !apierrors.IsNotFound(err) {
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
			needsGeneration := util.CertificateNeedsRenewal(certificate, o.clock.Now(), 0.8)
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
		Now:        o.clock.Now,
	}

	caCert, err := caCertConfig.GenerateCertificate()
	if err != nil {
		return nil, fmt.Errorf("failed to generate ca certificate: %w", err)
	}

	certConfig := &secretsutil.CertificateSecretConfig{
		CertType:   secretsutil.ServerClientCert,
		SigningCA:  caCert,
		CommonName: fmt.Sprintf("%swebhook-service.%s.svc.cluster.local", o.imports.NamePrefix, o.imports.Namespace),
		DNSNames: []string{
			fmt.Sprintf("%swebhook-service", o.imports.NamePrefix),
			fmt.Sprintf("%swebhook-service.%s", o.imports.NamePrefix, o.imports.Namespace),
			fmt.Sprintf("%swebhook-service.%s.svc", o.imports.NamePrefix, o.imports.Namespace),
			fmt.Sprintf("%swebhook-service.%s.svc.cluster", o.imports.NamePrefix, o.imports.Namespace),
			fmt.Sprintf("%swebhook-service.%s.svc.cluster.local", o.imports.NamePrefix, o.imports.Namespace),
		},
		Now: o.clock.Now,
	}

	cert, err := certConfig.GenerateCertificate()
	if err != nil {
		return nil, fmt.Errorf("failed to generate certificate for webhook service: %w", err)
	}

	return cert, nil
}

// createOrUpdateTLSSecret creates or updates the tls secret for the webhook-server on the runtime cluster so that it can be fetched on a next reconcile run.
func (o *operation) createOrUpdateTLSSecret(ctx context.Context, cert *secretsutil.Certificate) error {
	o.log.Info("creating or updating tls certificate secret")
	rtClient := o.runtimeCluster().client

	objKey := client.ObjectKey{Namespace: o.imports.Namespace, Name: o.imports.NamePrefix + TLSSecretSuffix}

	secret := &corev1.Secret{}
	if err := rtClient.Get(ctx, objKey, secret); err != nil {
		if !apierrors.IsNotFound(err) {
			return err
		}

		secret.Namespace = objKey.Namespace
		secret.Name = objKey.Name
		secret.Type = corev1.SecretTypeTLS
		secret.Data = map[string][]byte{
			corev1.TLSCertKey:       cert.CertificatePEM,
			corev1.TLSPrivateKeyKey: cert.PrivateKeyPEM,
		}

		if err := rtClient.Create(ctx, secret); err != nil {
			return fmt.Errorf("failed to store tls cert secret: %w", err)
		}

		return nil
	}

	existing := secret.DeepCopyObject()
	secret.Type = corev1.SecretTypeTLS
	secret.Data[corev1.TLSCertKey] = cert.CertificatePEM
	secret.Data[corev1.TLSPrivateKeyKey] = cert.PrivateKeyPEM

	if equality.Semantic.DeepEqual(existing, secret) {
		return nil
	}

	if err := rtClient.Update(ctx, secret); err != nil {
		return fmt.Errorf("failed to update tls cert secret: %w", err)
	}

	return nil
}

// setTLSCertificate loads the tls certificate for the gardenlogin-controller-manager from a secret or generates a new certificate.
// The tls key and tls pem file is written to the respective directory of the kustomize config
func (o *operation) setTLSCertificate(ctx context.Context) (*secretsutil.Certificate, error) {
	tlsCert, err := o.loadOrGenerateTLSCertificate(ctx)
	if err != nil {
		return nil, fmt.Errorf("could not load or generate gardenlogin tls certificate: %w", err)
	}

	err = ioutil.WriteFile(o.contents.GardenloginTLSKeyPemFile, tlsCert.PrivateKeyPEM, 0600)
	if err != nil {
		return nil, fmt.Errorf("failed to write tls key pem file to path %s: %w", o.contents.GardenloginTLSKeyPemFile, err)
	}

	err = ioutil.WriteFile(o.contents.GardenloginTLSPemFile, tlsCert.CertificatePEM, 0600)
	if err != nil {
		return nil, fmt.Errorf("failed to write tls pem file to path %s: %w", o.contents.GardenloginTLSPemFile, err)
	}

	return tlsCert, nil
}

// setImages uses kustomize cli to set the image for the controller (gardenlogin) and kube-rbac-proxy
func (o *operation) setImages() error {
	cmd := exec.Command("kustomize", "edit", "set", "image", fmt.Sprintf("controller=%s", o.imageRefs.GardenloginImage))
	cmd.Dir = o.contents.ManagerPath

	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to set controller image %s, Output %s: %w", o.imageRefs.GardenloginImage, out, err)
	}

	cmd = exec.Command("kustomize", "edit", "set", "image", fmt.Sprintf("gcr.io/kubebuilder/kube-rbac-proxy=%s", o.imageRefs.KubeRBACProxyImage))
	cmd.Dir = o.contents.ManagerPath

	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to set kube-rbac-proxy image %s, Output %s: %w", o.imageRefs.KubeRBACProxyImage, out, err)
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

// setManagerConfig writes the manger config from the imports to the given overlay paths
func (o *operation) setManagerConfig(overlayPaths []string) error {
	config, err := yaml.Marshal(o.imports.ManagerConfig)
	if err != nil {
		return fmt.Errorf("failed to marshal manager config: %w", err)
	}

	for _, overlayPath := range overlayPaths {
		if err = ioutil.WriteFile(overlayPath, config, 0600); err != nil {
			return fmt.Errorf("failed to write manager config to path %s: %w", overlayPath, err)
		}
	}

	return nil
}

// patchResourceRequirements uses kustomize cli to patch the resource requirements for the manager and kube-rbac-proxy container according to the import parameters
func (o *operation) patchResourceRequirements(overlayPaths []string) error {
	patch := bytes.NewBuffer(nil)
	if err := tplResources.Execute(patch, map[string]interface{}{
		"managerResources":       o.imports.ManagerResources,
		"kubeRbacProxyResources": o.imports.KubeRBACProxyResources,
	}); err != nil {
		return err
	}

	for _, overlayPath := range overlayPaths {
		cmd := exec.Command("kustomize", "edit", "add", "patch", "--patch", patch.String())
		cmd.Dir = overlayPath

		if out, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("failed to patch resource requirements for overlay path %s, Output: %s: %w", overlayPath, out, err)
		}
	}

	return nil
}

// setGardenloginKubeconfig generates a kubeconfig for the gardenlogin-controller-manager and adds it to the overlay using kustomize cli. It reads the token of from the controller-manager service account
func (o *operation) setGardenloginKubeconfig(ctx context.Context) error {
	serviceAccountName := fmt.Sprintf("%scontroller-manager", o.imports.NamePrefix)

	serviceAccount := &corev1.ServiceAccount{}
	if err := o.multiCluster.applicationCluster.client.Get(ctx, client.ObjectKey{Namespace: o.imports.Namespace, Name: serviceAccountName}, serviceAccount); err != nil {
		return err
	}

	childCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	secret, err := waitUntilTokenAvailable(childCtx, o.applicationCluster().clientSet, serviceAccount)
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
