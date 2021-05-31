/*
SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors

SPDX-License-Identifier: Apache-2.0
*/

package controllers

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"sync"
	"time"

	"github.com/gardener/garden-login-controller-manager/api/v1alpha1"
	"github.com/gardener/garden-login-controller-manager/api/v1alpha1/constants"
	"github.com/gardener/garden-login-controller-manager/internal/util"

	gardencorev1alpha1 "github.com/gardener/gardener/pkg/apis/core/v1alpha1"
	gardencorev1beta1 "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	v1beta1constants "github.com/gardener/gardener/pkg/apis/core/v1beta1/constants"
	"github.com/gardener/gardener/pkg/utils/infodata"
	"github.com/gardener/gardener/pkg/utils/secrets"
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	kErros "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/json"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	clientcmdlatest "k8s.io/client-go/tools/clientcmd/api/latest"
	clientcmdv1 "k8s.io/client-go/tools/clientcmd/api/v1"
	"k8s.io/client-go/tools/record"
	"k8s.io/utils/pointer"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
)

// ShootStateReconciler reconciles a ShootState object
type ShootStateReconciler struct {
	Scheme *runtime.Scheme
	*ClientSet
	Recorder                    record.EventRecorder
	Log                         logr.Logger
	Config                      *util.ControllerManagerConfiguration
	ReconcilerCountPerNamespace map[string]int
	mutex                       sync.RWMutex
	configMutex                 sync.RWMutex
}

//+kubebuilder:rbac:groups=,resources=configmaps,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=,resources=configmaps/finalizers,verbs=update
//+kubebuilder:rbac:groups="core.gardener.cloud",resources=shootstate,verbs=get;list;watch;

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *ShootStateReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = r.Log.WithValues("shootstate", req.NamespacedName)

	if err := r.increaseCounterForNamespace(req.Namespace); err != nil {
		r.Log.Info("maximum parallel reconciles reached for namespace - requeuing the req", "namespace", req.Namespace, "name", req.Name)

		return ctrl.Result{
			RequeueAfter: wait.Jitter(time.Duration(int64(100*time.Millisecond)), 50), // requeue after 100ms - 5s
		}, nil
	}

	res, err := r.handleRequest(ctx, req)

	r.decreaseCounterForNamespace(req.Namespace)

	return res, err
}

// SetupWithManager sets up the controller with the Manager.
func (r *ShootStateReconciler) SetupWithManager(mgr ctrl.Manager, config util.ShootStateControllerConfiguration) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&gardencorev1alpha1.ShootState{}).
		Named("main").
		WithOptions(controller.Options{
			MaxConcurrentReconciles: config.MaxConcurrentReconciles,
		}).
		Complete(r)
}

