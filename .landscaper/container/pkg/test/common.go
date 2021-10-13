/*
SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors

SPDX-License-Identifier: Apache-2.0
*/

package test

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"time"

	lsv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	"k8s.io/client-go/rest"
	clientcmdv1 "k8s.io/client-go/tools/clientcmd/api/v1"
	"sigs.k8s.io/yaml"
)

var seededRand = rand.New(
	rand.NewSource(time.Now().UnixNano()))

// StringWithCharset creates a random string of provided length with the given charset. If charset is nil, a-z0-9 will be used as charset
func StringWithCharset(length int, charset *string) string {
	c := "abcdefghijklmnopqrstuvwxyz0123456789"

	if charset != nil {
		c = *charset
	}

	b := make([]byte, length)
	for i := range b {
		b[i] = c[seededRand.Intn(len(c))]
	}

	return string(b)
}

// KubeconfigFromRestConfig returns a kubeconfig yaml for the provided rest config
func KubeconfigFromRestConfig(restConfig *rest.Config) ([]byte, error) {
	cfg := &clientcmdv1.Config{
		APIVersion: "v1",
		Kind:       "Config",
		Preferences: clientcmdv1.Preferences{
			Colors: false,
		},
		Clusters: []clientcmdv1.NamedCluster{
			{
				Name: "cluster",
				Cluster: clientcmdv1.Cluster{
					Server:                   restConfig.Host,
					InsecureSkipTLSVerify:    restConfig.Insecure,
					CertificateAuthorityData: restConfig.TLSClientConfig.CAData,
				},
			},
		},
		AuthInfos: []clientcmdv1.NamedAuthInfo{
			{
				Name: "auth",
				AuthInfo: clientcmdv1.AuthInfo{
					ClientKeyData:         restConfig.TLSClientConfig.KeyData,
					ClientCertificateData: restConfig.TLSClientConfig.CertData,
				},
			},
		},
		Contexts: []clientcmdv1.NamedContext{
			{
				Name: "ctx",
				Context: clientcmdv1.Context{
					Cluster:  "cluster",
					AuthInfo: "auth",
				},
			},
		},
		CurrentContext: "ctx",
	}

	return yaml.Marshal(cfg)
}

func NewKubernetesClusterTarget(kubeconfig *string, secretRef *lsv1alpha1.SecretReference) (*lsv1alpha1.Target, error) {
	configBytes, err := json.Marshal(lsv1alpha1.KubernetesClusterTargetConfig{
		Kubeconfig: lsv1alpha1.ValueRef{
			StrVal:    kubeconfig,
			SecretRef: secretRef,
		},
	})
	if err != nil {
		return nil, fmt.Errorf("unable to decode target config: %w", err)
	}

	return &lsv1alpha1.Target{
		Spec: lsv1alpha1.TargetSpec{
			Type:          lsv1alpha1.KubernetesClusterTargetType,
			Configuration: lsv1alpha1.NewAnyJSON(configBytes),
		},
	}, nil
}
