/*
SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors

SPDX-License-Identifier: Apache-2.0
*/

package test

import (
	"crypto/tls"
	"fmt"
	"net"
	"net/url"
	"os"
	"time"

	"github.com/gardener/gardenlogin-controller-manager/api/v1alpha1/constants"
	"github.com/gardener/gardenlogin-controller-manager/internal/util"

	"github.com/gardener/gardener/pkg/client/kubernetes"
	gardenenvtest "github.com/gardener/gardener/pkg/envtest"
	"github.com/onsi/ginkgo"
	"github.com/onsi/gomega"
	admissionregistrationv1 "k8s.io/api/admissionregistration/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"
	"k8s.io/utils/pointer"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

var (
	gardenTestEnv *gardenenvtest.GardenerTestEnvironment

	configMapValidatingWebhookPath = "/configmap/validate"
)

type Environment struct {
	GardenEnv  *gardenenvtest.GardenerTestEnvironment
	K8sManager ctrl.Manager
	Config     *rest.Config
	K8sClient  client.Client
}

func New(validator admission.Handler) Environment {
	logf.SetLogger(zap.New(zap.WriteTo(ginkgo.GinkgoWriter), zap.UseDevMode(true)))

	ginkgo.By("bootstrapping test environment")

	failPolicy := admissionregistrationv1.Fail
	rules := []admissionregistrationv1.RuleWithOperations{
		{
			Operations: []admissionregistrationv1.OperationType{
				admissionregistrationv1.Create,
				admissionregistrationv1.Update,
			},
			Rule: admissionregistrationv1.Rule{
				APIGroups:   []string{""},
				APIVersions: []string{"v1"},
				Resources:   []string{"configmaps"},
			},
		},
	}

	webhookInstallOptions := envtest.WebhookInstallOptions{
		ValidatingWebhooks: []client.Object{
			&admissionregistrationv1.ValidatingWebhookConfiguration{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-validating-webhook-configuration",
				},
				TypeMeta: metav1.TypeMeta{
					Kind:       "ValidatingWebhookConfiguration",
					APIVersion: "admissionregistration.k8s.io/v1beta1",
				},
				Webhooks: []admissionregistrationv1.ValidatingWebhook{
					{
						Name:           "test-validating-create-update-gardenlogin.gardener.cloud",
						FailurePolicy:  &failPolicy,
						TimeoutSeconds: pointer.Int32Ptr(2),
						ClientConfig: admissionregistrationv1.WebhookClientConfig{
							Service: &admissionregistrationv1.ServiceReference{
								Path: &configMapValidatingWebhookPath,
							},
						},
						Rules: rules,
						ObjectSelector: &metav1.LabelSelector{
							MatchLabels: map[string]string{
								constants.GardenerOperationsRole: constants.GardenerOperationsKubeconfig,
							},
						},
					},
				},
			},
		},
	}

	var apiServerURL *url.URL

	apiServer := os.Getenv("ENVTEST_APISERVER_URL")
	if apiServer != "" {
		var err error
		apiServerURL, err = url.Parse(apiServer)
		gomega.Expect(err).NotTo(gomega.HaveOccurred())
	}

	ginkgo.By("bootstrapping test environment")

	gardenTestEnv = &gardenenvtest.GardenerTestEnvironment{
		Environment: &envtest.Environment{
			ControlPlane: envtest.ControlPlane{
				APIServer: &envtest.APIServer{
					URL: apiServerURL,
				},
			},
			WebhookInstallOptions: webhookInstallOptions,
		},
		GardenerAPIServer: &gardenenvtest.GardenerAPIServer{
			StopTimeout: 2 * time.Minute,
		},
	}

	cfg, err := gardenTestEnv.Start()
	gomega.Expect(err).NotTo(gomega.HaveOccurred())
	gomega.Expect(cfg).NotTo(gomega.BeNil())

	k8sClient, err := client.New(cfg, client.Options{Scheme: kubernetes.GardenScheme})
	gomega.Expect(err).NotTo(gomega.HaveOccurred())
	gomega.Expect(k8sClient).NotTo(gomega.BeNil())

	//+kubebuilder:scaffold:scheme

	ctrl.SetLogger(zap.New(zap.UseDevMode(true)))
	k8sManager, err := ctrl.NewManager(cfg, ctrl.Options{
		Scheme:             kubernetes.GardenScheme,
		LeaderElection:     false,
		Host:               gardenTestEnv.WebhookInstallOptions.LocalServingHost,
		Port:               gardenTestEnv.WebhookInstallOptions.LocalServingPort,
		CertDir:            gardenTestEnv.WebhookInstallOptions.LocalServingCertDir,
		MetricsBindAddress: "0", // disabled
	})
	gomega.Expect(err).ToNot(gomega.HaveOccurred())

	ginkgo.By("setting configuring webhook server")
	gomega.Expect(err).ToNot(gomega.HaveOccurred())

	hookServer := k8sManager.GetWebhookServer()
	hookServer.Register(configMapValidatingWebhookPath, &webhook.Admission{Handler: validator})

	return Environment{
		gardenTestEnv,
		k8sManager,
		cfg,
		k8sClient,
	}
}

func (e Environment) Start() {
	go func() {
		err := e.K8sManager.Start(ctrl.SetupSignalHandler())
		gomega.Expect(err).ToNot(gomega.HaveOccurred())
	}()

	ginkgo.By("waiting for webhook server to be ready")

	d := &net.Dialer{Timeout: time.Second}

	gomega.Eventually(func() error {
		serverURL := fmt.Sprintf("%s:%d", e.GardenEnv.WebhookInstallOptions.LocalServingHost, e.GardenEnv.WebhookInstallOptions.LocalServingPort)
		conn, err := tls.DialWithDialer(d, "tcp", serverURL, &tls.Config{
			InsecureSkipVerify: true,
		})
		if err != nil {
			return err
		}
		return conn.Close()
	}).Should(gomega.Succeed())
}

func DefaultConfiguration() *util.ControllerManagerConfiguration {
	return &util.ControllerManagerConfiguration{
		Controllers: util.ControllerManagerControllerConfiguration{
			Shoot: util.ShootControllerConfiguration{
				MaxConcurrentReconciles:             50,
				MaxConcurrentReconcilesPerNamespace: 3,
			},
		},
		Webhooks: util.ControllerManagerWebhookConfiguration{
			ConfigMapValidation: util.ConfigMapValidatingWebhookConfiguration{
				MaxObjectSize: 100 * 1024,
			},
		},
	}
}