func (r *ShootStateReconciler) handleRequest(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	kubeconfigName := fmt.Sprintf("%s.kubeconfig", req.Name)
	kubeconfigConfigMap := &corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{Name: kubeconfigName, Namespace: req.Namespace}}

	// fetch ShootState
	shootState := &gardencorev1alpha1.ShootState{}

	if err := r.Get(ctx, req.NamespacedName, shootState); err != nil {
		if kErros.IsNotFound(err) {
			// shootstate does not exist anymore - cleanup kubeconfig configmap
			return ctrl.Result{}, client.IgnoreNotFound(r.Client.Delete(ctx, kubeconfigConfigMap))
		}
		// Error reading the object - requeue the request
		return ctrl.Result{}, err
	}

	if shootState.DeletionTimestamp != nil {
		// shootstate is in deletion - cleanup kubeconfig configmap
		return ctrl.Result{}, client.IgnoreNotFound(r.Client.Delete(ctx, kubeconfigConfigMap))
	}

	// fetch Shoot
	shoot := &gardencorev1beta1.Shoot{}

	if err := r.Client.Get(ctx, req.NamespacedName, shoot); err != nil {
		if kErros.IsNotFound(err) {
			// shoot does not exist anymore - cleanup kubeconfig configmap
			return ctrl.Result{}, client.IgnoreNotFound(r.Client.Delete(ctx, kubeconfigConfigMap))
		}
		// Error reading the object - requeue the request
		return ctrl.Result{}, err
	}

	if shoot.DeletionTimestamp != nil {
		// shoot is in deletion - cleanup kubeconfig configmap
		return ctrl.Result{}, client.IgnoreNotFound(r.Client.Delete(ctx, kubeconfigConfigMap))
	}

	if len(shoot.Status.AdvertisedAddresses) == 0 {
		return ctrl.Result{}, errors.New("no kube-apiserver advertised addresses in Shoot .status.advertisedAddresses")
	}

	ca, err := infodata.GetInfoData(shootState.Spec.Gardener, v1beta1constants.SecretNameCACluster)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to get ca infoData: %w", err)
	}

	if ca == nil {
		return ctrl.Result{}, errors.New("certificate authority not yet provisioned")
	}

	caInfoData, ok := ca.(*secrets.CertificateInfoData)
	if !ok {
		return ctrl.Result{}, errors.New("could not convert InfoData entry ca to CertificateInfoData")
	}

	caCert := caInfoData.Certificate

	if err = util.ValidateCertificate(caCert); err != nil {
		return ctrl.Result{}, fmt.Errorf("an error occured validating the ca certificate: %w", err)
	}

	clusterIdentityConfigMap := &corev1.ConfigMap{}
	key := types.NamespacedName{
		Name:      v1beta1constants.ClusterIdentity,
		Namespace: "kube-system",
	}

	if err = r.Client.Get(ctx, key, clusterIdentityConfigMap); err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to fetch garden cluster identity: %w", err)
	}

	if clusterIdentityConfigMap.Data == nil {
		return ctrl.Result{}, errors.New("cluster identity configmap data not set")
	}

	kubeconfigRequest := kubeConfigRequest{
		namespace:             shoot.Namespace,
		shootName:             shoot.Name,
		gardenClusterIdentity: clusterIdentityConfigMap.Data[v1beta1constants.ClusterIdentity],
	}

	for _, address := range shoot.Status.AdvertisedAddresses {
		u, err := url.Parse(address.URL)
		if err != nil {
			return ctrl.Result{}, fmt.Errorf("could not parse shoot server url: %w", err)
		}

		kubeconfigRequest.clusters = append(kubeconfigRequest.clusters, cluster{
			name:          address.Name,
			apiServerHost: u.Host,
			caCert:        caCert,
		})
	}

	if err = kubeconfigRequest.validate(); err != nil {
		return ctrl.Result{}, fmt.Errorf("validation failed for kubeconfig request: %w", err)
	}

	kubeconfig, err := kubeconfigRequest.generate()
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("generation failed for kubeconfig request: %w", err)
	}

	ownerReference := metav1.NewControllerRef(shoot, gardencorev1beta1.SchemeGroupVersion.WithKind("Shoot"))
	ownerReference.BlockOwnerDeletion = pointer.BoolPtr(false)

	// store the kubeconfig in a ConfigMap, as it does not contain any credentials or other secret data
	if _, err = ctrl.CreateOrUpdate(ctx, r.ClientSet, kubeconfigConfigMap, func() error {
		kubeconfigConfigMap.OwnerReferences = []metav1.OwnerReference{*ownerReference}

		if kubeconfigConfigMap.Labels == nil {
			kubeconfigConfigMap.Labels = make(map[string]string)
		}
		kubeconfigConfigMap.Labels[constants.GardenerOperationsRole] = constants.GardenerOperationsKubeconfig

		if kubeconfigConfigMap.Data == nil {
			kubeconfigConfigMap.Data = make(map[string]string)
		}
		kubeconfigConfigMap.Data[constants.DataKeyKubeconfig] = string(kubeconfig)
		return nil
	}); err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to create or update kubeconfig configmap %shootState/%shootState: %w", kubeconfigConfigMap.Namespace, kubeconfigConfigMap.Name, err)
	}

	return ctrl.Result{}, nil
}

// kubeConfigRequest is a struct which holds information about a Kubeconfig to be generated.
type kubeConfigRequest struct {
	// cluster holds all the cluster on which the kube-apiserver can be reached
	clusters []cluster
	// namespace is the namespace where the shoot resides
	namespace string
	// shootName is the name of the shoot
	shootName string
	// gardenClusterIdentity is the cluster identifier of the garden cluster.
	gardenClusterIdentity string
}

// cluster holds the data to describe and connect to a kuberentes cluster
type cluster struct {
	// name is the name of the shoot advertised address, usually "external", "internal" or "unmanaged"
	name string
	// apiServerHost is the host of the kube-apiserver
	apiServerHost string

	// caCert holds the ca certificate for the cluster
	//+optional
	caCert []byte
}

// validate validates the kubeconfig request by ensuring that all required fields are set
func (k *kubeConfigRequest) validate() error {
	if len(k.clusters) == 0 {
		return errors.New("missing clusters")
	}

	for n, cluster := range k.clusters {
		if cluster.name == "" {
			return fmt.Errorf("no name defined for cluster[%d]", n)
		}

		if cluster.apiServerHost == "" {
			return fmt.Errorf("no api server host defined for cluster[%d]", n)
		}
	}

	if k.namespace == "" {
		return errors.New("no namespace defined for kubeconfig request")
	}

	if k.shootName == "" {
		return errors.New("no shoot name defined for kubeconfig request")
	}

	if k.gardenClusterIdentity == "" {
		return errors.New("no garden cluster identity defined for kubeconfig request")
	}

	return nil
}

