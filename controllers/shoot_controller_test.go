/*
SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors

SPDX-License-Identifier: Apache-2.0
*/

package controllers

import (
	"encoding/json"
	"fmt"
	"github.com/gardener/gardener/pkg/utils"
	"time"

	gardencorev1alpha1 "github.com/gardener/gardener/pkg/apis/core/v1alpha1"
	gardencorev1beta1 "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	corev1beta1constants "github.com/gardener/gardener/pkg/apis/core/v1beta1/constants"
	"github.com/gardener/gardener/pkg/utils/gardener"
	"github.com/gardener/gardener/pkg/utils/secrets"
	"github.com/gardener/gardener/pkg/utils/test/matchers"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/utils/pointer"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/gardener/gardenlogin-controller-manager/api/v1alpha1"
	"github.com/gardener/gardenlogin-controller-manager/api/v1alpha1/constants"
	"github.com/gardener/gardenlogin-controller-manager/internal/test"
)

var _ = Describe("ShootController", func() {

	const (
		infrastructureType = "foo-infra"
		cloudProfile       = "cloudprofile-" + infrastructureType
		machineType        = "foo-machine-type"
		networkType        = "foo-network"
		osType             = "foo-os"
		region             = "foo-region"
		version            = "1.0.0"
		k8sVersion         = "1.20.0"
		k8sVersionLegacy   = "1.19.0" // legacy kubeconfig should be rendered
		sharedNs           = "shared-ns"
		sharedInfraSecret  = "shared-infra-secret"

		timeout  = time.Second * 10
		interval = time.Millisecond * 250
	)
	BeforeEach(func() {
		cmConfig = test.DefaultConfiguration()
		shootReconciler.injectConfig(cmConfig)

		By("ensuring that required resources for shoot and shootstate exist")
		Expect(k8sClient.Create(ctx, &gardencorev1beta1.ControllerRegistration{
			ObjectMeta: metav1.ObjectMeta{Name: "foo"},
			Spec: gardencorev1beta1.ControllerRegistrationSpec{
				Resources: []gardencorev1beta1.ControllerResource{
					{
						Kind:    "Network",
						Type:    networkType,
						Primary: pointer.BoolPtr(true),
					},
					{
						Kind:    "OperatingSystemConfig",
						Type:    osType,
						Primary: pointer.BoolPtr(true),
					},
					{
						Kind:    "Infrastructure",
						Type:    infrastructureType,
						Primary: pointer.BoolPtr(true),
					},
					{
						Kind:    "ControlPlane",
						Type:    infrastructureType,
						Primary: pointer.BoolPtr(true),
					},
					{
						Kind:    "Worker",
						Type:    infrastructureType,
						Primary: pointer.BoolPtr(true),
					},
				},
			},
		})).
			To(Or(Succeed(), matchers.BeAlreadyExistsError()))

		Expect(k8sClient.Create(ctx, &gardencorev1beta1.CloudProfile{
			ObjectMeta: metav1.ObjectMeta{Name: cloudProfile},
			Spec: gardencorev1beta1.CloudProfileSpec{
				CABundle: nil,
				Kubernetes: gardencorev1beta1.KubernetesSettings{
					Versions: []gardencorev1beta1.ExpirableVersion{
						{Version: k8sVersion},
						{Version: k8sVersionLegacy},
					},
				},
				MachineImages: []gardencorev1beta1.MachineImage{
					{
						Name: osType,
						Versions: []gardencorev1beta1.MachineImageVersion{
							{
								ExpirableVersion: gardencorev1beta1.ExpirableVersion{
									Version: version,
								},
							},
						},
					},
				},
				MachineTypes: []gardencorev1beta1.MachineType{
					{
						CPU:    resource.MustParse("42"),
						Memory: resource.MustParse("42"),
						Name:   machineType,
						Usable: pointer.BoolPtr(true),
					},
				},
				Regions: []gardencorev1beta1.Region{
					{
						Name: region,
					},
				},
				Type: infrastructureType,
			},
		})).
			To(Or(Succeed(), matchers.BeAlreadyExistsError()))

		Expect(k8sClient.Create(ctx, &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: sharedNs}})).To(Or(Succeed(), matchers.BeAlreadyExistsError()))
		Expect(k8sClient.Create(ctx, &corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: sharedInfraSecret, Namespace: sharedNs}})).To(Or(Succeed(), matchers.BeAlreadyExistsError()))
	})
	Describe("reconcile shoot", func() {
		var (
			namespace    string
			configMapKey types.NamespacedName
			ca           *secrets.Certificate

			shoot               *gardencorev1beta1.Shoot
			shootState          *gardencorev1alpha1.ShootState
			advertisedAddresses []gardencorev1beta1.ShootAdvertisedAddress

			hardQuota         string
			usedQuota         string
			resourceQuota     *corev1.ResourceQuota
			withResourceQuota bool
		)

		const (
			name          = "foo-shoot"
			secretBinding = "foo-secret-binding"
			domain        = "foo.bar.baz"
		)

		BeforeEach(func() {
			suffix := test.StringWithCharset(randomLength, charset)

			namespace = "garden-" + suffix
			Expect(k8sClient.Create(ctx, &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{
				Name: namespace,
				Labels: map[string]string{
					corev1beta1constants.GardenRole: corev1beta1constants.GardenRoleProject,
				},
			}})).To(Succeed())

			Expect(k8sClient.Create(ctx, &gardencorev1beta1.Project{
				ObjectMeta: metav1.ObjectMeta{Name: "p" + suffix},
				Spec: gardencorev1beta1.ProjectSpec{
					Namespace: pointer.StringPtr(namespace),
				},
			})).To(Succeed())

			Expect(k8sClient.Create(ctx, &gardencorev1beta1.SecretBinding{
				ObjectMeta: metav1.ObjectMeta{
					Name:      secretBinding,
					Namespace: namespace,
				},
				SecretRef: corev1.SecretReference{
					Name:      sharedInfraSecret,
					Namespace: sharedNs,
				},
			})).To(Succeed())

			shoot = &gardencorev1beta1.Shoot{
				ObjectMeta: metav1.ObjectMeta{
					Name:      name,
					Namespace: namespace,
				},
				Spec: gardencorev1beta1.ShootSpec{
					CloudProfileName: cloudProfile,
					DNS: &gardencorev1beta1.DNS{
						Domain: pointer.StringPtr(domain),
					},
					Kubernetes: gardencorev1beta1.Kubernetes{
						Version: k8sVersion,
					},
					Networking: gardencorev1beta1.Networking{
						Type: networkType,
					},
					Provider: gardencorev1beta1.Provider{
						Type:                 infrastructureType,
						ControlPlaneConfig:   nil,
						InfrastructureConfig: nil,
						Workers: []gardencorev1beta1.Worker{
							{
								Name: "foo-worker",
								Machine: gardencorev1beta1.Machine{
									Type: machineType,
									Image: &gardencorev1beta1.ShootMachineImage{
										Name:    osType,
										Version: pointer.StringPtr(version),
									},
								},
								Maximum: 1,
								Minimum: 1,
							},
						},
					},
					Region:            region,
					SecretBindingName: secretBinding,
				},
			}

			advertisedAddresses = []gardencorev1beta1.ShootAdvertisedAddress{
				{
					Name: "shoot-address1",
					URL:  "https://api." + domain,
				},
				{
					Name: "shoot-address2",
					URL:  "https://api2." + domain,
				},
			}

			ca = generateCaCert()
			caRaw := []byte(`{"ca.crt":"` + utils.EncodeBase64(ca.CertificatePEM) + `"}`)

			shootState = &gardencorev1alpha1.ShootState{
				ObjectMeta: metav1.ObjectMeta{
					Name:      shoot.Name,
					Namespace: namespace,
				},
				Spec: gardencorev1alpha1.ShootStateSpec{
					Gardener: []gardencorev1alpha1.GardenerResourceData{
						{
							Name: corev1beta1constants.SecretNameCACluster,
							Type: "secret",
							Data: runtime.RawExtension{Raw: caRaw},
						},
					},
				},
			}

			configMapKey = types.NamespacedName{
				Namespace: namespace,
				Name:      shoot.Name + ".kubeconfig",
			}

			resourceQuota = &corev1.ResourceQuota{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "gardener",
					Namespace: namespace,
				},
			}
			withResourceQuota = false
		})

		JustBeforeEach(func() {
			if withResourceQuota {
				By("creating resourceQuota")
				resourceQuota.Spec.Hard = corev1.ResourceList{
					"count/configmaps": resource.MustParse(hardQuota),
				}
				Expect(k8sClient.Create(ctx, resourceQuota)).To(Succeed())

				By("patching resource quota status")
				resourceQuotaCopy := resourceQuota.DeepCopy()
				resourceQuota.Status.Used = corev1.ResourceList{
					"count/configmaps": resource.MustParse(usedQuota),
				}
				resourceQuota.Status.Hard = corev1.ResourceList{
					"count/configmaps": resource.MustParse(hardQuota),
				}
				Expect(k8sClient.Status().Patch(ctx, resourceQuota, client.MergeFrom(resourceQuotaCopy))).To(Succeed())
			}

			By("creating shoot")
			Expect(k8sClient.Create(ctx, shoot)).To(Succeed())

			By("patching shoot status")
			shootCopy := shoot.DeepCopy()
			shoot.Status.AdvertisedAddresses = advertisedAddresses
			Expect(k8sClient.Status().Patch(ctx, shoot, client.MergeFrom(shootCopy))).To(Succeed())

			By("creating shootstate")
			Expect(k8sClient.Create(ctx, shootState)).To(Succeed())
		})

		It("should create kubeconfig configMap", func() {
			shoot.Spec.Kubernetes.Version = k8sVersion

			var kubeconfig string
			Eventually(func() bool {
				configMap := &corev1.ConfigMap{}
				err := k8sClient.Get(ctx, configMapKey, configMap)
				if err != nil {
					return false
				}

				kubeconfig = configMap.Data[constants.DataKeyKubeconfig]
				return kubeconfig != ""
			}, timeout, interval).Should(BeTrue())

			clientConfig, err := clientcmd.NewClientConfigFromBytes([]byte(kubeconfig))
			Expect(err).ToNot(HaveOccurred())

			rawConfig, err := clientConfig.RawConfig()
			Expect(err).ToNot(HaveOccurred())

			Expect(rawConfig.Clusters).To(HaveLen(2))
			currentCluster := rawConfig.Contexts[rawConfig.CurrentContext].Cluster
			Expect(rawConfig.Clusters[currentCluster].Server).To(Equal("https://api." + domain))
			Expect(rawConfig.Clusters[currentCluster].CertificateAuthorityData).To(Equal(ca.CertificatePEM))
			Expect(rawConfig.Clusters[currentCluster].Extensions).ToNot(BeEmpty())

			execConfig := rawConfig.Clusters[currentCluster].Extensions["client.authentication.k8s.io/exec"].(*runtime.Unknown)
			Expect(execConfig.Raw).ToNot(BeNil())

			var extension v1alpha1.ExecPluginConfig
			Expect(json.Unmarshal(execConfig.Raw, &extension)).To(Succeed())
			Expect(extension).To(Equal(v1alpha1.ExecPluginConfig{
				ShootRef: v1alpha1.ShootRef{
					Namespace: namespace,
					Name:      name,
				},
				GardenClusterIdentity: "envtest",
			}))

			Expect(rawConfig.Contexts).To(HaveLen(2))

			Expect(rawConfig.AuthInfos).To(HaveLen(1))
			currentAuthInfo := rawConfig.Contexts[rawConfig.CurrentContext].AuthInfo
			Expect(rawConfig.AuthInfos[currentAuthInfo].Exec.Command).To(Equal("kubectl"))
			Expect(rawConfig.AuthInfos[currentAuthInfo].Exec.Args).To(Equal([]string{
				"gardenlogin",
				"get-client-certificate",
			}))
		})

		It("should restore kubeconfig configMap", func() {
			shoot.Spec.Kubernetes.Version = k8sVersion

			configMap := &corev1.ConfigMap{}
			Eventually(func() error {
				return k8sClient.Get(ctx, configMapKey, configMap)
			}).Should(Succeed())

			By("changing the kubeconfig")
			configMap.Data[constants.DataKeyKubeconfig] = "foo-kubeconfig"
			Expect(k8sClient.Update(ctx, configMap)).To(Succeed())

			By("verifying that kubeconfig is restored")
			Eventually(func() string {
				err := k8sClient.Get(ctx, configMapKey, configMap)
				if err != nil {
					return ""
				}

				kubeconfig := configMap.Data[constants.DataKeyKubeconfig]
				return kubeconfig
			}, timeout, interval).ShouldNot(Equal("foo-kubeconfig"))
		})

		It("should delete kubeconfig configMap", func() {
			By("deleting shoot")
			shoot := &gardencorev1beta1.Shoot{
				ObjectMeta: metav1.ObjectMeta{
					Name:      name,
					Namespace: namespace,
				},
			}
			shootCopy := shoot.DeepCopy()
			shoot.Annotations = map[string]string{
				gardener.ConfirmationDeletion: "true",
			}
			Expect(k8sClient.Patch(ctx, shoot, client.MergeFrom(shootCopy))).To(Succeed())
			Expect(k8sClient.Delete(ctx, shoot)).To(Succeed())

			By("ensuring shoot is deleted")
			Eventually(func() bool {
				err := k8sClient.Get(ctx, types.NamespacedName{Name: name, Namespace: namespace}, &gardencorev1beta1.Shoot{})
				return apierrors.IsNotFound(err)
			}, timeout, interval).Should(BeTrue())

			By("verifying configMap is deleted")
			Eventually(func() bool {
				err := k8sClient.Get(ctx, configMapKey, &corev1.ConfigMap{})
				return apierrors.IsNotFound(err)
			}, timeout, interval).Should(BeTrue())
		})

		It("should not delete kubeconfig configMap when shoot deletion timestamp is set", func() {
			By("deleting shoot")
			shoot := &gardencorev1beta1.Shoot{
				ObjectMeta: metav1.ObjectMeta{
					Name:      name,
					Namespace: namespace,
				},
			}
			shootCopy := shoot.DeepCopy()
			shoot.Finalizers = append(shoot.Finalizers, "envtest") // add dummy finalizer to ensure that the resource is not removed
			shoot.Annotations = map[string]string{
				gardener.ConfirmationDeletion: "true",
			}
			Expect(k8sClient.Patch(ctx, shoot, client.MergeFrom(shootCopy))).To(Succeed())
			Expect(k8sClient.Delete(ctx, shoot)).To(Succeed())

			By("ensuring shoot is not deleted")
			Consistently(func() bool {
				err := k8sClient.Get(ctx, types.NamespacedName{Name: name, Namespace: namespace}, &gardencorev1beta1.Shoot{})
				return err == nil
			}, timeout, interval).Should(BeTrue())

			By("ensuring configMap is not deleted")
			Consistently(func() bool {
				err := k8sClient.Get(ctx, configMapKey, &corev1.ConfigMap{})
				return err == nil
			}, timeout, interval).Should(BeTrue())
		})

		It("should not delete kubeconfig configMap when shootState deletion timestamp is set", func() {
			By("deleting shoot")
			shootState := &gardencorev1alpha1.ShootState{
				ObjectMeta: metav1.ObjectMeta{
					Name:      name,
					Namespace: namespace,
				},
			}
			shootStateCopy := shootState.DeepCopy()
			shootState.Finalizers = append(shootState.Finalizers, "envtest") // add dummy finalizer to ensure that the resource is not removed
			shootState.Annotations = map[string]string{
				gardener.ConfirmationDeletion: "true",
			}
			Expect(k8sClient.Patch(ctx, shootState, client.MergeFrom(shootStateCopy))).To(Succeed())
			Expect(k8sClient.Delete(ctx, shootState)).To(Succeed())

			By("ensuring shootState is not deleted")
			Consistently(func() bool {
				err := k8sClient.Get(ctx, types.NamespacedName{Name: name, Namespace: namespace}, &gardencorev1alpha1.ShootState{})
				return err == nil
			}, timeout, interval).Should(BeTrue())

			By("ensuring configMap is not deleted")
			Consistently(func() bool {
				err := k8sClient.Get(ctx, configMapKey, &corev1.ConfigMap{})
				return err == nil
			}, timeout, interval).Should(BeTrue())
		})

		Describe("resource quota", func() {

			BeforeEach(func() {
				hardQuota = "2"
				usedQuota = "2"
				withResourceQuota = true

				cmConfig.Controllers.Shoot.QuotaExceededRetryDelay = 50 * time.Millisecond
				shootReconciler.injectConfig(cmConfig)
			})

			It("should create kubeconfig configMap after quota increase", func() {
				By("verifying that the configMap is not created")
				Consistently(func() bool {
					configMap := &corev1.ConfigMap{}
					err := k8sClient.Get(ctx, configMapKey, configMap)
					return apierrors.IsNotFound(err)
				}).Should(BeTrue())

				By("increasing resource quota")
				resourceQuotaCopy := resourceQuota.DeepCopy()
				resourceQuota.Spec.Hard = corev1.ResourceList{
					"count/configmaps": resource.MustParse("3"),
				}
				Expect(k8sClient.Patch(ctx, resourceQuota, client.MergeFrom(resourceQuotaCopy))).To(Succeed())
				resourceQuotaCopy = resourceQuota.DeepCopy()
				resourceQuota.Status.Hard = corev1.ResourceList{
					"count/configmaps": resource.MustParse("3"),
				}
				Expect(k8sClient.Status().Patch(ctx, resourceQuota, client.MergeFrom(resourceQuotaCopy))).To(Succeed())

				By("validating that a configMap containing a kubeconfig data key is created")
				Eventually(func() bool {
					configMap := &corev1.ConfigMap{}
					if err := k8sClient.Get(ctx, configMapKey, configMap); err != nil {
						return false
					}

					kubeconfig := configMap.Data[constants.DataKeyKubeconfig]
					return kubeconfig != ""
				}).Should(BeTrue())
			})

			It("should create kubeconfig configMap after usage freed", func() {
				By("verifying that the configMap is not created")
				isConfigMapNotFound := func() bool {
					configMap := &corev1.ConfigMap{}
					if err := k8sClient.Get(ctx, configMapKey, configMap); err != nil {
						return apierrors.IsNotFound(err)
					}

					return false
				}
				Consistently(isConfigMapNotFound).Should(BeTrue())

				By("lowering configMap usage")
				resourceQuotaCopy := resourceQuota.DeepCopy()
				Expect(k8sClient.Patch(ctx, resourceQuota, client.MergeFrom(resourceQuotaCopy))).To(Succeed())
				resourceQuotaCopy = resourceQuota.DeepCopy()
				resourceQuota.Status.Used = corev1.ResourceList{
					"count/configmaps": resource.MustParse("1"),
				}
				Expect(k8sClient.Status().Patch(ctx, resourceQuota, client.MergeFrom(resourceQuotaCopy))).To(Succeed())

				By("validating that a configMap containing a kubeconfig data key is created")
				Eventually(func() bool {
					configMap := &corev1.ConfigMap{}
					if err := k8sClient.Get(ctx, configMapKey, configMap); err != nil {
						return false
					}

					kubeconfig := configMap.Data[constants.DataKeyKubeconfig]
					return kubeconfig != ""
				}).Should(BeTrue())
			})
		})

		Context("legacy kubeconfig", func() {
			BeforeEach(func() {
				By("having shoot kubernetes version < v1.20.0")
				shoot.Spec.Kubernetes.Version = k8sVersionLegacy
			})

			It("should create legacy kubeconfig configMap", func() {
				var kubeconfig string
				Eventually(func() bool {
					configMap := &corev1.ConfigMap{}
					err := k8sClient.Get(ctx, configMapKey, configMap)
					if err != nil {
						return false
					}

					kubeconfig = configMap.Data[constants.DataKeyKubeconfig]
					return kubeconfig != ""
				}, timeout, interval).Should(BeTrue())

				clientConfig, err := clientcmd.NewClientConfigFromBytes([]byte(kubeconfig))
				Expect(err).ToNot(HaveOccurred())

				rawConfig, err := clientConfig.RawConfig()
				Expect(err).ToNot(HaveOccurred())

				Expect(rawConfig.Clusters).To(HaveLen(2))
				currentCluster := rawConfig.Contexts[rawConfig.CurrentContext].Cluster
				Expect(rawConfig.Clusters[currentCluster].Server).To(Equal("https://api." + domain))
				Expect(rawConfig.Clusters[currentCluster].CertificateAuthorityData).To(Equal(ca.CertificatePEM))
				Expect(rawConfig.Clusters[currentCluster].Extensions).To(BeEmpty())

				Expect(rawConfig.Contexts).To(HaveLen(2))

				Expect(rawConfig.AuthInfos).To(HaveLen(1))
				currentAuthInfo := rawConfig.Contexts[rawConfig.CurrentContext].AuthInfo
				Expect(rawConfig.AuthInfos[currentAuthInfo].Exec.Command).To(Equal("kubectl"))
				Expect(rawConfig.AuthInfos[currentAuthInfo].Exec.Args).To(Equal([]string{
					"gardenlogin",
					"get-client-certificate",
					"--name=foo-shoot",
					fmt.Sprintf("--namespace=%s", namespace),
					"--garden-cluster-identity=envtest",
				}))
			})
		})
	})

})

func generateCaCert() *secrets.Certificate {
	csc := &secrets.CertificateSecretConfig{
		Name:       "ca-test",
		CommonName: "ca-test",
		CertType:   secrets.CACert,
	}
	caCertificate, err := csc.GenerateCertificate()
	Expect(err).ToNot(HaveOccurred())

	return caCertificate
}