// generate generates a Kubernetes kubeconfig for communicating with the kube-apiserver by using
// a client certificate. If <basicAuthUser> and <basicAuthPass> are non-empty string, a second user object
// containing the Basic Authentication credentials is added to the Kubeconfig.
func (k *kubeConfigRequest) generate() ([]byte, error) {
	authName := fmt.Sprintf("%s--%s", k.namespace, k.shootName)
	name := fmt.Sprintf("%s-%s", authName, k.clusters[0].name)

	var authInfos []clientcmdv1.NamedAuthInfo
	authInfos = append(authInfos, clientcmdv1.NamedAuthInfo{
		Name: authName,
		AuthInfo: clientcmdv1.AuthInfo{
			Exec: &clientcmdv1.ExecConfig{
				Command: "kubectl",
				Args: []string{
					"garden-login",
					"get-client-certificate",
				},
				Env:                nil,
				APIVersion:         clientcmdv1.SchemeGroupVersion.String(),
				InstallHint:        "",
				ProvideClusterInfo: true,
			},
		},
	})

	config := &clientcmdv1.Config{
		CurrentContext: name,
		Clusters:       []clientcmdv1.NamedCluster{},
		Contexts:       []clientcmdv1.NamedContext{},
		AuthInfos:      authInfos,
	}

	extension := v1alpha1.ExecPluginConfig{
		ShootRef: v1alpha1.ShootRef{
			Namespace: k.namespace,
			Name:      k.shootName,
		},
		GardenClusterIdentity: k.gardenClusterIdentity,
	}

	raw, err := json.Marshal(extension)
	if err != nil {
		return nil, fmt.Errorf("could not json marshal cluster extension: %w", err)
	}

	for _, cluster := range k.clusters {
		name := fmt.Sprintf("%s-%s", authName, cluster.name)

		config.Clusters = append(config.Clusters, clientcmdv1.NamedCluster{
			Name: name,
			Cluster: clientcmdv1.Cluster{
				CertificateAuthorityData: cluster.caCert,
				Server:                   fmt.Sprintf("https://%s", cluster.apiServerHost),
				Extensions: []clientcmdv1.NamedExtension{
					{
						Name: "client.authentication.k8s.io/exec",
						Extension: runtime.RawExtension{
							Raw: raw,
						},
					},
				},
			},
		})
		config.Contexts = append(config.Contexts, clientcmdv1.NamedContext{
			Name: name,
			Context: clientcmdv1.Context{
				Cluster:  name,
				AuthInfo: authName,
			},
		})
	}

	return runtime.Encode(clientcmdlatest.Codec, config)
}

// getConfig returns the util.ControllerManagerConfiguration of the ShootStateReconciler
func (r *ShootStateReconciler) getConfig() *util.ControllerManagerConfiguration {
	r.configMutex.RLock()
	defer r.configMutex.RUnlock()

	return r.Config
}

//// injectConfig is mainly used for tests to inject util.ControllerManagerConfiguration configuration
//func (r *ShootStateReconciler) injectConfig(config *util.ControllerManagerConfiguration) {
//	r.configMutex.Lock()
//	defer r.configMutex.Unlock()
//
//	r.Config = config
//}

func (r *ShootStateReconciler) increaseCounterForNamespace(namespace string) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	var counter int
	if c, exists := r.ReconcilerCountPerNamespace[namespace]; !exists {
		counter = 1
	} else {
		counter = c + 1
	}

	if counter > r.getConfig().Controllers.ShootState.MaxConcurrentReconcilesPerNamespace {
		return fmt.Errorf("max count reached")
	}

	r.ReconcilerCountPerNamespace[namespace] = counter

	return nil
}

func (r *ShootStateReconciler) decreaseCounterForNamespace(namespace string) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	var counter int

	c, exists := r.ReconcilerCountPerNamespace[namespace]
	if !exists {
		panic("entry expected!")
	}

	counter = c - 1
	if counter == 0 {
		delete(r.ReconcilerCountPerNamespace, namespace)
	} else {
		r.ReconcilerCountPerNamespace[namespace] = counter
	}
}

type ClientSet struct {
	*rest.Config
	client.Client
	Kubernetes kubernetes.Interface
}

func NewClientSet(config *rest.Config, client client.Client, kubernetes kubernetes.Interface) *ClientSet {
	return &ClientSet{config, client, kubernetes}
}
